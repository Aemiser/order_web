package wuyin

import (
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	"path/filepath"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// 吴饮结账：按 仓库标签 × 商品 × 规格(斤) 聚合。件数 = 规格条目数
// （每条目即 1 件，如两个"9斤"条目 = 2 件），按 元/件 计价。
// 与一点点 checkout 的差异：件数语义、分组维度是 tag、无商品归属归一。

// WuyinCheckoutProduct 单商品行：按日期对齐的件数 + 单价（回填）+ 金额
type WuyinCheckoutProduct struct {
	Name         string  `json:"name"`
	Spec         string  `json:"spec"`         // 规格串，如 "9斤"
	ItemsPerDate []int   `json:"itemsPerDate"` // 与 Dates 下标对齐（件数）
	TotalItems   int     `json:"totalItems"`   // 总件数（= 该规格条目数）
	PricePerItem float64 `json:"pricePerItem"` // 元/件（价格配置回填，未配置为 0）
	TotalAmount  float64 `json:"totalAmount"`  // TotalItems × PricePerItem
}

// WuyinCheckoutResponse 单个标签的账单
type WuyinCheckoutResponse struct {
	Tag        string                 `json:"tag"`
	Dates      []string               `json:"dates"` // "10月29日"
	Products   []WuyinCheckoutProduct `json:"products"`
	Count      int                    `json:"count"`      // 该标签订单条数
	GrandTotal float64                `json:"grandTotal"` // 金额合计
}

// WuyinPriceRequest 设置价格入参
type WuyinPriceRequest struct {
	ProductName  string  `json:"productName"`
	Spec         string  `json:"spec"` // 规格串，如 "9斤"
	PricePerItem float64 `json:"pricePerItem"`
}

// 最近一次 daydata 的查询参数快照（SetPrice 后重算用，同一点点 checkout 的
// checkoutresp 内存态约定：重启后先查 daydata 再改价）
var (
	lastRangeStart, lastRangeEnd time.Time
	lastTags                     []string
)

// priceKey 价格配置复合键：商品 + 规格（"香水柠檬|9斤"）
func priceKey(productName, spec string) string {
	return productName + "|" + spec
}

// loadWuyinPriceMap 价格配置 → map[商品|规格]元/件
func loadWuyinPriceMap() map[string]float64 {
	m := map[string]float64{}
	var list []models.WuyinPriceConfig
	if err := global.DbWuyin.Find(&list).Error; err != nil {
		return m
	}
	for _, p := range list {
		m[priceKey(p.ProductName, p.Spec)] = p.PricePerItem
	}
	return m
}

// GetWuyinDayData 按标签分组的结账数据。tags 为空 = 全部标签。
func GetWuyinDayData(start, end time.Time, tags []string) ([]WuyinCheckoutResponse, error) {
	if start.IsZero() || end.IsZero() {
		return []WuyinCheckoutResponse{}, nil
	}
	lastRangeStart, lastRangeEnd, lastTags = start, end, tags

	begindate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	enddate := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)

	query := global.DbWuyin.Model(&models.WuyinAddressInfo{}).
		Where("created_at >= ? AND created_at <= ?", begindate, enddate)
	if len(tags) > 0 {
		query = query.Where("tag IN ?", tags)
	}
	var list []models.WuyinAddressInfo
	if err := query.Order("created_at").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	priceMap := loadWuyinPriceMap()

	// tag → date → (商品,规格) → 件数；每个规格条目 1 件
	type itemKey struct{ name, spec string }
	type group struct {
		dates map[string]map[itemKey]int // date → (商品,规格) → 件数
		count int
	}
	groups := map[string]*group{}
	dateSet := map[string]bool{}
	for _, info := range list {
		tag := info.Tag
		if tag == "" {
			tag = DefaultTag
		}
		g := groups[tag]
		if g == nil {
			g = &group{dates: map[string]map[itemKey]int{}}
			groups[tag] = g
		}
		g.count++
		date := info.CreatedAt.Format("2006-01-02")
		dateSet[date] = true
		if g.dates[date] == nil {
			g.dates[date] = map[itemKey]int{}
		}
		for _, spec := range info.ProductSpecs {
			key := itemKey{name: spec.Name, spec: fmt.Sprintf("%d斤", spec.Weight)}
			g.dates[date][key]++ // 每条目即 1 件
		}
	}

	// 日期列升序
	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	dateLabels := make([]string, len(dates))
	dateIdx := map[string]int{}
	for i, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		dateLabels[i] = fmt.Sprintf("%d月%d日", t.Month(), t.Day())
		dateIdx[d] = i
	}

	// 稳定输出：标签按名称排序
	tagNames := make([]string, 0, len(groups))
	for t := range groups {
		tagNames = append(tagNames, t)
	}
	sort.Strings(tagNames)

	resp := make([]WuyinCheckoutResponse, 0, len(tagNames))
	for _, tag := range tagNames {
		g := groups[tag]
		// (商品,规格) → 按日期对齐的件数行（写入时按 dateIdx 定位，懒扩到全日期宽）
		rowItems := map[itemKey][]int{}
		rowTotals := map[itemKey]int{}
		for date, items := range g.dates {
			for key, cnt := range items {
				row := rowItems[key]
				if len(row) < len(dates) {
					expanded := make([]int, len(dates))
					copy(expanded, row)
					row = expanded
				}
				row[dateIdx[date]] += cnt
				rowItems[key] = row
				rowTotals[key] += cnt
			}
		}
		keys := make([]itemKey, 0, len(rowItems))
		for k := range rowItems {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].name != keys[j].name {
				return keys[i].name < keys[j].name
			}
			return keys[i].spec < keys[j].spec
		})

		products := make([]WuyinCheckoutProduct, 0, len(keys))
		grand := 0.0
		for _, k := range keys {
			price := priceMap[priceKey(k.name, k.spec)]
			total := rowTotals[k]
			amount := float64(total) * price
			grand += amount
			products = append(products, WuyinCheckoutProduct{
				Name:         k.name,
				Spec:         k.spec,
				ItemsPerDate: rowItems[k],
				TotalItems:   total,
				PricePerItem: price,
				TotalAmount:  amount,
			})
		}
		resp = append(resp, WuyinCheckoutResponse{
			Tag:        tag,
			Dates:      dateLabels,
			Products:   products,
			Count:      g.count,
			GrandTotal: grand,
		})
	}
	return resp, nil
}

