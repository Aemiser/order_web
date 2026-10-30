package main

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/flags"
	"order_server/go/global"
	"order_server/go/router"
	"order_server/go/server/wuyin"

	"github.com/sirupsen/logrus"
)

func main() {
	flags.Parse()
	global.Config = core.ReadConf()
	core.InitLogrus()
	global.Db = core.InitDB()
	global.DbWuyin = core.InitWuyinDB()
	global.ProductBelongMap = core.ReadProductBelongMap()
	global.AiMap = core.ReadAiMap()
	global.SystemPrompt = core.Readsystemprompt()
	global.SystemPromptWuyin = core.ReadsystempromptWuyin()
	flags.Run()
	// 吴饮标签缓存：放在 flags.Run() 之后——-dbwuyin 首次建表前表还不存在，
	// 先加载只会在日志里留一条 Error 1146 告警（-dbwuyin 分支建表后会自行加载）
	if global.DbWuyin != nil {
		if err := wuyin.LoadTagConfigs(); err != nil {
			logrus.Warnf("吴饮标签配置加载失败: %v", err)
		}
	}

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
