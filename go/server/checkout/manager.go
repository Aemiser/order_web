package checkout

import (
	"fmt"
	"order_server/go/utils/execl"
	"sort"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

// Order 订单结构
type Order struct {
	Date        time.Time
	ProductName string
	Spec        string
	Quantity    int
	Unit        string
}

// ProductSpecs 商品规格集合
type ProductSpecs struct {
	Name     string
	Specs    []string
	SpecMap  map[string]int // 商品规格对应的数量
	TotalQty int
}

// SpecManager 规格管理器
type SpecManager struct {
	Orders     []Order
	DateMap    map[string][]Order
	ProductMap map[string]*ProductSpecs //商品对应下面的规格信息
}

// NewSpecManager 创建规格管理器
func NewSpecManager() *SpecManager {
	return &SpecManager{
		Orders:     make([]Order, 0),
		DateMap:    make(map[string][]Order),
		ProductMap: make(map[string]*ProductSpecs),
	}
}

// AddOrder 添加订单
func (sm *SpecManager) AddOrder(order Order) {
	sm.Orders = append(sm.Orders, order)

	// 使用格式化的日期字符串作为key
	dateKey := execl.FormatDate(order.Date)
	sm.DateMap[dateKey] = append(sm.DateMap[dateKey], order)

	// 按商品分组
	// 遍历利商品，对不存在的商品，就创建一个存入map中
	if _, exists := sm.ProductMap[order.ProductName]; !exists {
		sm.ProductMap[order.ProductName] = &ProductSpecs{
			Name:    order.ProductName,
			Specs:   []string{},
			SpecMap: make(map[string]int),
		}
	}

	// 从该订单中获取该商品分类
	product := sm.ProductMap[order.ProductName]
	// 对于该商品下的规格，若存在则进行加法，不存在则创建key.value
	product.SpecMap[order.Spec] = product.SpecMap[order.Spec] + order.Quantity
	// 统计商品分类总数
	product.TotalQty += order.Quantity
}

// BuildSpecList 构建规格列表
func (sm *SpecManager) BuildSpecList() {
	for _, product := range sm.ProductMap {
		// 收集所有规格
		for spec := range product.SpecMap {
			product.Specs = append(product.Specs, spec)
		}
		// 规格排序（按数字大小）
		sort.Slice(product.Specs, func(i, j int) bool {
			numI := execl.ExtractNumber(product.Specs[i])
			numJ := execl.ExtractNumber(product.Specs[j])
			return numI < numJ
		})
	}
}

// GenerateExcel 生成Excel
func (sm *SpecManager) GenerateExcel(filename string) error {
	sm.BuildSpecList()

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "商品销售统计"
	f.SetSheetName("Sheet1", sheetName)

	// 获取所有日期并排序
	dates := make([]string, 0, len(sm.DateMap))
	for date := range sm.DateMap {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// 获取所有商品并排序
	products := make([]string, 0, len(sm.ProductMap))
	for product := range sm.ProductMap {
		products = append(products, product)
	}
	sort.Strings(products)

	// 创建样式
	styles := sm.createStyles(f)

	// ===== 设置表头结构 =====
	// 第一行：商品名称（跨两行？不，这里商品名称应该在第一行，日期在第二行）
	// 修正：日期应该独占两行（合并行），商品名称在右边占据多列

	// 第1行：空出A列（留给日期合并行用），从B列开始放商品名称
	col := 2 // B列开始

	// 记录每个商品的起始列和结束列
	productCols := make(map[string]int)    // 商品起始列
	productEndCols := make(map[string]int) // 商品结束列

	// 第1行：设置商品名称（占据第1行）
	for _, product := range products {
		specs := sm.ProductMap[product].Specs
		productCols[product] = col
		productEndCols[product] = col + len(specs) - 1

		// 商品名称放在第1行，合并该商品的所有规格列
		startCell := execl.GetColumnLetter(col) + "1"
		endCell := execl.GetColumnLetter(col+len(specs)-1) + "1"
		f.MergeCell(sheetName, startCell, endCell)
		f.SetCellValue(sheetName, startCell, product)
		f.SetCellStyle(sheetName, startCell, endCell, styles["productHeader"])

		col += len(specs)
	}

	// 第2行：设置规格（在第2行）
	col = 2 // 重置列，从B列开始
	for _, product := range products {
		specs := sm.ProductMap[product].Specs
		for i, spec := range specs {
			specCell := execl.GetColumnLetter(col+i) + "2"
			f.SetCellValue(sheetName, specCell, spec)
			f.SetCellStyle(sheetName, specCell, specCell, styles["specHeader"])
		}
		col += len(specs)
	}

	// ===== 关键修正：日期列独占两行（合并行）=====
	// 合并A1和A2单元格，让日期标题独占两行
	f.MergeCell(sheetName, "A1", "A2")
	f.SetCellValue(sheetName, "A1", "日期")
	f.SetCellStyle(sheetName, "A1", "A2", styles["dateHeader"])

	// 设置垂直居中样式
	verticalCenterStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical:   "center",
			Horizontal: "center",
		},
	})
	f.SetCellStyle(sheetName, "A1", "A1", verticalCenterStyle)

	// 填充数据 - 从第3行开始
	for rowIdx, dateStr := range dates {
		row := rowIdx + 3

		// 解析日期
		date, _ := time.Parse("2006-01-02", dateStr)

		// 设置日期（格式：MM-DD）
		dateCell := fmt.Sprintf("A%d", row)
		f.SetCellValue(sheetName, dateCell, execl.FormatShortDate(date))

		// 设置每个商品的数量
		for _, product := range products {
			startCol := productCols[product]
			specs := sm.ProductMap[product].Specs

			for i, spec := range specs {
				// 查找该日期下该商品该规格的数量
				qty := 0
				for _, order := range sm.DateMap[dateStr] {
					if order.ProductName == product && order.Spec == spec {
						qty += order.Quantity
					}
				}

				cell := execl.GetColumnLetter(startCol+i) + strconv.Itoa(row)
				if qty > 0 {
					f.SetCellValue(sheetName, cell, qty)
					f.SetCellStyle(sheetName, cell, cell, styles["data"])
				} else {
					// 空单元格
					f.SetCellStyle(sheetName, cell, cell, styles["empty"])
				}
			}
		}
	}

	// 添加合计行
	totalRow := len(dates) + 3
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", totalRow), "合计")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("A%d", totalRow), styles["total"])

	// 计算每个规格的合计
	for _, product := range products {
		startCol := productCols[product]
		specs := sm.ProductMap[product].Specs

		for i, spec := range specs {
			total := 0
			for _, orders := range sm.DateMap {
				for _, order := range orders {
					if order.ProductName == product && order.Spec == spec {
						total += order.Quantity
					}
				}
			}

			cell := execl.GetColumnLetter(startCol+i) + strconv.Itoa(totalRow)
			if total > 0 {
				f.SetCellValue(sheetName, cell, total)
				f.SetCellStyle(sheetName, cell, cell, styles["total"])
			}
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 10) // 日期列

	for _, product := range products {
		for i := productCols[product]; i <= productEndCols[product]; i++ {
			colLetter := execl.GetColumnLetter(i)
			f.SetColWidth(sheetName, colLetter, colLetter, 10)
		}
	}

	// 添加边框
	lastCol := execl.GetColumnLetter(col - 1)
	f.SetCellStyle(sheetName, "A1", lastCol+strconv.Itoa(totalRow), styles["border"])

	// 添加自动筛选
	//f.AutoFilter(sheetName, "A1", lastCol+"2", "")

	//// 创建样式
	//// 应用样式到单元格
	//f.SetCellStyle(sheetName, "A1", "A8", styles["dateHeader"])
	//f.SetCellStyle(sheetName, "B2", "D2", styles["specHeader"])
	//f.SetCellStyle(sheetName, "B3", "N8", styles["data"])

	return f.SaveAs(filename)
}

