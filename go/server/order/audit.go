package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	"order_server/go/server/order/ai_model"
	alog "order_server/go/server/log"
	"order_server/go/utils/httputils"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// 本文件实现预览审核（契约 §3.2 / 架构 §5）：
// 程序对比（对 global.LastInputText 做确定性正则解析得到基准订单，架构 §5.2 方案 C）
// + AI 复核（仅对可疑项批量调用一次，失败不阻塞响应）。
//
// B6/缺陷2（2026-10-17 修复）：审核全程只读 global.AddressList / global.LastInputText，不修改两者；
// 识别在途期间（recordLastInputText 已先行更新 LastInputText、AddressList 待 AI 返回后才写入/写完）
// 两者处于半更新状态，直接审核会把未落定的新增批判为 missing 误报。
// 一致性由 global.RecognitionInFlight 在途计数保证：识别入口（enter.go OrderRec/StreamOrderRec）
// 进入 +1、退出 -1，本文件 AuditPreviewData 入口检测在途即拒绝（50000 业务错误）。
// 选择标志位而非 RWMutex 的理由：锁方案要么粒度细（写点逐个加锁，流式分支仍存在
// LastInputText 与 AddressList 跨秒级不一致窗口，修不掉误报），要么持锁横跨整个外部
// AI 调用（引入长时间持锁与死锁面）；在途标志精准覆盖「不一致窗口」本身且零锁风险。

// AuditRequest 预览审核入参（契约 §3.2：空 JSON 对象，无必填字段；
// 原始文本由后端 global.LastInputText 暂存，请求不回传文本）。
type AuditRequest struct{}

// AuditItem 审核差异项（契约 §3.2）。订单业务字段中文键（与 ExceladdressInfo 逐字一致，
// 前端可复用行渲染），审核域字段英文小驼峰（VID/diffType/diffDesc/aiVerdict/aiSuggest）。
type AuditItem struct {
	VID          uint   `json:"VID"`        // 预览行标识（与 ExceladdressInfo.VID 一致；缺单项为 0）
	Recipient    string `json:"收件人"`      // 缺单/错单时为原始文本侧值，其余为预览侧值
	Phone        string `json:"手机号"`
	Address      string `json:"收货地址"`
	ProductSpecs string `json:"商品规格"`    // 展示串（ProductSpecsString 格式）
	DiffType     string `json:"diffType"`   // 枚举见下，值冻结（契约 §8.1）
	DiffDesc     string `json:"diffDesc"`   // 程序对比差异描述（中文，normal 为空串）
	AiVerdict    string `json:"aiVerdict"`  // "真差异" | "误报" | ""（未调 AI 或 AI 不可用）
	AiSuggest    string `json:"aiSuggest"`  // AI 修正建议，无则 ""
}

// AuditSummary 审核汇总计数（契约 §3.2）。
type AuditSummary struct {
	Missing int `json:"missing"` // 缺单
	Wrong   int `json:"wrong"`   // 错单（wrong-key + wrong-spec 合计）
	Extra   int `json:"extra"`   // 多单
	Normal  int `json:"normal"`  // 正常
}

// AuditResponse 预览审核出参（契约 §3.2）。
type AuditResponse struct {
	Items    []AuditItem  `json:"items"`    // 按 missing→wrong-key→wrong-spec→extra→normal 排序，同级按 VID 升序
	Summary  AuditSummary `json:"summary"`
	AiFailed bool         `json:"aiFailed"` // AI 复核环节异常时 true（此时 aiVerdict/aiSuggest 全空，程序对比结果仍有效）
}

// diffType 枚举（契约 §3.2，字符串枚举不入 models/enum 的 iota 体系；值冻结，新增只能尾加）
const (
	diffTypeMissing   = "missing"   // 缺单：原始文本有、预览表无
	diffTypeWrongKey  = "wrong-key" // 错单-关键字段：收件人/手机号/地址不一致
	diffTypeWrongSpec = "wrong-spec" // 错单-规格：规格数量或内容不一致
	diffTypeExtra     = "extra"     // 多单：预览表有、原始文本无
	diffTypeNormal    = "normal"   // 一致
)