// SetWuyinPrice 保存价格（单事务 upsert）并以最近一次查询范围重算响应。
// spec 为规格串（如 "9斤"），由前端从账单行携带。
func SetWuyinPrice(list []WuyinPriceRequest) ([]WuyinCheckoutResponse, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("没有需要保存的价格")
	}
	for i := range list {
		if list[i].Spec == "" {
			return nil, fmt.Errorf("第 %d 项缺少规格（spec）", i+1)
		}
	}
	err := global.DbWuyin.Transaction(func(tx *gorm.DB) error {
		for _, p := range list {
			var existing models.WuyinPriceConfig
			err := tx.Where("product_name = ? AND spec = ?", p.ProductName, p.Spec).
				First(&existing).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					if err := tx.Create(&models.WuyinPriceConfig{
						ProductName:  p.ProductName,
						Spec:         p.Spec,
						PricePerItem: p.PricePerItem,
					}).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			if err := tx.Model(&existing).Update("price_per_item", p.PricePerItem).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		alog.Logaction(enum.LogErrLevel, "吴饮保存价格失败", err.Error())
		return nil, fmt.Errorf("保存价格失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮保存价格", fmt.Sprintf("%d 项", len(list)))

	// 以最近一次 daydata 范围重算（重启后未查询过则为零值 → 空组）
	return GetWuyinDayData(lastRangeStart, lastRangeEnd, lastTags)
}

// ExportWuyinCheckout 生成结账 Excel：每个标签一个 sheet。tags 为空 = 全部。
func ExportWuyinCheckout(tags []string) (string, error) {
	if lastRangeStart.IsZero() || lastRangeEnd.IsZero() {
		return "", fmt.Errorf("请先查询结账数据")
	}
	groups, err := GetWuyinDayData(lastRangeStart, lastRangeEnd, tags)
	if err != nil {
		return "", err
	}
	if len(groups) == 0 {
		return "", fmt.Errorf("该区间没有订单数据")
	}

	f := excelize.NewFile()
	defer f.Close()
	f.DeleteSheet("Sheet1")

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	for _, g := range groups {
		// Excel sheet 名即标签名（NewSheet 返回索引，这里直接以名字引用）
		sheetName := g.Tag
		if _, err := f.NewSheet(sheetName); err != nil {
			return "", fmt.Errorf("创建工作表失败: %w", err)
		}
		// 表头：商品 | 规格 | 各日期 | 总件数 | 单价(元/件) | 金额
		headers := append([]string{"商品", "规格"}, g.Dates...)
		headers = append(headers, "总件数", "单价(元/件)", "金额(元)")
		for i, h := range headers {
			cell := fmt.Sprintf("%c1", 'A'+i)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
		for r, p := range g.Products {
			row := r + 2
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), p.Name)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), p.Spec)
			for c, n := range p.ItemsPerDate {
				cell := fmt.Sprintf("%c%d", 'C'+c, row)
				if n == 0 {
					f.SetCellValue(sheetName, cell, "—")
				} else {
					f.SetCellValue(sheetName, cell, n)
				}
			}
			base := 'C' + len(g.Dates)
			f.SetCellValue(sheetName, fmt.Sprintf("%c%d", base, row), p.TotalItems)
			f.SetCellValue(sheetName, fmt.Sprintf("%c%d", base+1, row), p.PricePerItem)
			f.SetCellValue(sheetName, fmt.Sprintf("%c%d", base+2, row), p.TotalAmount)
		}
		// 合计行
		totalRow := len(g.Products) + 2
		base := 'C' + len(g.Dates)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", totalRow), "合计")
		f.SetCellValue(sheetName, fmt.Sprintf("%c%d", base+2, totalRow), g.GrandTotal)
		f.SetColWidth(sheetName, "A", "A", 22)
		f.SetColWidth(sheetName, "B", "B", 10)
		if len(g.Dates) > 0 {
			f.SetColWidth(sheetName, "C", string(rune('A'+len(headers)-1)), 12)
		}
	}

	filename := fmt.Sprintf("吴饮结账_%s_%s.xlsx",
		lastRangeStart.Format("0102"), lastRangeEnd.Format("0102"))
	filePath := filepath.Join(global.Config.SavePath.Excel, filename)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存 Excel 失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮结账导出", filePath)
	return filePath, nil
}
