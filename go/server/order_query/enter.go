package order_query

import (
	"order_server/go/common"
	"order_server/go/models"

	"github.com/sirupsen/logrus"
)

type OrderQueryRequest struct {
	common.PageInfo
	addresssinfo models.AddressInfo
}

func OrderQuery(oq OrderQueryRequest) (list []models.AddressInfo) {
	_list, _, err := common.ListQuery(oq, common.Options{
		PageInfo:     oq.PageInfo,
		Preloads:     []string{"ProductSpecsJSON"},
		DefaultOrder: "created_at desc",
	})

	if err != nil {
		logrus.Errorf("查询失败 %s", err)
		return nil
	}

	for _, item := range _list {
		list = append(list, item.addresssinfo)
	}
	return

}