// ===== 程序对比：原始文本确定性解析 =====

var (
	// 11 位手机号（1 开头第 2 位 3-9），与 systemPrompt「提取 11 位手机号码」规则对应
	auditPhoneRe = regexp.MustCompile(`1[3-9]\d{9}`)
	// 商品规格：【名称数字斤】（与 ordertutil.Specprocessing 同正则，保证基准与预览同解析口径）
	auditSpecRe = regexp.MustCompile(`【([^】]+?)(\d+)(?:斤)?】`)
	// 行首「3-4」类日期/序号标记（systemPrompt：以「3-4」或「店名：」作为一条新记录的开始）
	auditMarkerRe = regexp.MustCompile(`^\d{1,2}-\d{1,2}`)
)

// baselineOrder 程序解析原始文本得到的基准订单（架构 §5.2）。
type baselineOrder struct {
	Recipient string
	Phone     string
	Address   string
	Specs     []models.ProductSpec
}

// parseBaselineOrders 对原始识别文本做确定性正则解析（架构 §5.1 方案 C）。
// 记录切分规则（参考 systemPrompt.txt 的输入示例格式）：
//  1. 行首形如「3-4」的日期/序号标记、或含「店名：」的行 → 新记录开始；
//  2. 当前行含 11 位手机号且当前记录已有手机号 → 新记录开始
//     （兼容「张三， 13800138003\n地址\n【规格】」类无标记紧凑格式——每条订单有且仅有一个手机号）。
// 字段提取：「店名：/地址：」显式前缀优先；无前缀时手机号所在行的号码前缀段为收件人、
// 其后首个非「【规格】」行为收货地址；规格用与 Specprocessing 相同的正则逐项提取。
// 已知限制（架构 §5.1 已接受）：正则解析对非标准格式容错有限（如同行并列多条订单、
// 地址内嵌 11 位数字被误切分等），误报由 AI 复核与人工确认兜底。
func parseBaselineOrders(text string) []baselineOrder {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	var records [][]string
	var cur []string
	curHasPhone := false
	flush := func() {
		if len(cur) > 0 {
			records = append(records, cur)
		}
		cur = nil
		curHasPhone = false
	}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if (auditMarkerRe.MatchString(line) || strings.Contains(line, "店名：")) ||
			(curHasPhone && auditPhoneRe.MatchString(line)) {
			if len(cur) > 0 {
				flush()
			}
		}
		cur = append(cur, line)
		if auditPhoneRe.MatchString(line) {
			curHasPhone = true
		}
	}
	flush()

	orders := make([]baselineOrder, 0, len(records))
	for _, rec := range records {
		o := extractBaselineOrder(rec)
		// 全字段为空的孤立噪声片段（如单独一行标记）不是订单，跳过
		if o.Phone == "" && o.Recipient == "" && o.Address == "" && len(o.Specs) == 0 {
			continue
		}
		orders = append(orders, o)
	}
	return orders
}

