package order

import (
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	"time"
)

// ProvinceStat 省份统计项（契约 §3.1）。
type ProvinceStat struct {
	Name      string `json:"name"`      // 省份全称（存储值，34 行政区枚举）
	ShortName string `json:"shortName"` // 省份简称（utils/order provinceTable 映射填充，不入库）
	Value     int    `json:"value"`     // 订单数
}

// ProvinceStatsResponse 省份统计响应（契约 §3.1）。
type ProvinceStatsResponse struct {
	List  []ProvinceStat `json:"list"`  // 按 value 降序、同值按 name 升序；空为 null
	Total int            `json:"total"` // Σvalue（province 非空订单总数）
}

// ProvinceStats 省份统计：按 province 分组计数，排除空值（GORM 自动过滤已软删行）。
// start/end 均为零值时全量统计；否则按 created_at 闭区间过滤（与 /order/query 时间语义一致，
// handler 层保证两参数成对传入）。统计口径为 address_infos 行数（非商品件数）。
func ProvinceStats(start, end time.Time) (ProvinceStatsResponse, error) {
	var rows []struct {
		Name  string
		Value int64
	}

	q := global.Db.Model(&models.AddressInfo{}).
		Select("province AS name, count(*) AS value").
		Where("province <> '' AND province IS NOT NULL")
	if !start.IsZero() && !end.IsZero() {
		q = q.Where("created_at between ? and ?", start, end)
	}

	if err := q.Group("province").
		Order("value DESC, name ASC").
		Find(&rows).Error; err != nil {
		return ProvinceStatsResponse{}, err
	}

	resp := ProvinceStatsResponse{List: nil}
	for _, r := range rows {
		resp.List = append(resp.List, ProvinceStat{
			Name:      r.Name,
			ShortName: ordertutil.ProvinceShortName(r.Name),
			Value:     int(r.Value),
		})
		resp.Total += int(r.Value)
	}
	return resp, nil
}
