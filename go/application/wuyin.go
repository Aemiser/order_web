package application

import (
	"io"
	"order_server/go/models"
	"order_server/go/models/enum"
	"order_server/go/server/wuyin"
	wuyinutil "order_server/go/utils/wuyin"
	"time"
)

// ---- 吴饮（吾饮良品）品牌链路的 application 层：handler 不直连 server（同 B1 规则）----

// 与 server 层同类型的入参/出参别名
type (
	WuyinOrderQueryRequest  = wuyin.WuyinOrderQueryRequest
	WuyinOrderQueryResponse = wuyin.WuyinOrderQueryResponse
	WuyinTagStat            = wuyin.WuyinTagStat
	WuyinTagConfigRequest   = wuyin.TagConfigRequest
	UpdateWuyinOrderRequest = wuyin.UpdateWuyinOrderRequest
)

// WuyinCheckDB 吴饮端点统一前置校验
func WuyinCheckDB() error {
	return wuyin.CheckDB()
}

// WuyinRecognition 吴饮同步识别
func WuyinRecognition(text string, model enum.AIModelType, previewType enum.RecPreviewType) []wuyinutil.WuyinExceladdressInfo {
	return wuyin.OrderRecWuyin(text, model, previewType)
}

// WuyinStreamRecognition 吴饮流式识别（回调由 handler 注入 SSE 写入）
func WuyinStreamRecognition(text string, model enum.AIModelType, previewType enum.RecPreviewType, callback wuyin.StreamCallback, done wuyin.DoneCallback) {
	wuyin.StreamOrderRecWuyin(text, model, previewType, callback, done)
}

// GetWuyinPreview 吴饮预览列表
func GetWuyinPreview() []wuyinutil.WuyinExceladdressInfo {
	return wuyin.GetWuyinPreview()
}

// SetWuyinPreview 修改一条吴饮预览数据
func SetWuyinPreview(address wuyinutil.WuyinExceladdressInfo) error {
	return wuyin.SetWuyinPreview(address)
}

// DeleteWuyinPreview 删除一条吴饮预览数据
func DeleteWuyinPreview(id uint) error {
	return wuyin.DeleteWuyinPreview(id)
}

// ImportWuyinExcel 表格导入（multipart 文件经 handler 读出 reader 与文件名）
func ImportWuyinExcel(reader io.Reader, filename, tag string) ([]wuyinutil.WuyinExceladdressInfo, error) {
	return wuyin.ImportExcel(reader, filename, tag)
}

// CreateWuyinOrder 吴饮预览数据导入独立库
func CreateWuyinOrder(date time.Time) error {
	return wuyin.CreateWuyinOrder(date)
}

// ExportWuyinOrder 吴饮预览数据导出 Excel，返回下载地址
func ExportWuyinOrder() (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := wuyin.SaveWuyinToExcel()
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}

// WuyinOrderQuery 吴饮订单分页查询
func WuyinOrderQuery(req WuyinOrderQueryRequest) (WuyinOrderQueryResponse, error) {
	return wuyin.WuyinOrderQuery(req)
}

// WuyinTagStats 标签订单数统计（详情页侧边栏）
func WuyinTagStats(start, end time.Time) ([]WuyinTagStat, error) {
	return wuyin.WuyinTagStats(start, end)
}

// ListWuyinTagConfigs 标签配置列表
func ListWuyinTagConfigs() []models.WuyinTagConfig {
	return wuyin.ListTagConfigs()
}

// CreateWuyinTagConfig 新增标签
func CreateWuyinTagConfig(req WuyinTagConfigRequest) error {
	return wuyin.CreateTagConfig(req)
}

// UpdateWuyinTagConfig 修改标签
func UpdateWuyinTagConfig(id uint, req WuyinTagConfigRequest) error {
	return wuyin.UpdateTagConfig(id, req)
}

// DeleteWuyinTagConfig 删除标签
func DeleteWuyinTagConfig(id uint) error {
	return wuyin.DeleteTagConfig(id)
}

// UpdateWuyinOrder 修改吴饮历史订单
func UpdateWuyinOrder(id uint, req UpdateWuyinOrderRequest) error {
	return wuyin.UpdateWuyinOrder(id, req)
}

// DeleteWuyinOrder 删除吴饮历史订单
func DeleteWuyinOrder(id uint) error {
	return wuyin.DeleteWuyinOrder(id)
}

// ---- 吴饮结账（件计价 × 标签分组）----

// WuyinCheckoutResponse / WuyinPriceRequest 与 server 层同类型
type (
	WuyinCheckoutResponse = wuyin.WuyinCheckoutResponse
	WuyinCheckoutProduct  = wuyin.WuyinCheckoutProduct
	WuyinPriceRequest     = wuyin.WuyinPriceRequest
)

// GetWuyinDayData 按标签分组的结账数据（tags 为空 = 全部标签）
func GetWuyinDayData(start, end time.Time, tags []string) ([]WuyinCheckoutResponse, error) {
	return wuyin.GetWuyinDayData(start, end, tags)
}

// SetWuyinPrice 保存价格并以最近一次查询范围重算
func SetWuyinPrice(list []WuyinPriceRequest) ([]WuyinCheckoutResponse, error) {
	return wuyin.SetWuyinPrice(list)
}

// ExportWuyinCheckout 结账 Excel 导出（每标签一个 sheet），返回下载地址
func ExportWuyinCheckout(tags []string) (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := wuyin.ExportWuyinCheckout(tags)
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}