// createStyles 创建Excel样式
func (sm *SpecManager) createStyles(f *excelize.File) map[string]int {
	styles := make(map[string]int)

	// 商品表头样式
	productHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})
	styles["productHeader"] = productHeader

	// 规格表头样式
	specHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9E1F2"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})
	styles["specHeader"] = specHeader

	// 日期表头样式
	dateHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFC000"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})
	styles["dateHeader"] = dateHeader

	// 日期数据样式
	dateData, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFF2CC"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})
	styles["dateData"] = dateData

	// 数据样式
	data, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#CCCCCC", Style: 1},
			{Type: "right", Color: "#CCCCCC", Style: 1},
		},
	})
	styles["data"] = data

	// 空单元格样式
	empty, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F5F5F5"},
			Pattern: 1,
		},
	})
	styles["empty"] = empty

	// 合计行样式
	total, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E2EFDA"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#000000", Style: 2},
			{Type: "bottom", Color: "#000000", Style: 2},
		},
	})
	styles["total"] = total

	// 边框样式
	border, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	styles["border"] = border

	return styles
}

// PrintSummary 打印统计摘要
func (sm *SpecManager) PrintSummary() {
	fmt.Println("=== 商品销售统计 ===")

	// 按日期统计
	fmt.Println("\n按日期统计：")
	dates := make([]string, 0, len(sm.DateMap))
	for date := range sm.DateMap {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	for _, dateStr := range dates {
		orders := sm.DateMap[dateStr]
		totalQty := 0
		productStats := make(map[string]int)

		for _, order := range orders {
			totalQty += order.Quantity
			productStats[order.ProductName] += order.Quantity
		}

		fmt.Printf("  %s: 总销量 %d\n", dateStr, totalQty)
		for product, qty := range productStats {
			fmt.Printf("    %s: %d\n", product, qty)
		}
	}

	// 按商品统计
	fmt.Println("\n按商品统计：")
	for _, product := range sm.ProductMap {
		fmt.Printf("  %s:\n", product.Name)
		fmt.Printf("    总销量: %d\n", product.TotalQty)
		fmt.Println("    规格分布:")
		for _, spec := range product.Specs {
			fmt.Printf("      %s: %d\n", spec, product.SpecMap[spec])
		}
	}
}