// extractBaselineOrder 从单条记录的行列表中提取订单字段。
func extractBaselineOrder(rec []string) baselineOrder {
	var o baselineOrder
	recText := strings.Join(rec, "\n")

	// 手机号：记录内第一个 11 位号码
	o.Phone = auditPhoneRe.FindString(recText)

	// 规格：与 Specprocessing 同正则
	for _, m := range auditSpecRe.FindAllStringSubmatch(recText, -1) {
		name := strings.TrimSpace(m[1])
		weight, err := strconv.Atoi(m[2])
		if err != nil || name == "" {
			continue
		}
		o.Specs = append(o.Specs, models.ProductSpec{Name: name, Weight: weight})
	}

	// 收件人 / 收货地址：显式前缀优先，紧凑格式兜底
	phoneLineIdx := -1
	for idx, line := range rec {
		switch {
		case strings.Contains(line, "店名："):
			if o.Recipient == "" {
				r := afterPrefix(line, "店名：")
				if o.Phone != "" {
					r = strings.ReplaceAll(r, o.Phone, "") // 「店名：X 手机号」同行书写时剔除号码污染
					r = strings.TrimSpace(strings.Trim(r, "，,、；;。.：: -—"))
				}
				o.Recipient = r
			}
		case strings.Contains(line, "地址："):
			if o.Address == "" {
				o.Address = afterPrefix(line, "地址：")
			}
		case strings.Contains(line, "电话：") || strings.Contains(line, "电话:"):
			// 显式电话前缀行：「电话」标签本身不是收件人；但「张三 电话：P」时取标签前段为收件人
			if phoneLineIdx < 0 {
				phoneLineIdx = idx
			}
			if o.Recipient == "" {
				if i := strings.Index(line, "电话"); i > 0 {
					o.Recipient = strings.TrimSpace(strings.Trim(strings.TrimSpace(line[:i]), "，,、；;。.：: -—"))
				}
			}
		case phoneLineIdx < 0 && o.Phone != "" && strings.Contains(line, o.Phone):
			phoneLineIdx = idx
			if o.Recipient == "" {
				// 紧凑格式：手机号之前的部分为收件人（剔除分隔符）
				prefix := strings.TrimSpace(line[:strings.Index(line, o.Phone)])
				prefix = strings.Trim(prefix, "，,、；;。.：: -—")
				o.Recipient = prefix
			}
		}
	}
	// 无「地址：」显式前缀时：手机号所在行之后的首个非规格行作为收货地址
	if o.Address == "" && phoneLineIdx >= 0 {
		for _, line := range rec[phoneLineIdx+1:] {
			if strings.Contains(line, "【") {
				continue
			}
			o.Address = line
			break
		}
	}
	return o
}

// afterPrefix 取 line 中前缀之后的段（去首尾空白）。
func afterPrefix(line, prefix string) string {
	i := strings.Index(line, prefix)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(line[i+len(prefix):])
}

// addressPrefix 取地址前 8 个字符（契约 §3.2 约定1：无手机号时与收件人共同构成兜底匹配键；
// 按中文习惯取字符而非字节）。
func addressPrefix(address string) string {
	r := []rune(strings.TrimSpace(address))
	if len(r) > 8 {
		r = r[:8]
	}
	return string(r)
}

// auditMatchKey 匹配键（契约 §3.2 约定1）：手机号为主；无手机号时以
// 收件人 + 地址前 8 字符兜底，保证无手机号记录仍可对齐。规格不作匹配键（审核重点）。
func auditMatchKey(phone, recipient, address string) string {
	if strings.TrimSpace(phone) != "" {
		return "P:" + strings.TrimSpace(phone)
	}
	return "N:" + strings.TrimSpace(recipient) + "|" + addressPrefix(address)
}

// baselineSpecsString 基准侧规格展示串（与 AddressInfo.ProductSpecsString 同格式）。
func baselineSpecsString(specs []models.ProductSpec) string {
	parts := make([]string, 0, len(specs))
	for _, s := range specs {
		parts = append(parts, fmt.Sprintf("【%s%d斤】", s.Name, s.Weight))
	}
	return strings.Join(parts, ",")
}

// specsEqual 规格比较：数量与内容一致才视为一致（元素序不敏感，用户行内编辑可能调序）。
func specsEqual(a, b []models.ProductSpec) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(s models.ProductSpec) string { return s.Name + "|" + strconv.Itoa(s.Weight) }
	ka := make([]string, 0, len(a))
	kb := make([]string, 0, len(b))
	for i := range a {
		ka = append(ka, key(a[i]))
		kb = append(kb, key(b[i]))
	}
	sort.Strings(ka)
	sort.Strings(kb)
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

