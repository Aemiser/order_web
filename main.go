package main

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/flags"
	"order_server/go/global"
	"order_server/go/router"

	"github.com/sirupsen/logrus"
)

func main() {
	flags.Parse()
	global.Config = core.ReadConf()
	core.InitLogrus()
	global.Db = core.InitDB()
	global.ProductBelongMap = core.ReadProductBelongMap()
	global.AiMap = core.ReadAiMap()
	global.SystemPrompt = core.Readsystemprompt()
	flags.Run()

	port := global.Config.Server.Port
	if port == 0 {
		port = 8080
	}
	addr := fmt.Sprintf(":%d", port)

	logrus.Infof("HTTP 服务启动: %s", addr)
	if err := router.Setup().Run(addr); err != nil {
		logrus.Fatalf("HTTP 服务启动失败: %v", err)
	}
}
