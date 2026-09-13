package order

import (
	"order_server/go/models/enum"
	"order_server/go/server/order/ai_model"
	ordertutil "order_server/go/utils/order"
)

func OrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	switch model {
	case enum.AIModelType_DeepSeek:
		return ai_model.OrderRecDeepseek(inputText, previewType)
	case enum.AIModelType_Qwen:
		return ai_model.OrderRecQwen(inputText, previewType)
	}
	return nil
}

type StreamCallback func(data string) error
type DoneCallback func()

func StreamOrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType, callback StreamCallback, doneCallback DoneCallback) {
	switch model {
	case enum.AIModelType_DeepSeek:
		ai_model.StreamRecDeepseek(inputText, previewType, callback, doneCallback)
	case enum.AIModelType_Qwen:
		ai_model.StreamRecQwen(inputText, previewType, callback, doneCallback)
	}
}
