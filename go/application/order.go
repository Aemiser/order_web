package application

import (
	"order_server/go/global"
	"order_server/go/models/enum"
	"order_server/go/server/order"
	ordertutil "order_server/go/utils/order"
	"time"
)

// OrderQueryRequest 订单详情查询入参（与 server 层同类型）
type OrderQueryRequest = order.OrderQueryRequest

// OrderQueryResponse 订单详情查询出参
type OrderQueryResponse = order.OrderQueryResponse

// ListAIModels 返回已配置的大模型
func ListAIModels() map[string]int {
	return global.AiMap
}

// Recognition 同步订单识别
func Recognition(text string, model enum.AIModelType, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	return order.OrderRec(text, model, previewType)
}

// StreamRecognition 流式订单识别，回调由 handler 注入（SSE 写入）
func StreamRecognition(text string, model enum.AIModelType, previewType enum.RecPreviewType, callback order.StreamCallback, done order.DoneCallback) {
	order.StreamOrderRec(text, model, previewType, callback, done)
}

// GetPreview 获取预览列表
func GetPreview() []ordertutil.ExceladdressInfo {
	return order.GetpreviewData()
}

// SetPreview 修改一条预览数据
func SetPreview(address ordertutil.ExceladdressInfo) error {
	return order.SetpreviewData(address)
}

// DeletePreview 删除一条预览数据
func DeletePreview(id uint) error {
	return order.DeletepreviewData(id)
}

// CreateOrder 把预览数据导入数据库
func CreateOrder(date time.Time) error {
	return order.CreateOrder(date)
}

// ExportOrder 导出预览数据为 Excel，返回下载地址
func ExportOrder() (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := order.SaveToExcel()
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}

// OrderQuery 分页查询历史订单
func OrderQuery(req order.OrderQueryRequest) order.OrderQueryResponse {
	return order.OrderQuery(req)
}
