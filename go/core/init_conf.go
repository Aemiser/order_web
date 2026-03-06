package core

import (
	"fmt"
	"order_server/go/conf"
	"order_server/go/flags"
	"order_server/go/global"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func ReadConf() (c *conf.Config) {
	byteData, err := os.ReadFile(flags.FlagOptions.File)
	if err != nil {
		panic(err)
	}
	c = new(conf.Config)
	err = yaml.Unmarshal(byteData, &c)
	if err != nil {
		panic(fmt.Sprintf("yaml配置文件格式错误 %s \n", err))
	}

	fmt.Printf("读取配置文件 %s 成功\n", flags.FlagOptions.File)
	return
}

func WriteConf() {
	byteDate, err := yaml.Marshal(global.Config)
	if err != nil {
		logrus.Errorf("yaml配置文件格式错误 %s \n", err)
	}
	err = os.WriteFile(flags.FlagOptions.File, byteDate, 0666)
	if err != nil {
		logrus.Errorf("写入配置文件 %s 失败 \n", flags.FlagOptions.File)
	}
	logrus.Infof("写入配置文件 %s 成功\n", flags.FlagOptions.File)
}
