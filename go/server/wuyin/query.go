package wuyin

import (
	"fmt"
	"order_server/go/common"
	"order_server/go/global"
	"order_server/go/models"
	wuyinutil "order_server/go/utils/wuyin"
	"time"
)

// WuyinOrderQueryRequest 吴饮订单查询入参：在一点点 OrderQueryRequest 基础上
// 增加 Tags 多选（IN 过滤）；单 Tag 也可经结构体字段等值过滤，两者并存时取交集。
type WuyinOrderQueryRequest struct {
	common.PageInfo
	WuyinAddressInfo models.WuyinAddressInfo `json:"addressInfo"`
	Starttime        time.Time               `json:"startTime"`
	Endtime          time.Time               `json:"endTime"`
	Tags             []string                `json:"tags"`
}

// WuyinOrderQueryResponse 吴饮订单查询出参
type WuyinOrderQueryResponse struct {
	List  []wuyinutil.WuyinExceladdressInfo `json:"list"`
	Count int                               `json:"count"`
}

// wuyinListQuery 吴饮分页查询。与 common.ListQuery 同构但绑定 global.DbWuyin
// （common 版写死 global.Db，且一点点链路约束不动，故此处平行实现），
// 并追加 tags IN 过滤。
func wuyinListQuery(oq WuyinOrderQueryRequest) (list []models.WuyinAddressInfo, count int, err error) {
	query := global.DbWuyin.Model(&models.WuyinAddressInfo{}).Where(&oq.WuyinAddressInfo)

	// 关键词模糊查询（列名与一点点保持一致的蛇形实际列名口径）
	if len(oq.Key) > 0 {
		likes := global.DbWuyin.Where("")
		for _, column := range []string{"recipient", "address", "phone", "product_specs", "remarks", "province", "tag", "sender"} {
			likes.Or(fmt.Sprintf("%s like ?", column), fmt.Sprintf("%%%s%%", oq.Key))
		}
		query = query.Where(likes)
	}

	if !(oq.Starttime.IsZero() && oq.Endtime.IsZero()) {
		query = query.Where("created_at between ? and ?", oq.Starttime, oq.Endtime)
	}

	// tag 多选（侧边栏点选若干标签）
	if len(oq.Tags) > 0 {
		query = query.Where("tag IN ?", oq.Tags)
	}

	var _count int64
	query.Count(&_count)
	count = int(_count)

	order := oq.Order
	if order == "" {
		order = "created_at"
	}
	query = query.Order(order)

	if limit := oq.GetLimit(); limit != 0 {
		query = query.Offset(oq.GetOffset()).Limit(limit)
	}
	err = query.Find(&list).Error
	return
}

// WuyinOrderQuery 吴饮订单分页查询
func WuyinOrderQuery(oq WuyinOrderQueryRequest) (resp WuyinOrderQueryResponse, err error) {
	_list, count, err := wuyinListQuery(oq)
	if err != nil {
		return resp, fmt.Errorf("查询失败: %w", err)
	}
	resp = WuyinOrderQueryResponse{
		List:  wuyinutil.WuyinToExcelAddress(_list),
		Count: count,
	}
	return
}

// WuyinTagStat 标签订单数统计（详情页侧边栏）
type WuyinTagStat struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// WuyinTagStats 按标签统计订单数（时间范围可选）。未分配也计入（tag 为空串的行归入
// DefaultTag），侧边栏据此渲染各标签的量级。
func WuyinTagStats(start, end time.Time) ([]WuyinTagStat, error) {
	query := global.DbWuyin.Model(&models.WuyinAddressInfo{})
	if !(start.IsZero() && end.IsZero()) {
		query = query.Where("created_at between ? and ?", start, end)
	}

	type row struct {
		Tag   string
		Total int
	}
	var rows []row
	if err := query.Select("COALESCE(NULLIF(tag, ''), ?) AS tag, COUNT(*) AS total", DefaultTag).
		Group("tag").Order("tag").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计失败: %w", err)
	}

	stats := make([]WuyinTagStat, 0, len(rows))
	for _, r := range rows {
		stats = append(stats, WuyinTagStat{Tag: r.Tag, Count: r.Total})
	}
	return stats, nil
}
