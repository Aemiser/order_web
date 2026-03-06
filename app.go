package main

import (
	"context"
	"fmt"
	"order_server/go/models"
	"order_server/go/server/config"
	"order_server/go/server/order_query"
	"order_server/go/server/order_recognition"
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
func (a *App) OrderRecognition(context string) {
	order_recognition.OrderRec(context)
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
func (a *App) OrderQuery(oq order_query.OrderQueryRequest) []models.AddressInfo {
	return order_query.OrderQuery(oq)
}
