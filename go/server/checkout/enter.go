package checkout

import (
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	time2 "order_server/go/utils/time"
	"time"
)

func Checkout(timedate time.Time) {
	fmt.Println("========== 商品销售统计Excel生成器 ==========")

	// 创建规格管理器
	sm := NewSpecManager()

	// 添加数据
	Getdata(sm, timedate)
	// 打印统计摘要
	sm.PrintSummary()

	// 生成Excel文件
	fmt.Println("\n========== 生成Excel文件 ==========")

	// 生成带时间戳的文件名
	filename := fmt.Sprintf("商品销售统计_%s.xlsx", time.Now().Format("2006-01-02_150405"))

	err := sm.GenerateExcel(filename)
	if err != nil {
		fmt.Printf("❌ 生成Excel失败: %v\n", err)
	} else {
		fmt.Printf("✅ Excel生成成功！文件: %s\n", filename)
		fmt.Printf("📁 文件保存在当前目录下\n")
	}

	// 显示一些统计信息
	fmt.Println("\n========== 统计信息 ==========")
	fmt.Printf("总订单数: %d\n", len(sm.Orders))
	fmt.Printf("总商品种类: %d\n", len(sm.ProductMap))
	fmt.Printf("总销售天数: %d\n", len(sm.DateMap))

	// 显示所有商品
	fmt.Println("\n📦 商品列表:")
	for _, product := range sm.ProductMap {
		fmt.Printf("  - %s (共%d种规格, 总销量:%d)\n",
			product.Name, len(product.Specs), product.TotalQty)
	}

	// 显示所有日期
	fmt.Println("\n📅 销售日期:")
	dates := make([]string, 0, len(sm.DateMap))
	for date := range sm.DateMap {
		dates = append(dates, date)
	}
	for _, date := range dates {
		fmt.Printf("  - %s\n", date)
	}
}

func Getdata(sm *SpecManager, timedate time.Time) {
	// 起始日期
	begindate := time.Date(timedate.Year(), timedate.Month(), 1, 0, 0, 0, 0, time.Local)
	// 截至日期
	enddate := time.Date(timedate.Year(), timedate.Month(), time2.GetDaysInMonth(timedate.Year(), timedate.Month()), 23, 59, 59, 0, time.Local)
	// 查询数据库获取该月数据
	var _list []models.AddressInfo
	global.Db.Debug().Model(&_list).Preload("ProductSpecs").Where("created_at >= ? AND created_at <= ?", begindate, enddate).Find(&_list)
	fmt.Println(_list)

	// 统计该月每天的订单数，以及漏统计的日期
	var monthdate = make(map[string]int, 0)
	for _, info := range _list {
		if _, ok := monthdate[info.CreatedAt.Format("2006-01-02")]; ok {
			monthdate[info.CreatedAt.Format("2006-01-02")]++
		} else {
			monthdate[info.CreatedAt.Format("2006-01-02")] = 1
		}
	}

	for k, v := range monthdate {
		fmt.Println(k, v)
	}

	var productMap = map[string][]string{
		"香水柠檬":  []string{"香水", "香柠"},
		"广香一级":  []string{"广香"},
		"青柠尤力克": []string{"青柠", "尤力克"},
		"黄柠檬":   []string{"安岳黄柠檬", "安岳柠檬"},
	}

	for _, info := range _list {
		for _, spec := range info.ProductSpecs {
			// 数据预处理
			for k, v := range productMap {
				for _, vv := range v {
					if spec.Name == vv {
						spec.Name = k
					}
				}
			}

			sm.AddOrder(Order{
				Date:        info.CreatedAt,
				ProductName: spec.Name,
				Spec:        fmt.Sprintf("%d 斤", spec.Weight),
				Quantity:    1,
				Unit:        "斤",
			})
		}
	}

}
