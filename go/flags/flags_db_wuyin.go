package flags

import (
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/server/wuyin"

	"github.com/sirupsen/logrus"
)

// FlagDBWuyin 吴饮独立库迁移（`go run main.go -dbwuyin`）。
// 与一点点的 -db（FlagDB）分开：两库独立演进，互不强制同时迁移。
func FlagDBWuyin() {
	if global.DbWuyin == nil {
		logrus.Errorf("吴饮数据库未配置（settings.yaml dbWuyin），无法迁移")
		return
	}
	err := global.DbWuyin.AutoMigrate(
		models.WuyinAddressInfo{},
		models.WuyinTagConfig{},
		models.WuyinPriceConfig{},
	)
	if err != nil {
		logrus.Errorf("吴饮数据库迁移失败 %s", err)
		return
	}
	logrus.Info("吴饮数据库迁移成功")

	// 迁移后加载标签配置（空表会播种默认仓库标签）
	if err := wuyin.LoadTagConfigs(); err != nil {
		logrus.Errorf("吴饮标签配置加载失败 %s", err)
		return
	}
	logrus.Infof("吴饮标签配置加载完成，共 %d 个", len(global.WuyinTagConfigs))
}
