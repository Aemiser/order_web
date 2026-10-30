package wuyinutil

import (
	"encoding/json"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 吴饮链路的工具函数。与 ordertutil（一点点）平行：类型与计价单位不同（件 vs 斤），
// 不复用以避免两品牌语义互相渗透。

// WuyinExceladdressInfo 吴饮预览/列表行的前端交换格式。
// ProductSpecs 为字符串形态（"【香水柠檬9斤】"，条目数即件数），与一点点预览行
// 同构，便于复用前端编辑交互。
type WuyinExceladdressInfo struct {
	CreateAt     time.Time `json:"createdAt"`
	ID           uint      `json:"id"`
	PreviewID    uint      `json:"VID"`
	Recipient    string    `json:"收件人"`
	Phone        string    `json:"手机号"`
	Address      string    `json:"收货地址"`
	Province     string    `json:"省份"`
	ProductSpecs string    `json:"商品规格"`
	Remarks      string    `json:"备注"`
	Tag          string    `json:"标签"`
	Sender       string    `json:"发送人"`
}

// WuyinToExcelAddress 内部模型 → 前端交换行
func WuyinToExcelAddress(addresss []models.WuyinAddressInfo) (exceladdress []WuyinExceladdressInfo) {
	for _, address := range addresss {
		exceladdress = append(exceladdress, WuyinExceladdressInfo{
			PreviewID:    address.PreviewID,
			ID:           address.ID,
			CreateAt:     address.CreatedAt,
			Address:      address.Address,
			Phone:        address.Phone,
			Province:     address.Province,
			ProductSpecs: address.ProductSpecsString(false),
			Recipient:    address.Recipient,
			Remarks:      address.Remarks,
			Tag:          address.Tag,
			Sender:       address.Sender,
		})
	}
	return
}

// SetVidWuyin 为新识别批次分配预览 VID（追加在现有 WuyinAddressList 之后）
func SetVidWuyin(infolist []models.WuyinAddressInfo) []models.WuyinAddressInfo {
	length := len(global.WuyinAddressList)
	for i := range infolist {
		addr := &infolist[i]
		addr.PreviewID = uint(length + i + 1)
	}
	return infolist
}

// SpecItemsProcessing 解析"【香水柠檬9斤】"形态的规格字符串（预览编辑回传用）。
// 与 ordertutil.Specprocessing 同构（斤单位）；多个条目即多件。
func SpecItemsProcessing(input string) (models.WuyinProductSpecsJSON, error) {
	re := regexp.MustCompile(`【([^】]+?)(\d+)(?:斤)?】`)
	matches := re.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("没有找到匹配的内容")
	}
	var items models.WuyinProductSpecsJSON
	for _, match := range matches {
		if len(match) >= 3 {
			number, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, fmt.Errorf("转换数量失败: %v", err)
			}
			items = append(items, models.WuyinProductSpec{Name: match[1], Weight: number})
		}
	}
	return items, nil
}

// RespBodyToJsonWuyin 解析模型返回的 JSON（数组或单对象）为吴饮地址列表。
func RespBodyToJsonWuyin(content string) ([]models.WuyinAddressInfo, error) {
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var addressList []models.WuyinAddressInfo
	if err := json.Unmarshal([]byte(content), &addressList); err != nil {
		var singleAddr models.WuyinAddressInfo
		if err2 := json.Unmarshal([]byte(content), &singleAddr); err2 == nil {
			addressList = []models.WuyinAddressInfo{singleAddr}
		} else {
			return nil, fmt.Errorf("地址信息解析失败: %w\n原始返回: %s", err, content)
		}
	}
	return addressList, nil
}

// senderLineRe 匹配消息头两种形态："[2026-10-29 16:52] 广西～南昌订货微信: "
// （wechat-cli --format text 导出）与裸的 "合肥仓工作微信: "（手工整理）。
// 时间戳段整体可选，冒号前捕获发送人昵称。
var senderLineRe = regexp.MustCompile(`^(?:\[\d{4}-\d{2}-\d{2}[^\]]*\]\s*)?([^:\n]{1,64}):\s`)

// ExtractSender 从一段粘贴文本中提取首个"发送人:"前缀（wechat-cli --format text 的
// 消息头格式）。AI 兜底提取失败时用于整段同发送人的场景；提取不到返回空串。
func ExtractSender(text string) string {
	if m := senderLineRe.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// recipientRe 提取地址中最后一个中文括号内的人名（吴饮订单"地址（收件人）吾饮"格式）。
var recipientRe = regexp.MustCompile(`（([^（）]{2,20})）`)

// ExtractRecipientFromAddress 从"广西百色市…（潘雪晶）吾饮"形态的客户字段提取收件人。
// 取最后一个括号组（前面括号可能是店名/楼宇名）；无括号返回空串。
func ExtractRecipientFromAddress(customer string) string {
	ms := recipientRe.FindAllStringSubmatch(customer, -1)
	if len(ms) == 0 {
		return ""
	}
	return ms[len(ms)-1][1]
}

// CleanCustomerAddress 清洗客户字段为收货地址：去掉结尾的"吾饮"字样与首尾空白。
func CleanCustomerAddress(customer string) string {
	s := strings.TrimSpace(customer)
	s = strings.TrimSuffix(s, "吾饮")
	return strings.TrimSpace(s)
}

// NormalizeProductName 物料名称归一："香水柠檬（9斤）"→"香水柠檬"（去掉规格括号后缀）。
func NormalizeProductName(name string) string {
	s := strings.TrimSpace(name)
	if i := strings.IndexAny(s, "（("); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}
