package flags

import (
	"order_server/go/global"
	"order_server/go/models"

	"github.com/sirupsen/logrus"
)

func FlagDB() {
	err := global.Db.AutoMigrate(
		models.AddressInfo{},
		models.LogModels{},
	)
	if err != nil {
		logrus.Warnf("数据库迁移失败 %s", err)
	}
	logrus.Info("数据库迁移成功")

}
