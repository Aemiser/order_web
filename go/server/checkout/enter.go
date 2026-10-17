package checkout

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	time2 "order_server/go/utils/time"
	"path/filepath"
	"regexp"
	"sort"
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
var begtime2, endtime2 string

// 添加结账数据
func AddcheckoutData(sm *SpecManager) {
	fmt.Println("====================")
	// 处理数据,获取总件数
	fmt.Println("处理数据:", addresinfolist)
	fmt.Println("====================")
	var orderlist []models.Order

	// 按日期分组统计销售数据
	dateProductSpecCount := make(map[string]map[string]map[string]int)
	for _, info := range addresinfolist {
		date := info.CreatedAt.Format("2006-01-02")
		if _, ok := dateProductSpecCount[date]; !ok {
			dateProductSpecCount[date] = make(map[string]map[string]int)
		}

		for _, spec := range info.ProductSpecs {
			productName := spec.Name
			specName := fmt.Sprintf("%d斤", spec.Weight)

			if _, ok := dateProductSpecCount[date][productName]; !ok {
				dateProductSpecCount[date][productName] = make(map[string]int)
			}
			dateProductSpecCount[date][productName][specName]++
		}
	}

	// 生成按日期分组的订单列表
	for date, productSpecMap := range dateProductSpecCount {
		fmt.Printf("\n日期: %s\n", date)
		fmt.Println("商品销售数据:")

		for productName, specMap := range productSpecMap {
			for specName, count := range specMap {
				// 查找对应的价格数据
				var pricePerKg, pricePerItem float64
				for _, pr := range proPrice {
					if pr.ProductName == productName && pr.Spec == specName {
						pricePerKg = pr.PricePerKg
						pricePerItem = pr.PricePerItem
						break
					}
				}

				// 计算金额
				totalAmount := float64(count) * pricePerItem

				// 打印销售数据
				fmt.Printf("  - 商品: %s, 规格: %s, 数量: %d, 单价: %.2f元/件, 总金额: %.2f元\n",
					productName, specName, count, pricePerItem, totalAmount)

				// 创建订单并添加到列表
				dateTime, _ := time.Parse("2006-01-02", date)
				order := models.Order{
					Date:         dateTime,
					ProductName:  productName,
					Spec:         specName,
					Quantity:     count,
					PricePerKg:   pricePerKg,
					PricePerItem: pricePerItem,
				}
				orderlist = append(orderlist, order)
				fmt.Println("orderlist", orderlist)
				// 添加到规格管理器
				sm.AddOrder(order)
			}
		}
	}

	// 打印总订单列表
	fmt.Println("\n====================")
	fmt.Println("生成的总订单列表:")
	for _, order := range orderlist {
		fmt.Printf("日期: %s, 商品: %s, 规格: %s, 数量: %d, 单价: %.2f元/件, 总金额: %.2f元\n",
			order.Date.Format("2006-01-02"), order.ProductName, order.Spec, order.Quantity,
			order.PricePerItem, float64(order.Quantity)*order.PricePerItem)
	}
	fmt.Println("====================")
}

