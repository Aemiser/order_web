package global

import (
	"order_server/go/conf"
	"order_server/go/models"

	"gorm.io/gorm"
)

const (
	Version = "10.0.1"
)

var (
	Config      *conf.Config
	Db          *gorm.DB
	AddressList []models.AddressInfo
)
