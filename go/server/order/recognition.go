package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DeepSeekRequest API请求结构
type DeepSeekRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse API响应结构
type DeepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const (
	apiURL    = "https://api.deepseek.com/v1/chat/completions"
	modelName = "deepseek-chat"
)

// 系统提示词
const systemPrompt = `## 角色与任务
你是一个信息提取助手。请从用户提供的原始文本中，精确提取多条地址信息，并整理成指定的JSON数组格式。

## 输入文本格式示例
用户输入的文本通常包含多条记录，每条记录格式类似如下（顺序和格式可能略有变化）：
3-4\n店名：XXX店\n电话：XXX\n地址：XXX，1點點奶茶店（请送到门店）\n【商品规格1】【商品规格2】...（备注信息）

多条记录之间可能用换行、空格或其他符号分隔，也可能连续排列。

## 提取规则（非常重要）
1. **记录分割**: 请智能识别文本中的多条记录。每条记录通常包含“店名”、“电话”、“地址”、“【商品规格】”这几个核心要素。以“3-4”或“店名：”作为一条新记录的开始标记。
2. **收件人**: 请严格将“店名：”后面的内容（例如“长虹路旗舰店”）填入此字段。如果记录中没有明确的“店名：”前缀但有店铺名称（如“东湖分店”紧挨着电话号码），则直接提取该店铺名作为收件人。
3. **手机号**: 提取每条记录中的11位手机号码。
4. **收货地址**: 提取“地址：”后面的完整内容。必须包含“1點點奶茶店（请送到门店）”这部分（如果存在），确保配送的准确性。
5. **商品规格**: 
   - 提取被【】括起来的所有内容，每条记录可能包含一个或多个商品规格
   - 将每个【】中的内容解析为结构化的对象格式，包含两个字段：
     * 'name': 商品名称（字符串类型）
     * 'weight': 商品重量（数值类型，单位为斤）
   - 解析规则：
     * 商品名称和重量之间可能用空格、冒号、或其他符号分隔
     * 常见的格式如：“香水柠檬5斤”、“黄柠檬10斤”、“广香一级30斤”等
     * 重量单位默认为“斤”，需要去除单位只保留数值
6. **备注**: 
   - 提取最后一个【商品规格】的右括号“】”之后的所有内容
   - 备注信息可能包含括号、特殊符号、文字说明等
   - 如果【商品规格】之后没有内容，则备注字段为空字符串
   - 备注信息应完整保留原始格式，包括括号、空格等

## 商品规格解析示例
- 输入："【香水柠檬5斤】" → 解析为 '{"name": "香水柠檬", "weight": 5}'
- 输入："【黄柠檬10斤】" → 解析为 '{"name": "黄柠檬", "weight": 10}'
- 输入："【广香一级30斤】" → 解析为 '{"name": "广香一级", "weight": 30}'
- 输入："【香水柠檬5斤】【黄柠檬10斤】" → 解析为 '[{"name": "香水柠檬", "weight": 5}, {"name": "黄柠檬", "weight": 10}]'

## 备注提取示例
- 输入："【黄柠檬10斤】（记得发两箱哦~~~）" → 备注为 "（记得发两箱哦~~~）"
- 输入："【广香一级30斤】 请尽快发货" → 备注为 " 请尽快发货"
- 输入："【香水柠檬5斤】" → 备注为 ""（空字符串）

## 输出要求
- 请只输出一个纯净的、严格的**JSON数组**，不要包含任何解释、Markdown标记或额外的文字。
- 数组中的每个元素是一个JSON对象，包含以下字段：
  * '收件人': 字符串类型
  * '手机号': 字符串类型
  * '收货地址': 字符串类型
  * '商品规格': 数组类型，数组中的每个元素是 '{"name": "商品名称", "weight": 重量数值}' 格式的对象
  * '备注': 字符串类型，如果无备注信息则为空字符串

- 如果输入文本中只有一条记录，也请以包含单个对象的数组形式返回（例如：[{...}]）。
- 严格按照输入顺序输出对应的识别结果。

## 输出格式示例
[
{
"收件人": "长虹路旗舰店",
"手机号": "13800138001",
"收货地址": "福建省厦门市思明区长虹路1号，1點點奶茶店（请送到门店）",
"商品规格": [
{"name": "香水柠檬", "weight": 5}
],
"备注": ""
},
{
"收件人": "东湖分店",
"手机号": "13800138002",
"收货地址": "福建省厦门市湖里区东湖西路2号示例大厦A栋一楼，1點點奶茶店（请送到门店）",
"商品规格": [
{"name": "香水柠檬", "weight": 5},
{"name": "黄柠檬", "weight": 10}
],
"备注": ""
},
{
"收件人": "张三",
"手机号": "13800138003",
"收货地址": "北京市东城区示例街3号-某茶馆",
"商品规格": [
{"name": "黄柠檬", "weight": 10},
{"name": "广香一级", "weight": 30}
],
"备注": "（记得发两箱哦~~~）"
}
]

## 用户输入示例
3-4
店名：长虹路旗舰店
电话：13800138001
地址：福建省厦门市思明区长虹路1号，1點點奶茶店（请送到门店）
【香水柠檬5斤】

3-4
店名：东湖分店
电话：13800138002
地址：福建省厦门市湖里区东湖西路2号示例大厦A栋一楼，1點點奶茶店（请送到门店）
【香水柠檬5斤】【黄柠檬10斤】

张三， 13800138003
北京市东城区示例街3号-某茶馆
【黄柠檬10斤】
【广香一级30斤】（记得发两箱哦~~~）`

