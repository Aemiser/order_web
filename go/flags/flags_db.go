package flags

import (
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/server/order"

	"github.com/sirupsen/logrus"
)

// FlagDB 数据库迁移唯一入口（B3.1）。仅 `go run main.go -db` 触发。
// 依赖方向：flags → server/order 为单向依赖、无环（server/order 不引用 flags/core，B5 允许 flags 用 global.Db）。
func FlagDB() {
	err := global.Db.AutoMigrate(
		models.AddressInfo{},
		models.LogModels{},
		models.PriceConfig{}, // 2026-10-17 新增：价格配置表（契约 §1.3）
		models.User{},        // 2026-10-18 新增：登录账号表
	)
	if err != nil {
		// B2 不吞错：迁移失败必须中止，不再假成功（原实现失败后仍打印「迁移成功」）
		logrus.Errorf("数据库迁移失败 %s", err)
		return
	}
	logrus.Info("数据库迁移成功")

	// RUL-2：迁移成功后自动幂等回填省份存量数据（仅 province 为空的行，可重复执行）
	updated, remaining, err := order.BackfillProvince()
	if err != nil {
		logrus.Errorf("省份存量回填失败 %s", err)
		return
	}
	logrus.Infof("省份存量回填完成：回填 %d 行，仍为空 %d 行", updated, remaining)
}