// diffTypeOrder items 排序权重（契约 §3.2：missing→wrong-key→wrong-spec→extra→normal）。
var diffTypeOrder = map[string]int{
	diffTypeMissing:   0,
	diffTypeWrongKey:  1,
	diffTypeWrongSpec: 2,
	diffTypeExtra:     3,
	diffTypeNormal:    4,
}

// compareBaselineWithPreview 程序对比（架构 §5.2）：
// 以匹配键对齐基准订单与预览行，分类 missing/wrong-key/wrong-spec/extra/normal。
// 纯只读函数：不修改 preview（调用方传入 global.AddressList 的共享底层数组，此处不写）。
func compareBaselineWithPreview(baseline []baselineOrder, preview []models.AddressInfo) []AuditItem {
	items := make([]AuditItem, 0, len(baseline)+len(preview))
	matched := make([]bool, len(preview))

	// 匹配键 → 预览行下标（同键多行时按出现顺序一对一消费）
	pmap := make(map[string][]int, len(preview))
	for j, p := range preview {
		key := auditMatchKey(p.Phone, p.Recipient, p.Address)
		pmap[key] = append(pmap[key], j)
	}

	// 1. 遍历基准：手机号在预览中不存在 → missing；存在则比较关键字段与规格
	for _, b := range baseline {
		key := auditMatchKey(b.Phone, b.Recipient, b.Address)
		var pj = -1
		for _, j := range pmap[key] {
			if !matched[j] {
				pj = j
				break
			}
		}
		if pj < 0 {
			items = append(items, AuditItem{
				VID:          0,
				Recipient:    b.Recipient,
				Phone:        b.Phone,
				Address:      b.Address,
				ProductSpecs: baselineSpecsString(b.Specs),
				DiffType:     diffTypeMissing,
				DiffDesc:     fmt.Sprintf("原始文本含此订单，预览表无对应行（%s）", orderRefDesc(b.Phone, b.Recipient)),
			})
			continue
		}
		matched[pj] = true
		p := preview[pj]

		keyDiffs := keyFieldDiffs(b, p)
		specDiff := !specsEqual(b.Specs, p.ProductSpecs)
		switch {
		case len(keyDiffs) > 0:
			// 关键字段错误优先于规格错误（严重级高，契约 §3.2）；规格差异一并说明
			if specDiff {
				keyDiffs = append(keyDiffs,
					fmt.Sprintf("原始文本规格为%s，预览为%s", baselineSpecsString(b.Specs), p.ProductSpecsString(false)))
			}
			items = append(items, AuditItem{
				VID:          p.PreviewID,
				Recipient:    b.Recipient,
				Phone:        b.Phone,
				Address:      b.Address,
				ProductSpecs: baselineSpecsString(b.Specs),
				DiffType:     diffTypeWrongKey,
				DiffDesc:     strings.Join(keyDiffs, "；"),
			})
		case specDiff:
			items = append(items, AuditItem{
				VID:          p.PreviewID,
				Recipient:    b.Recipient,
				Phone:        b.Phone,
				Address:      b.Address,
				ProductSpecs: baselineSpecsString(b.Specs),
				DiffType:     diffTypeWrongSpec,
				DiffDesc: fmt.Sprintf("原始文本规格为%s，预览为%s",
					baselineSpecsString(b.Specs), p.ProductSpecsString(false)),
			})
		default:
			items = append(items, AuditItem{
				VID:          p.PreviewID,
				Recipient:    p.Recipient,
				Phone:        p.Phone,
				Address:      p.Address,
				ProductSpecs: p.ProductSpecsString(false),
				DiffType:     diffTypeNormal,
			})
		}
	}

	// 2. 遍历预览：未被基准匹配的行 → extra
	for j, p := range preview {
		if matched[j] {
			continue
		}
		items = append(items, AuditItem{
			VID:          p.PreviewID,
			Recipient:    p.Recipient,
			Phone:        p.Phone,
			Address:      p.Address,
			ProductSpecs: p.ProductSpecsString(false),
			DiffType:     diffTypeExtra,
			DiffDesc:     fmt.Sprintf("预览表含此订单，原始文本无对应行（%s）", orderRefDesc(p.Phone, p.Recipient)),
		})
	}

	// 3. 排序：missing→wrong-key→wrong-spec→extra→normal，同级按 VID 升序
	sort.SliceStable(items, func(i, j int) bool {
		oi, oj := diffTypeOrder[items[i].DiffType], diffTypeOrder[items[j].DiffType]
		if oi != oj {
			return oi < oj
		}
		return items[i].VID < items[j].VID
	})
	return items
}

