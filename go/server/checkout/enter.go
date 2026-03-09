package checkout

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	time2 "order_server/go/utils/time"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type Spec struct {
	Name         string  `json:"name"`
	PricePerKg   float64 `json:"pricePerKg"`   // 每斤/元（用户输入）
	PricePerItem float64 `json:"pricePerItem"` // 每件/元（用户输入，可选）
	Total        int     `json:"total"`        // 总件数
}
type ProSPecs struct {
	SID   uint
	Name  string `json:"Name"`
	Specs []Spec
}

type CheckoutResponse struct {
	FieldList       []string            // 未分类字段
	PBMap           map[string][]string // 商品归属map
	Count           uint                // 查询的条数
	ProSPecs        []ProSPecs          // 商品规格及分类.及表头
	TableData       []string            // 表格数据
	TotalRow        []int               // 总件数行
	PricePerKgRow   []float64           // 每件/元 行（用户输入）
	PricePerItemRow []float64           // 每件/元 行
	TotalAmountRow  []float64           // 总金额行
}

type ProPriceRequest struct {
	ProductName  string  `json:"productName"`  // 商品: 香水柠檬
	Spec         string  `json:"spec"`         // 规格 : 5斤
	Total        int     `json:"total"`        // 总件数
	PricePerKg   float64 `json:"pricePerKg"`   //每斤/元
	PricePerItem float64 `json:"pricePerItem"` // 每件/元
	TotalAmount  float64 `json:"totalAmount"`  // 总金额
}

type CheackDay struct {
	Date  time.Time
	specs []ProSPecs
}

var checkoutresp = CheckoutResponse{}           // 响应副本
var gspecsMap = make(map[string]map[string]int) // map[商品规格名称]map[商品规格]总件数
var proPrice = []ProPriceRequest{}
var addresinfolist = []models.AddressInfo{}

// 添加结账数据
func AddcheckoutData(sm *SpecManager, list []models.AddressInfo) {

	var order = &models.Order{}
	// 处理数据,获取总件数
	for _, info := range addresinfolist {
		for _, spec := range info.ProductSpecs {
			for k, v := range gspecsMap {
				if spec.Name == k {
					for kk, vv := range v {
						if spec.Weight == ordertutil.GetNumberInSpec(kk) {
							order.Quantity = vv
							order.Date = info.CreatedAt
							order.ProductName = spec.Name
							order.Spec = fmt.Sprintf("%d斤", spec.Weight)
						}
					}
				}
			}
		}
	}

	// 获取商品单价
	for _, info := range addresinfolist {
		for _, spec := range info.ProductSpecs {
			for _, pr := range proPrice {
				if pr.ProductName == spec.Name {
					order.PricePerKg = pr.PricePerKg
					order.PricePerItem = pr.PricePerItem
				}
			}
		}
	}

	sm.AddOrder(*order)

}

