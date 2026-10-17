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
// 契约 §3.5 变更4（2026-10-17）：查询失败上抛错误（B2 不吞错），由 handler 转 50000
func OrderQuery(req order.OrderQueryRequest) (order.OrderQueryResponse, error) {
	return order.OrderQuery(req)
}

// ProvinceStat 省份统计项（与 server 层同类型，契约 §3.1）
type ProvinceStat = order.ProvinceStat

// ProvinceStatsResponse 省份统计响应（与 server 层同类型）
type ProvinceStatsResponse = order.ProvinceStatsResponse

// ProvinceStats 省份统计。start/end 均为零值时全量；handler 层保证两参数成对传入。
func ProvinceStats(start, end time.Time) (ProvinceStatsResponse, error) {
	return order.ProvinceStats(start, end)
}

// UpdateOrderRequest 修改历史订单入参（与 server 层同类型，契约 §3.3）。
// handler 经此别名取类型（B1：禁止 handler 直连 server）。
type UpdateOrderRequest = order.UpdateOrderRequest

// UpdateOrder 修改一条历史订单（契约 §3.3）。T4：参数校验在 handler，
// 存在性校验/规格解析在 server。clientIP 为操作来源 IP（M3 留痕），由 handler 注入。
func UpdateOrder(id uint, req UpdateOrderRequest, clientIP string) error {
	return order.UpdateOrder(id, req, clientIP)
}

// DeleteOrder 软删除一条历史订单（契约 §3.4，非幂等：重复删除返回 50000 订单不存在）。
// clientIP 为操作来源 IP（M3 留痕），由 handler 注入。
func DeleteOrder(id uint, clientIP string) error {
	return order.DeleteOrder(id, clientIP)
}

// AuditRequest 预览审核入参（与 server 层同类型，契约 §3.2：空 JSON 对象，无必填字段；
// 原始文本由后端 global.LastInputText 暂存，请求不回传文本）。
type AuditRequest = order.AuditRequest

// AuditResponse 预览审核出参（与 server 层同类型，契约 §3.2：
// items 按 missing→wrong-key→wrong-spec→extra→normal 排序，同级按 VID 升序）。
type AuditResponse = order.AuditResponse

// AuditPreview 预览审核：程序对比 + AI 复核（契约 §3.2，只读幂等端点，
// 不修改预览数据与原始文本；AI 复核失败不阻塞，aiFailed=true 降级返回程序对比结果）。
func AuditPreview(req AuditRequest) (AuditResponse, error) {
	return order.AuditPreviewData(req)
}
