package main

import (
	"context"
	"fmt"
	"order_server/go/models"
	"order_server/go/models/enum"
	"order_server/go/server/config"
	"order_server/go/server/order"
	ordertutil "order_server/go/utils/order"
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
func (a *App) OrderQuery(oq order.OrderQueryRequest) []models.AddressInfo {
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
