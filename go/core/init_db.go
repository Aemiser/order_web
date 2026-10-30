package core

import (
	"fmt"
	"order_server/go/global"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func InitDB() *gorm.DB {
	if len(global.Config.DB) == 0 {
		logrus.Fatalf("数据库配置为空")
		return nil
	}

	dc := global.Config.DB[0] //读
	//TODO pgsql的支持

	db, err := gorm.Open(mysql.Open(dc.GetDSN()), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, //不生成外键约束
	})
	if err != nil {
		logrus.Fatalf("数据库连接失败: %v", err)
	}
	sqlDB, err := db.DB()
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量。
	sqlDB.SetMaxIdleConns(dc.MaxIdleConns)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(dc.MaxOpenConns)
	// SetConnMaxLifetime 设置了可以重新使用连接的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
	logrus.Infof("数据库连接成功")

	// 读写库存在，则配置读写分离
	if len(global.Config.DB) > 1 {
		fmt.Println("读写库存在")
		readList := []gorm.Dialector{}
		for _, d := range global.Config.DB[1:] {
			readList = append(readList, mysql.Open(d.GetDSN()))
		}
		db.Use(dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{mysql.Open(dc.GetDSN())}, // 写
			Replicas: readList,                                  // 读
			Policy:   dbresolver.RandomPolicy{},
		}))
	}
	return db
}

// InitWuyinDB 吴饮独立数据库连接（settings.yaml dbWuyin）。
// 未配置时返回 nil 且不 Fatal——吴饮是可选品牌，缺失只禁用 /api/wuyin/* 端点，
// 不能拖垮一点点主链路的启动（与 InitDB 的 Fatal 语义刻意不同）。
func InitWuyinDB() *gorm.DB {
	dw := global.Config.DbWuyin
	if dw.Empty() {
		logrus.Infof("吴饮数据库未配置（dbWuyin），/api/wuyin 端点将不可用")
		return nil
	}

	db, err := gorm.Open(mysql.Open(dw.GetDSN()), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, //不生成外键约束
	})
	if err != nil {
		logrus.Fatalf("吴饮数据库连接失败: %v", err)
	}
	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(dw.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dw.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	logrus.Infof("吴饮数据库连接成功")
	return db
}