func Checkout(timedate time.Time) {
	fmt.Println("========== 商品销售统计Excel生成器 ==========")

	// 创建规格管理器
	sm := NewSpecManager()

	// 添加数据
	AddcheckoutData(sm, addresinfolist)
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
	addresinfolist = _list
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

	// 获取商品分类及规格，同时计算该分类的总件数 int 键为商品分类名称，值为商品规格 map[string]int   ==》 商品规格为键，总件数为值
	specsMap := make(map[string]map[string]int)
	for _, info := range _list {
		for _, spec := range info.ProductSpecs { // 得到这订单所有的商品规格
			wg := fmt.Sprintf("%d斤", spec.Weight)
			if _, ok := specsMap[spec.Name]; !ok {
				specsMap[spec.Name] = make(map[string]int)
			}
			specsMap[spec.Name][wg] += 1
		}
	}
	gspecsMap = specsMap

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
			// 这里可以从全局变量或数据库中获取价格信息
			// 暂时使用默认值
			speclist = append(speclist, Spec{
				Name:         vv,
				PricePerKg:   0, // 默认为0，需要用户填写
				PricePerItem: 0, // 默认为0，需要用户填写
			})
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
			specStr := fmt.Sprintf("%d斤", spec.Weight)
			if _, ok := dateMap[date][productName]; !ok {
				dateMap[date][productName] = make(map[string]int)
			}
			dateMap[date][productName][specStr]++
		}
	}

	// 获取表格数据
	var tableData []string
	var _totalRowlist = make(map[string][]int, 0)
	var Rowlen int
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

		var rowlen int
		// 按照 ProSPecsList 的顺序和每个商品的规格顺序生成数据
		for _, product := range ProSPecsList {
			for _, spec := range product.Specs {
				if countVal, ok := productMap[product.Name][spec.Name]; ok {
					_totalRowlist[date] = append(_totalRowlist[date], countVal)
					rowData += fmt.Sprintf(",%d", countVal)
				} else {
					_totalRowlist[date] = append(_totalRowlist[date], 0)
					rowData += ",—"
				}
				rowlen++
			}
		}
		Rowlen = rowlen
		tableData = append(tableData, rowData)
	}

	// 获取总件数表格数据
	var totalRow = make([]int, Rowlen)
	for _, v := range _totalRowlist {
		for i, i2 := range v {
			totalRow[i] += i2
		}

	}

	// 构建响应
	resp := CheckoutResponse{
		FieldList: newSpecList,
		Count:     uint(len(_list)),
		PBMap:     global.ProductBelongMap,
		ProSPecs:  ProSPecsList,
		TableData: tableData,
		TotalRow:  totalRow,
	}
	checkoutresp = resp
	return resp
}

// 设置商品归属
func SetFieldList(mapp map[string][]string) {
	global.ProductBelongMap = mapp
	core.WriteProductBelongMap(mapp)
}

// 设置金额
func SetPrice(_list []ProPriceRequest) CheckoutResponse {

	fmt.Println("_list", _list)
	var pricePerkgRow, pricePerItemRow, totalAmountRow []float64
	// 获取该商品 ->规格的总件数
	for i := range _list { // 使用索引修改值
		for k, v := range gspecsMap {
			if _list[i].ProductName == k { // 定位商品
				for spec, total := range v {
					if spec == _list[i].Spec { // 定位规格
						_list[i].Total = total
						// 获取规格的斤数
						re, err := regexp.Compile(`\d+`)
						if err != nil {
							logrus.Errorf("设置金额获取规格的斤数error", err)
						}
						number := re.FindString(spec)
						num, err := strconv.Atoi(number)
						if err != nil {
							logrus.Errorf("设置金额转换error", err)
						}

						if _list[i].PricePerItem == 0.00 {
							_list[i].PricePerItem = _list[i].PricePerKg * float64(num) //每斤/元  * 斤 = 每件/元
						}

						// 计算总金额
						_list[i].TotalAmount = float64(_list[i].Total) * _list[i].PricePerItem
					}
				}
			}
		}
	}

	// 给副本响应数据赋值
	for i := range checkoutresp.ProSPecs { // 使用索引修改值
		for _, item := range _list { //
			if checkoutresp.ProSPecs[i].Name == item.ProductName { // 定位商品
				for j, spec := range checkoutresp.ProSPecs[i].Specs { // 遍历商品规格
					if spec.Name == item.Spec { // 定位规格
						checkoutresp.ProSPecs[i].Specs[j].PricePerKg = item.PricePerKg
						checkoutresp.ProSPecs[i].Specs[j].PricePerItem = item.PricePerItem

						pricePerkgRow = append(pricePerkgRow, item.PricePerKg)
						pricePerItemRow = append(pricePerItemRow, item.PricePerItem)
						totalAmountRow = append(totalAmountRow, item.TotalAmount)
					}

				}
			}
		}
	}
	fmt.Println("pricePerkgRow", pricePerkgRow)
	fmt.Println("pricePerItemRow", pricePerItemRow)
	fmt.Println("totalAmountRow", totalAmountRow)
	//设置返回值
	checkoutresp.PricePerKgRow = pricePerkgRow
	checkoutresp.PricePerItemRow = pricePerItemRow
	checkoutresp.TotalAmountRow = totalAmountRow
	fmt.Println("checkoutresp", checkoutresp)
	// 为execl添加数据
	return checkoutresp
}
