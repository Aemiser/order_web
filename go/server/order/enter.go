package order

import (
	"order_server/go/global"
	"order_server/go/models/enum"
	"order_server/go/server/order/ai_model"
	ordertutil "order_server/go/utils/order"
	"sync/atomic"
)

// OrderRec 识别统一入口（同步）。
// 识别全程维持 RecognitionInFlight 在途计数（缺陷2修复）：recordLastInputText 在入口
// 先行更新 LastInputText，AddressList 要等 AI 返回后才写入——在途窗口内审核会读到
// 半更新状态（新增批误报 missing），audit 入口据此拒绝（见 audit.go AuditPreviewData）。
func OrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	atomic.AddInt32(&global.RecognitionInFlight, 1)
	defer atomic.AddInt32(&global.RecognitionInFlight, -1)

	recordLastInputText(inputText, previewType)
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

// StreamOrderRec 识别统一入口（SSE 流式）。
// 在途计数同 OrderRec：SSE 识别（含中途断流）随本函数返回而结束，defer 保证计数回落。
func StreamOrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType, callback StreamCallback, doneCallback DoneCallback) {
	atomic.AddInt32(&global.RecognitionInFlight, 1)
	defer atomic.AddInt32(&global.RecognitionInFlight, -1)

	recordLastInputText(inputText, previewType)
	switch model {
	case enum.AIModelType_DeepSeek:
		ai_model.StreamRecDeepseek(inputText, previewType, callback, doneCallback)
	case enum.AIModelType_Qwen:
		ai_model.StreamRecQwen(inputText, previewType, callback, doneCallback)
	}
}

// recordLastInputText 按预览模式维护最近一次识别的原始文本（契约 §3.2 细化，供审核对比）：
// previewType=RE（清空重识别）→ 覆盖为本次 text；previewType=ADD（追加）→ 追加 "\n"+text。
// 若不做该区分，追加识别后审核会因基准文本缺失首批订单而误判为「多单」（契约 Q7）。
// 已知降级（可接受）：ai_model 内部识别失败时不改动 AddressList，但此处已先行更新
// LastInputText，此时审核会把失败批次判为 missing 项，由用户重新识别兜底。
// B6/R8：LastInputText 与 AddressList 同为内存态、单请求流写入，无并发场景，不加锁。
func recordLastInputText(text string, previewType enum.RecPreviewType) {
	if previewType == enum.RecPreviewType_ADD && global.LastInputText != "" {
		global.LastInputText += "\n" + text
		return
	}
	// RE 覆盖；ADD 且基准为空（如服务重启后直接追加）时等价于覆盖，避免产生前导空行
	global.LastInputText = text
}
