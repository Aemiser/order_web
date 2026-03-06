package global

import (
	"order_server/go/conf"

	"gorm.io/gorm"
)

const (
	Version = "10.0.1"
)

var (
	Config *conf.Config
	Db     *gorm.DB
)
