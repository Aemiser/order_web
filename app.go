package main

import (
	"context"
	"fmt"
	"order_server/go/global"
	"order_server/go/models/enum"
	"order_server/go/server/checkout"
	"order_server/go/server/config"
	alog "order_server/go/server/log"
	"order_server/go/server/order"
	ordertutil "order_server/go/utils/order"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// 订单识别
func (a *App) OrderRecognition(context string, model enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	return order.OrderRec(context, model)
}

// 获取配置消息
func (a *App) GetConfig() config.ReConf {
	return config.GetConf()
}

// 修改配置消息
func (a *App) SetConfig(conf config.ReConf) {
	config.SetConf(conf)
}

// 查询订单详情
func (a *App) OrderQuery(oq order.OrderQueryRequest) order.OrderQueryResponse {
	return order.OrderQuery(oq)
}

// 获取预览数据
func (a *App) GetpreviewData() []ordertutil.ExceladdressInfo {
	return order.GetpreviewData()
}

// 修改预览数据
func (a *App) SetpreviewData(addressList ordertutil.ExceladdressInfo) string {
	err := order.SetpreviewData(addressList)
	if err != nil {
		return err.Error()
	}
	return "修改成功"
}

// 删除预览数据
func (a *App) DeletepreviewData(id uint) string {
	err := order.DeletepreviewData(id)
	if err == nil {
		return "删除成功"
	}
	return err.Error()
}

// 订单导入数据库
func (a *App) CreateOrder() string {
	err := order.CreateOrder()
	if err == nil {
		return "订单导入成功"
	}
	return err.Error()
}

// 导出预览数据xlsx
func (a *App) SaveToExcel() string {
	// 选择文件夹
	folderPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "选择保存文件夹",
		DefaultDirectory:     "~",  // 默认打开用户目录
		CanCreateDirectories: true, // 允许用户创建新文件夹
	})
	if err != nil {
		logrus.Errorf("选择文件夹失败: %v", err)
		alog.Logaction(enum.LogErrLevel, "选择文件夹失败", err.Error())
	}
	global.Config.SavePath.Excel = folderPath

	order.SaveToExcel()

	// 使用 BrowserOpenURL 打开文件夹
	// 在 Windows 上会打开资源管理器，macOS 上打开 Finder
	runtime.BrowserOpenURL(a.ctx, folderPath)

	return "导出成功"
}

// 获取账单的查询数据
func (a *App) GetChackDaydata(statime time.Time, endtime time.Time) checkout.CheckoutResponse {
	return checkout.GetDaydata(statime, endtime)
}

// 修改商品归属
func (a *App) SetProductBelong(productBelongMap map[string][]string) {
	checkout.SetFieldList(productBelongMap)
}

// 保存价格数据
func (a *App) SetPriceData(req []checkout.ProPriceRequest) checkout.CheckoutResponse {
	return checkout.SetPrice(req)
}

// 获取账单的查询数据
func (a *App) GetDaydata(statime time.Time, endtime time.Time) checkout.CheckoutResponse {
	return checkout.GetDaydata(statime, endtime)
}

func (a *App) SaveCheackToExcel() {
	checkout.Checkout(a.ctx)
}
