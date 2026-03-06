package config

import (
	"order_server/go/conf"
	"order_server/go/core"
	"order_server/go/global"
)

type ReConf struct {
	Deepseek conf.Deepseek `json:"deepseek"`
	SavePath conf.SavePath `json:"savePath"`
}

func SetConf(conf ReConf) {
	global.Config.Deepseek = conf.Deepseek
	global.Config.SavePath = conf.SavePath
	core.WriteConf()
}

func GetConf() ReConf {
	var cf = ReConf{
		Deepseek: global.Config.Deepseek,
		SavePath: global.Config.SavePath,
	}
	return cf
}