// orderRefDesc 差异描述中的订单定位段：优先手机号，无手机号时用收件人。
func orderRefDesc(phone, recipient string) string {
	if strings.TrimSpace(phone) != "" {
		return "手机号 " + strings.TrimSpace(phone)
	}
	if strings.TrimSpace(recipient) != "" {
		return "收件人 " + strings.TrimSpace(recipient)
	}
	return "无可匹配字段"
}

// keyFieldDiffs 关键字段（收件人/地址）差异描述；手机号不一致不会出现在此（作为匹配键）。
func keyFieldDiffs(b baselineOrder, p models.AddressInfo) (diffs []string) {
	if strings.TrimSpace(b.Recipient) != strings.TrimSpace(p.Recipient) {
		diffs = append(diffs, fmt.Sprintf("原始文本收件人为「%s」，预览为「%s」", b.Recipient, p.Recipient))
	}
	if strings.TrimSpace(b.Address) != strings.TrimSpace(p.Address) {
		diffs = append(diffs, fmt.Sprintf("原始文本收货地址为「%s」，预览为「%s」", b.Address, p.Address))
	}
	return
}

// ===== AI 复核（架构 §5.3：可疑项批量一次调用，专用提示词）=====

// ai_model.DeepSeekRequest 的 apiURL/modelName 常量未导出，此处按 callChat 模式复刻同值；
// Qwen 的 APIURL/ModelName 已导出（ai_model 包），直接复用。
const (
	auditDeepseekURL   = "https://api.deepseek.com/v1/chat/completions"
	auditDeepseekModel = "deepseek-chat"
)

// auditSystemPrompt AI 复核专用提示词（架构 §5.3；不复用识别提示词）。
const auditSystemPrompt = `你是订单审核助手。下面给出「原始输入文本」和「程序对比发现的差异项列表」。
程序对比是用确定性正则解析原始文本得到的基准订单与预览表数据的比对结果，可能存在误判。
请逐项判断每个差异是「真差异」（原始文本与预览数据确实不一致，需要修正预览数据）还是「误报」（程序对比的误判，例如正则切分错误、字段归属错误等造成的假差异）。
判定为「真差异」时，请依据原始文本给出具体的修正建议（说明预览表对应行应改成的正确值）；
判定为「误报」时，建议为空字符串。
只输出一个纯净的 JSON 数组，不要包含任何解释、Markdown 标记或额外文字。
数组中每个元素的格式为：{"序号": <差异项序号，整数>, "判定": "真差异"或"误报", "建议": "<修正建议或空字符串>"}
必须逐项覆盖输入的全部差异项，序号与输入一一对应。`

