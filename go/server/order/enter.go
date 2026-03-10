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
