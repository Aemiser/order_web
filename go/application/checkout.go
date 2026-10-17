package application

import (
	"order_server/go/server/checkout"
	"time"
)

// GetDayData 按日期范围查询结账数据
func GetDayData(start, end time.Time) checkout.CheckoutResponse {
	return checkout.GetDaydata(start, end)
}

// SetProductBelong 保存商品归属配置
func SetProductBelong(mapp map[string][]string) {
	checkout.SetFieldList(mapp)
}

// SetPrice 保存价格并返回更新的结账数据
// 契约 §3.6：保存事务失败（任一项失败整体回滚）时透传 error，由 handler 转 50000
func SetPrice(list []checkout.ProPriceRequest) (checkout.CheckoutResponse, error) {
	return checkout.SetPrice(list)
}

// ExportCheckout 生成财务账单 Excel，返回下载地址
func ExportCheckout() (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := checkout.Checkout()
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}
