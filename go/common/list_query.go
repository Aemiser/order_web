package common

import (
	"fmt"
	"order_server/go/global"

	"gorm.io/gorm"
)

type PageInfo struct {
	Limit int    `form:"limit" json:"limit"`
	Page  int    `form:"page" json:"page"`
	Key   string `form:"key" json:"key"`
	Order string `form:"order" json:"order"`
}

func (p PageInfo) GetLimit() int {
	if p.Limit < 0 {
		p.Limit = 1
	}
	return p.Limit
}

func (p PageInfo) GetPage() int {
	if p.Page < 0 {
		p.Page = 1
	}
	return p.Page
}

func (p PageInfo) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

type Options struct {
	PageInfo     PageInfo
	Likes        []string // 关键字模糊查询的列
	Preloads     []string
	Where        *gorm.DB
	Debug        bool
	DefaultOrder string
}

func ListQuery[T any](model T, options Options) (list []T, count int, err error) {
	var query = global.Db.Model(model).Where(model)

	// 显示日志
	if options.Debug {
		query = query.Debug()
	}

	// 模糊查询
	if len(options.Likes) > 0 && options.PageInfo.Key != "" {
		fmt.Println("关键词模糊查询列:", options.Likes)
		likes := global.Db.Where("")
		for _, column := range options.Likes {
			likes.Or(
				fmt.Sprintf("%s like ?", column),
				fmt.Sprintf("%%%s%%", options.PageInfo.Key),
			)
		}
		query = query.Where(likes)
	}

	// 定制化查询
	if options.Where != nil {
		query = query.Where(options.Where)
	}

	// 预加载
	for _, preload := range options.Preloads {
		query = query.Preload(preload)
	}

	// 查总数
	var _count int64
	query.Count(&_count)
	count = int(_count)

	// 分页
	offest := options.PageInfo.GetOffset()
	limit := options.PageInfo.GetLimit()

	// 排序
	if options.PageInfo.Order != "" {
		query = query.Order(options.PageInfo.Order)
	} else {
		if options.DefaultOrder != "" {
			query = query.Order(options.DefaultOrder)
		}
	}

	// 排除limit 为0的情况
	if limit != 0 {
		query = query.Offset(offest).Limit(limit)
	}
	err = query.Find(&list).Error
	return
}