// Checkout 生成财务账单 Excel，返回生成文件的绝对路径
func Checkout() (string, error) {
	fmt.Println("========== 商品销售统计Excel生成器 ==========")

	// 创建规格管理器
	sm := NewSpecManager()

	// 添加数据
	AddcheckoutData(sm)
	fmt.Println(sm.Orders)
	// 打印统计摘要
	sm.PrintSummary()

	// 生成Excel文件
	fmt.Println("\n========== 生成Excel文件 ==========")
	fmt.Println("begtime:", begtime2)
	fmt.Println("endtime:", endtime2)

	// 生成带时间戳的文件名，输出到配置的导出目录
	filename := fmt.Sprintf("%s_%s结账.xlsx", begtime2, endtime2)
	filePath := filepath.Join(global.Config.SavePath.Excel, filename)

	if err := sm.GenerateExcel(filePath); err != nil {
		fmt.Printf("❌ 生成Excel失败: %v\n", err)
		return "", err
	}
	fmt.Printf("✅ Excel生成成功！文件: %s\n", filePath)

	// 显示一些统计信息
	fmt.Println("\n========== 统计信息 ==========")
	fmt.Printf("总订单数: %d\n", len(sm.Orders))
	fmt.Printf("总商品种类: %d\n", len(sm.ProductMap))
	fmt.Printf("总销售天数: %d\n", len(sm.DateMap))

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return filePath, nil
	}
	return absPath, nil
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
	// 验证日期参数是否有效
	if statime.IsZero() || endtime.IsZero() {
		// 返回空响应
		return CheckoutResponse{
			FieldList: []string{},
			Count:     0,
			ProSPecs:  []ProSPecs{},
			TableData: []string{},
			PBMap:     global.ProductBelongMap,
			TotalRow:  []int{},
		}
	}

	// 起始日期（契约 §3.7 修复：原 statime.Day()+1 少算第一天，起始应含当日；
	// 前端 startTime 已为当日 00:00:00，此处归一化到当日日界即可）
	begindate := time.Date(statime.Year(), statime.Month(), statime.Day(), 0, 0, 0, 0, time.Local)
	begtime2 = fmt.Sprintf("%d年%d月%d日", begindate.Year(), int(begindate.Month()), begindate.Day())
	// 截至日期
	enddate := time.Date(endtime.Year(), endtime.Month(), endtime.Day(), 23, 59, 59, 0, time.Local)
	endtime2 = fmt.Sprintf("%d年%d月%d日", endtime.Year(), int(endtime.Month()), endtime.Day())

	// 查询数据库获取日期数据（契约 §3.7 修复：先构建带条件的 query 再 Count 再 Find——
	// 原 Count 在 Where 之前执行，统计的是全表数而非区间条数；原 Order 在 Find 之后属死代码）
	var _list []models.AddressInfo
	var count int64
	q := global.Db.Debug().Model(&models.AddressInfo{}).
		Where("created_at >= ? AND created_at <= ?", begindate, enddate)
	q.Count(&count)
	q.Order("created_at").Find(&_list)
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

	// 解析规格中的数值部分
	parseSpecValue := func(spec string) int {
		re, err := regexp.Compile(`\d+`)
		if err != nil {
			return 0
		}
		numberStr := re.FindString(spec)
		number, err := strconv.Atoi(numberStr)
		if err != nil {
			return 0
		}
		return number
	}

	// 转换为切片格式
	ProSPecsMap := make(map[string][]string)
	for k, v := range specsMap {
		for spec := range v {
			ProSPecsMap[k] = append(ProSPecsMap[k], spec)
		}
		// 对规格进行降序排序
		sort.Slice(ProSPecsMap[k], func(i, j int) bool {
			return parseSpecValue(ProSPecsMap[k][i]) < parseSpecValue(ProSPecsMap[k][j])
		})
	}

	// Map转List
	// Spec 价格默认 0（未配置态），由响应构建后的 applyPrices 从持久化价格统一回填（契约 §3.7 行为变更 3）
	var ProSPecsList []ProSPecs
	sid := uint(1)
	for k, v := range ProSPecsMap {
		var speclist = make([]Spec, 0)
		for _, vv := range v {
			speclist = append(speclist, Spec{
				Name:         vv,
				PricePerKg:   0, // 未配置默认 0，applyPrices 回填
				PricePerItem: 0, // 未配置默认 0，applyPrices 回填
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
	// 契约 §3.7 行为变更 3/4：ProSPecs 价格从持久化回填，并直接计算三个价格行
	// （与 TotalRow 同序同长，TotalAmountRow[i]=TotalRow[i]×PricePerItemRow[i]），
	// 服务重启后价格仍回显；读取失败降级空 map（价格为 0），不阻塞结账查询
	applyPrices(&resp, LoadPriceMap())
	checkoutresp = resp
	return resp
}

// 设置商品归属
func SetFieldList(mapp map[string][]string) {
	global.ProductBelongMap = mapp
	core.WriteProductBelongMap(mapp)
}

// SetPrice 保存价格并返回重算后的结账响应（契约 §3.6 行为变更 1/2/4）：
//  1. 先 SavePriceConfig 单事务落库（pricePerItem==0 时按 元/斤×斤数 推导后持久化），
//     任一项失败整体回滚并返回 error，由上层转 50000——先落库再返回，杜绝「响应已变、
//     库未更新」的半更新态（原实现仅改内存响应、不落库，重启即丢）；
//  2. 成功后维护 proPrice（Excel 导出路径 AddcheckoutData 消费，回填落库推导单价，
//     避免用户未填每件价时 Excel 金额算成 0）；
//  3. 再以最新生效价格 applyPrices 重算 checkoutresp：ProSPecs 价格回填 + 三个价格行
//     （与 TotalRow 同序同长，TotalAmountRow[i]=TotalRow[i]×PricePerItemRow[i]）。
//
// 已知边界（B6 内存态，可接受）：checkoutresp 为 GetDaydata 构建的当前区间响应副本，
// 服务重启后未先查 daydata 直接调 SetPrice 时它为空响应——价格仍正常落库，
// 待下次 daydata 查询时回显。
func SetPrice(_list []ProPriceRequest) (CheckoutResponse, error) {
	// 先落库：单事务 upsert（RUL-1 不保留历史），失败返回 error（上层转 50000）
	if err := SavePriceConfig(_list); err != nil {
		logrus.Errorf("保存价格配置失败: %v", err)
		return CheckoutResponse{}, err
	}

	// 落库成功后以最新生效价格重算
	priceMap := LoadPriceMap()

	// 为 Excel 导出路径（AddcheckoutData）维护价格快照：
	// PricePerItem 回填落库推导值，保证导出金额口径与库中一致
	proPrice = make([]ProPriceRequest, len(_list))
	for i := range _list {
		proPrice[i] = _list[i]
		if pc, ok := priceMap[_list[i].ProductName][_list[i].Spec]; ok {
			proPrice[i].PricePerKg = pc.PricePerKg
			proPrice[i].PricePerItem = pc.PricePerItem
		}
	}

	// 重算响应：ProSPecs 价格回填 + 三个价格行（checkoutresp 为 GetDaydata 构建的区间副本）
	applyPrices(&checkoutresp, priceMap)
	return checkoutresp, nil
}