func OrderRec(inputText string, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	apiKey := global.Config.Deepseek.Apikey
	// 调用API进行识别
	addressList, err := RecognizeAddresses(apiKey, inputText)
	if err != nil {
		logrus.Errorf("识别失败: %v", err)
		alog.Logaction(enum.LogWarnLevel, "DeepSeek API识别识别失败", err.Error())
		return nil
	}
	// vid赋值
	addressList = ordertutil.SetVid(addressList)

	// JSON格式
	jsonData, _ := json.MarshalIndent(addressList, "", "  ")
	fmt.Println(string(jsonData))

	// 日志记录
	logcontent := "原始输入:" + "\n" + inputText + "\n" + "识别结果:" + "\n" + string(jsonData)
	alog.Logaction(enum.LogInfoLevel, "DeepSeek API识别", logcontent)

	// 预览数据模式判断
	// 追加
	if previewType == enum.RecPreviewType_ADD {
		global.AddressList = append(global.AddressList, addressList...)
	}

	// 清空重新识别
	if previewType == enum.RecPreviewType_RE {
		global.AddressList = addressList
	}

	// 转换成预览数据返回给前台
	return ordertutil.AddressToExcelAddress(global.AddressList)
}

// RecognizeAddresses 调用DeepSeek API识别多条地址
func RecognizeAddresses(apiKey, inputText string) ([]models.AddressInfo, error) {
	// 构建请求体
	reqBody := DeepSeekRequest{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: inputText},
		},
		Temperature: 0.5, // 低温使输出更确定
		MaxTokens:   2000,
		Stream:      false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %w", err)
	}

	// 创建HTTP客户端（设置超时）
	client := &http.Client{
		Timeout: 600 * time.Second, // 设置超时时间为600秒
	}

	// 创建请求
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp DeepSeekResponse
	if err = json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	// 检查API错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("模型未返回有效结果")
	}

	// 获取模型返回的JSON字符串
	content := apiResp.Choices[0].Message.Content

	// 清理可能存在的Markdown代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// 解析JSON数组
	var addressList []models.AddressInfo
	if err = json.Unmarshal([]byte(content), &addressList); err != nil {
		// 尝试解析为单个对象
		var singleAddr models.AddressInfo
		if err2 := json.Unmarshal([]byte(content), &singleAddr); err2 == nil {
			addressList = []models.AddressInfo{singleAddr}
		} else {
			return nil, fmt.Errorf("地址信息解析失败: %w\n原始返回: %s", err, content)
		}
	}

	return addressList, nil
}
