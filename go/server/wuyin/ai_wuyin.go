package wuyin

import (
	"encoding/json"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	"order_server/go/utils/httputils"
	wuyinutil "order_server/go/utils/wuyin"

	"github.com/sirupsen/logrus"
)

// 吴饮识别的 AI 调用层。与 server/order/ai_model 平行：请求结构自持
// （不依赖 ai_model 包内类型），提示词读 global.SystemPromptWuyin，
// 解析目标是 WuyinAddressInfo（件计价 + 发送人字段）。
// deepseek 与 qwen 均为 OpenAI 兼容格式，共用一套请求/响应结构。

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiRequest struct {
	Model       string      `json:"model"`
	Messages    []aiMessage `json:"messages"`
	Temperature float64     `json:"temperature"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Stream      bool        `json:"stream"`
}

type aiResponse struct {
	Choices []struct {
		Message aiMessage `json:"message"`
		Delta   aiMessage `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// aiEndpoint 按模型类型取 (apiURL, modelName, apiKey, timeout)
func aiEndpoint(model enum.AIModelType) (apiURL, modelName, apiKey string, timeout int64, err error) {
	switch model {
	case enum.AIModelType_DeepSeek:
		return "https://api.deepseek.com/v1/chat/completions", "deepseek-chat",
			global.Config.AiModel.Deepseek.Apikey, global.Config.AiModel.Deepseek.Timeout, nil
	case enum.AIModelType_Qwen:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions", "qwen-plus",
			global.Config.AiModel.Qwen.Apikey, global.Config.AiModel.Qwen.Timeout, nil
	}
	return "", "", "", 0, fmt.Errorf("不支持的模型类型: %d", model)
}

// buildMessages 组装 system+user 消息（吴饮提示词）
func buildMessages(inputText string) []aiMessage {
	return []aiMessage{
		{Role: "system", Content: global.SystemPromptWuyin},
		{Role: "user", Content: inputText},
	}
}

// recognizeWuyin 同步调用，返回吴饮地址列表（未打标、未赋 VID）
func recognizeWuyin(model enum.AIModelType, inputText string) ([]models.WuyinAddressInfo, error) {
	apiURL, modelName, apiKey, timeout, err := aiEndpoint(model)
	if err != nil {
		return nil, err
	}
	reqBody := aiRequest{
		Model:       modelName,
		Messages:    buildMessages(inputText),
		Temperature: 0.5,
		MaxTokens:   8192,
		Stream:      false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %w", err)
	}
	body, err := httputils.AiRequest(apiKey, apiURL, jsonData, timeout, nil)
	if err != nil {
		return nil, err
	}
	var apiResp aiResponse
	if err = json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("模型未返回有效结果")
	}
	return wuyinutil.RespBodyToJsonWuyin(apiResp.Choices[0].Message.Content)
}

// streamRecognizeWuyin 流式调用：chunkCallback 收到增量片段（可能拼出完整 JSON 对象时
// 解析为整条记录），doneCallback 在流结束时回调。行为与一点点 StreamRec 同构。
func streamRecognizeWuyin(model enum.AIModelType, inputText string, chunkCallback func(string) error, doneCallback func()) {
	apiURL, modelName, apiKey, timeout, err := aiEndpoint(model)
	if err != nil {
		logrus.Errorf("吴饮识别模型配置错误: %v", err)
		doneCallback()
		return
	}
	reqBody := aiRequest{
		Model:       modelName,
		Messages:    buildMessages(inputText),
		Temperature: 0.5,
		MaxTokens:   8192,
		Stream:      true,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logrus.Errorf("吴饮识别请求序列化失败: %v", err)
		doneCallback()
		return
	}
	err = httputils.AiStreamRequest(apiKey, apiURL, jsonData, timeout, nil, func(data string) error {
		if data == "DONE" {
			doneCallback()
			return nil
		}
		var streamResp aiResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return nil
		}
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			return chunkCallback(streamResp.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("吴饮流式请求失败: %v", err)
	}
}

// logWuyinRec 识别留痕（与一点点同款：原始输入 + 识别结果）
func logWuyinRec(title string, inputText string, addressList []models.WuyinAddressInfo) {
	jsonData, _ := json.MarshalIndent(addressList, "", "  ")
	logcontent := "原始输入:" + "\n" + inputText + "\n" + "识别结果:" + "\n" + string(jsonData)
	alog.Logaction(enum.LogInfoLevel, title, logcontent)
}