// auditChatRequest OpenAI 兼容的对话请求体（DeepSeek/Qwen 兼容模式同构）。
type auditChatRequest struct {
	Model       string            `json:"model"`
	Messages    []ai_model.Message `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	Stream      bool              `json:"stream"`
}

// aiAuditVerdict AI 复核单项判定（架构 §5.3 输出格式的解析结构）。
type aiAuditVerdict struct {
	Index   int    `json:"序号"`
	Verdict string `json:"判定"`
	Suggest string `json:"建议"`
}

// aiReviewItems 对可疑项批量调用一次 AI 复核，返回 序号(1 起) → 判定 的映射。
// 模型选择：DeepSeek 优先（apikey 已配置时），否则 Qwen；均未配置视为复核失败（降级）。
// 温度取低位值求判定稳定；超时/解析失败返回 error，由调用方置 aiFailed 降级（不阻塞响应）。
func aiReviewItems(text string, suspicious []AuditItem) (map[int]aiAuditVerdict, error) {
	if len(suspicious) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("原始输入文本：\n")
	b.WriteString(text)
	fmt.Fprintf(&b, "\n\n程序对比发现的差异项（共 %d 项）：\n", len(suspicious))
	for i, it := range suspicious {
		fmt.Fprintf(&b, "%d. [%s] 收件人=%s 手机号=%s 收货地址=%s 商品规格=%s\n",
			i+1, it.DiffType, it.Recipient, it.Phone, it.Address, it.ProductSpecs)
		fmt.Fprintf(&b, "   差异说明：%s\n", it.DiffDesc)
	}

	var apiKey, apiURL, model string
	var timeout int64
	var headers map[string]string
	switch {
	case global.Config.AiModel.Deepseek.Apikey != "":
		apiKey = global.Config.AiModel.Deepseek.Apikey
		apiURL, model = auditDeepseekURL, auditDeepseekModel
		timeout = global.Config.AiModel.Deepseek.Timeout
	case global.Config.AiModel.Qwen.Apikey != "":
		apiKey = global.Config.AiModel.Qwen.Apikey
		apiURL, model = ai_model.APIURL, ai_model.ModelName
		timeout = global.Config.AiModel.Qwen.Timeout
		headers = map[string]string{"X-DashScope-SSE": "enable"} // 与 ai_model/qwen.go callChat 一致
	default:
		return nil, errors.New("未配置可用的 AI 模型 apikey")
	}

	reqBody := auditChatRequest{
		Model: model,
		Messages: []ai_model.Message{
			{Role: "system", Content: auditSystemPrompt},
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.1, // 低温求判定稳定
		MaxTokens:   4096,
		Stream:      false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %w", err)
	}
	body, err := httputils.AiRequest(apiKey, apiURL, jsonData, timeout, headers)
	if err != nil {
		return nil, err
	}

	// DeepSeek/Qwen 均为 OpenAI 兼容结构，复用 QwenResponse 解析
	var apiResp ai_model.QwenResponse
	if err = json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, errors.New("模型未返回有效结果")
	}

	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	// 清理可能存在的 Markdown 代码块标记（与 ordertutil.RespBodyToJson 同口径）
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var verdicts []aiAuditVerdict
	if err = json.Unmarshal([]byte(content), &verdicts); err != nil {
		return nil, fmt.Errorf("AI 复核结果解析失败: %w", err)
	}

	result := make(map[int]aiAuditVerdict, len(verdicts))
	for _, v := range verdicts {
		// 判定值归一化：只接受冻结枚举「真差异」「误报」，其余措辞按包含关系归并
		switch {
		case strings.Contains(v.Verdict, "误报"):
			v.Verdict = "误报"
		case strings.Contains(v.Verdict, "真差异"):
			v.Verdict = "真差异"
		default:
			continue // 无有效判定的项保持空（前端按未判定展示）
		}
		if v.Index < 1 || v.Index > len(suspicious) {
			continue // 序号越界（AI 幻觉），忽略
		}
		result[v.Index] = v
	}
	if len(result) == 0 {
		return nil, errors.New("AI 复核未返回有效判定")
	}
	return result, nil
}

// ===== 审核入口（application 透传签名：T15 接线核对用）=====

// AuditPreviewData 预览审核入口（契约 §3.2）：
//  1. 识别在途（global.RecognitionInFlight > 0）→ 错误「识别进行中，请稍后再审核」（50000），
//     避免读到 LastInputText 已更新、AddressList 未写完的半更新状态（缺陷2，missing 误报）；
//  2. 预览态为空 → 错误「请先识别订单信息」（50000）；
//  3. 原始文本为空（如服务重启后）→ 错误「无可审核的原始文本，请重新识别」（50000）；
//  4. 程序对比：对 global.LastInputText 确定性解析为基准订单，与 global.AddressList
//     按匹配键对齐后分类差异并排序汇总；
//  5. AI 复核：仅对非 normal 可疑项批量调用一次；失败不阻塞——aiFailed=true、
//     aiVerdict/aiSuggest 保持空返回程序对比结果，并 Logaction(ErrLevel) 留痕（B2 不吞错）。
//
// 只读幂等：不修改 global.AddressList 与 global.LastInputText，可重复调用；
// AI 复核输出具非确定性，仅作建议（R3）。
func AuditPreviewData(req AuditRequest) (AuditResponse, error) {
	_ = req // 空结构体（契约 §3.2：无必填字段），保留入参供 application 层透传签名稳定

	// 在途检测（第一次）：识别入口已置位则直接拒绝，不读内存态
	if atomic.LoadInt32(&global.RecognitionInFlight) > 0 {
		return AuditResponse{}, errors.New("识别进行中，请稍后再审核")
	}

	// 快照读取：global 内存态的读取集中在此处，后续流程（含 AI 复核）只用快照，
	// 不再触碰 global，缩短并发窗口。
	text := global.LastInputText
	list := global.AddressList

	// 在途检测（第二次）：快照期间若有识别启动则放弃本次快照（可能已半更新）；
	// 检测后才启动的识别，其写入必然晚于本次快照完成，不影响一致性。
	if atomic.LoadInt32(&global.RecognitionInFlight) > 0 {
		return AuditResponse{}, errors.New("识别进行中，请稍后再审核")
	}

	if len(list) == 0 {
		return AuditResponse{}, errors.New("请先识别订单信息")
	}
	if strings.TrimSpace(text) == "" {
		return AuditResponse{}, errors.New("无可审核的原始文本，请重新识别")
	}

	baseline := parseBaselineOrders(text)
	items := compareBaselineWithPreview(baseline, list)

	resp := AuditResponse{Items: items}
	for _, it := range items {
		switch it.DiffType {
		case diffTypeMissing:
			resp.Summary.Missing++
		case diffTypeWrongKey, diffTypeWrongSpec:
			resp.Summary.Wrong++
		case diffTypeExtra:
			resp.Summary.Extra++
		case diffTypeNormal:
			resp.Summary.Normal++
		}
	}

	// AI 复核：仅可疑项（非 normal）参与，序号即其在 items 中的位置+1
	var suspiciousIdx []int
	for i, it := range items {
		if it.DiffType != diffTypeNormal {
			suspiciousIdx = append(suspiciousIdx, i)
		}
	}
	if len(suspiciousIdx) == 0 {
		return resp, nil // 无可疑项不调用 AI（契约 §3.2 约定2）
	}
	suspicious := make([]AuditItem, 0, len(suspiciousIdx))
	for _, i := range suspiciousIdx {
		suspicious = append(suspicious, items[i])
	}

	verdicts, err := aiReviewItems(text, suspicious)
	if err != nil {
		// 失败不阻塞：程序对比结果独立有效，显式标记 + 留痕（B2）
		resp.AiFailed = true
		logrus.Errorf("AI复核失败: %v", err)
		alog.Logaction(enum.LogErrLevel, "AI复核失败", err.Error())
		return resp, nil
	}
	for k, i := range suspiciousIdx {
		// AI 提示词中的序号 = 可疑项在 suspicious 列表中的 1 起位置，suspicious[k] 对应 items[i]
		if v, ok := verdicts[k+1]; ok {
			items[i].AiVerdict = v.Verdict
			items[i].AiSuggest = v.Suggest
		}
	}
	return resp, nil
}
