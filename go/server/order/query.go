package order

import (
	"fmt"
	"order_server/go/common"
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	"time"

	"github.com/sirupsen/logrus"
)

type OrderQueryRequest struct {
	common.PageInfo
	Addresssinfo models.AddressInfo
	Starttime    time.Time `json:"startTime"`
	Endtime      time.Time `json:"endTime"`
}

type OrderQueryResponse struct {
	List  []ordertutil.ExceladdressInfo `json:"list"`
	Count int                           `json:"count"`
}

func OrderQuery(oq OrderQueryRequest) (resp OrderQueryResponse, err error) {

	var addresslist = oq.Addresssinfo
	fmt.Println("参数列表：", addresslist)
	fmt.Println("参数列表page：", oq.Page)
	fmt.Println("参数列表limit：", oq.Limit)
	var option = common.Options{
		PageInfo:     oq.PageInfo,
		DefaultOrder: "created_at",
		// Likes 必须与数据库实际列名逐字符一致（B3.2，2026-10-17 实测修正）：
		// product_specs 为实际蛇形列名——GORM 对 tag `gorm:"productSpecs;type:json"` 中
		// 无 column: 前缀的段不识别，列名按默认蛇形命名生成（详见 backend-architecture B3.2.1 存量登记）；
		// province 为 T10 新增列（tag 显式 column:province）。
		Likes: []string{"recipient", "address", "phone", "product_specs", "remarks", "province"},
		Debug: true,
		Where: nil,
	}
	// 判断时间范围
	if !(oq.Starttime.IsZero() && oq.Endtime.IsZero()) {
		// 初始化 Where 字段
		option.Where = global.Db.Where("")
		option.Where.Where("created_at between ? and ?", oq.Starttime, oq.Endtime)

	}

	_list, count, err := common.ListQuery(addresslist, option)
	if err != nil {
		// B2 不吞错（契约 §3.5 变更4）：查询失败必须上抛 50000，不再静默返回空列表
		logrus.Errorf("查询失败 %s", err)
		return
	}

	list := ordertutil.AddressToExcelAddress(_list)
	resp = OrderQueryResponse{
		List:  list,
		Count: count,
	}
	for _, info := range list {
		fmt.Println(info)
	}
	return
}
