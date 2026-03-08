package checkout

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	time2 "order_server/go/utils/time"
	"time"
)

type Spec struct {
	Name         string  `json:"name"`
	PricePerKg   float64 `json:"pricePerKg"`   // 每斤/元（用户输入）
	PricePerItem float64 `json:"pricePerItem"` // 每件/元（用户输入，可选）

}
type ProSPecs struct {
	SID   uint
	Name  string `json:"Name"`
	Specs []Spec
}

type CheckoutResponse struct {
	FieldList []string            // 未分类字段
	Count     uint                // 查询的条数
	ProSPecs  []ProSPecs          // 商品规格及分类.及表头
	TableData []string            // 表格数据
	PBMap     map[string][]string // 商品归属map
}

func Checkout(timedate time.Time) {
	fmt.Println("========== 商品销售统计Excel生成器 ==========")

	// 创建规格管理器
	sm := NewSpecManager()

	// 添加数据
	fmt.Println(sm.Orders)
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

// 获取指定月份数据
func GetMonthdata(timedate time.Time) {
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

}

// 获取指定日期的数据
func GetDaydata(statime time.Time, endtime time.Time) CheckoutResponse {
	// 起始日期
	begindate := time.Date(statime.Year(), statime.Month(), statime.Day(), 0, 0, 0, 0, time.Local)
	// 截至日期
	enddate := time.Date(endtime.Year(), endtime.Month(), endtime.Day(), 23, 59, 59, 0, time.Local)

	// 查询数据库获取日期数据
	var _list []models.AddressInfo
	var count int64
	global.Db.Debug().Model(&_list).Count(&count).Where("created_at >= ? AND created_at <= ?", begindate, enddate).Find(&_list)

	// 检查是否存在新商品
	newSpecList := ordertutil.Specchecking(_list)

	// 对列表的商品规格进行统一的整理，使其只存在第一元素
	for i := range _list {
		for j := range _list[i].ProductSpecs {
			spec := &_list[i].ProductSpecs[j] // 获取商品规格的指针，确保修改能够生效
			for k, v := range global.ProductBelongMap {
				if spec.Name == k { // 如果商品规格名称等于商品分类第一元素，则跳过
					continue
				}

				for _, vv := range v { // 遍历商品分类子字段
					if spec.Name == vv { // 如果商品分类子字段等于商品分类子字段，则将其赋值为第一元素
						spec.Name = k
						// 找到匹配后跳出内层循环，避免重复修改
						break
					}
				}
			}
		}
	}

	// 获取商品分类及规格
	specsMap := make(map[string]map[string]bool)
	for _, info := range _list {
		for _, spec := range info.ProductSpecs { // 得到这订单所有的商品规格
			wg := fmt.Sprintf("%d 斤", spec.Weight)
			if _, ok := specsMap[spec.Name]; !ok {
				specsMap[spec.Name] = make(map[string]bool)
			}
			specsMap[spec.Name][wg] = true
		}
	}

	// 转换为切片格式
	ProSPecsMap := make(map[string][]string)
	for k, v := range specsMap {
		for spec := range v {
			ProSPecsMap[k] = append(ProSPecsMap[k], spec)
		}
	}

	// Map转List
	var ProSPecsList []ProSPecs
	sid := uint(1)
	for k, v := range ProSPecsMap {
		var speclist = make([]Spec, 0)
		for _, vv := range v {
			speclist = append(speclist, Spec{Name: vv})
		}
		ppcs := ProSPecs{
			SID:   sid,
			Name:  k,
			Specs: speclist,
		}

		ProSPecsList = append(ProSPecsList, ppcs)
		sid++
	}

	// 获取每天的总商品规格
	// 按日期分组
	dateMap := make(map[string]map[string]map[string]int)
	for _, info := range _list {
		date := info.CreatedAt.Format("2006-01-02")
		if _, ok := dateMap[date]; !ok {
			dateMap[date] = make(map[string]map[string]int)
		}
		for _, spec := range info.ProductSpecs {
			productName := spec.Name
			specStr := fmt.Sprintf("%d 斤", spec.Weight)
			if _, ok := dateMap[date][productName]; !ok {
				dateMap[date][productName] = make(map[string]int)
			}
			dateMap[date][productName][specStr]++
		}
	}

	// 获取表格数据
	var tableData []string

	// 生成表格数据
	for date, productMap := range dateMap {
		// 转换日期格式为 "3月8日" 格式
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			continue
		}
		dateStr := fmt.Sprintf("%d月%d日", t.Month(), t.Day())

		// 构建行数据
		rowData := dateStr

		// 按照 ProSPecsList 的顺序和每个商品的规格顺序生成数据
		for _, product := range ProSPecsList {
			for _, spec := range product.Specs {
				if productMap[product.Name] != nil {
					if countVal, ok := productMap[product.Name][spec]; ok {
						rowData += fmt.Sprintf(",%d", countVal)
					} else {
						rowData += ",—"
					}
				} else {
					rowData += ",—"
				}
			}
		}
		tableData = append(tableData, rowData)
	}

	// 构建响应
	resp := CheckoutResponse{
		FieldList: newSpecList,
		Count:     uint(len(_list)),
		PBMap:     global.ProductBelongMap,
		ProSPecs:  ProSPecsList,
		TableData: tableData,
	}

	return resp
}

// 设置商品归属
func SetFieldList(mapp map[string][]string) {
	global.ProductBelongMap = mapp
	core.WriteProductBelongMap(mapp)
}

// 添加结账数据
func AddcheckoutData(sm *SpecManager, list []models.AddressInfo) {

	for _, info := range list {
		for _, spec := range info.ProductSpecs {
			// 数据预处理
			for k, v := range global.ProductBelongMap {
				for _, vv := range v {
					if spec.Name == vv {
						spec.Name = k
					}
				}
			}

			sm.AddOrder(models.Order{
				ID:          info.ID,
				Date:        info.CreatedAt,
				ProductName: spec.Name,
				Spec:        fmt.Sprintf("%d 斤", spec.Weight),
				Quantity:    1,
			})
		}
	}

}
