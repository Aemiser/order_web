package wuyin

import (
	"fmt"
	"io"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"
	wuyinutil "order_server/go/utils/wuyin"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ImportExcel 导入仓库订单明细 xlsx 到吴饮预览（追加模式）。
// 真实表头（吾饮良品各仓日报）：日期|客户编码|客户|电话|物料编码|规格型号|物料名称|库存单位|实发数量；
// "客户"列为"地址（收件人）吾饮"复合格式。tag 入参为本次导入统一打的标签
// （前端按所选仓/文件名给出，缺省落 DefaultTag）。
// 规格口径（2026-10-30）：商品规格以斤存储——规格型号"9斤/件"给出每件斤数，
// 实发数量 N 件展开为 N 个 9 斤条目。
func ImportExcel(reader io.Reader, filename string, tag string) ([]wuyinutil.WuyinExceladdressInfo, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("打开 Excel 失败: %w", err)
	}
	defer f.Close()

	list, err := parseWuyinExcel(f)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		tag = DefaultTag
	}
	for i := range list {
		list[i].Tag = tag
		list[i].Sender = fmt.Sprintf("表格导入:%s", filename)
	}

	// 追加进预览（导入永远追加，不清空已识别内容）
	mergeIntoWuyinPreview(list, enum.RecPreviewType_ADD)

	alog.Logaction(enum.LogInfoLevel, "吴饮表格导入",
		fmt.Sprintf("文件=%s 标签=%s 条数=%d", filename, tag, len(list)))

	return wuyinutil.WuyinToExcelAddress(global.WuyinAddressList), nil
}

// parseWuyinExcel 纯解析：xlsx → 吴饮订单列表（不打标、不写预览、不落日志）。
// 拆出纯函数是为了可单测（ImportExcel 尾部的日志写库在无 DB 的测试环境会崩）。
func parseWuyinExcel(f *excelize.File) ([]models.WuyinAddressInfo, error) {
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("表格没有数据行")
	}

	// 表头定位（容错：按列名找，不假设固定顺序）
	header := rows[0]
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}
	customerCol, ok := colIdx["客户"]
	if !ok {
		return nil, fmt.Errorf("缺少「客户」列，识别到的表头：%v", header)
	}
	phoneCol := colIdx["电话"]
	productCol := colIdx["物料名称"]
	specCol := colIdx["规格型号"]
	qtyCol, hasQty := colIdx["实发数量"]
	if !hasQty {
		qtyCol = colIdx["数量"]
	}

	var list []models.WuyinAddressInfo
	for _, row := range rows[1:] {
		get := func(idx int) string {
			if idx < 0 || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		customer := get(customerCol)
		if customer == "" {
			continue // 空行/合计行前置拦截（合计行的客户列通常为空）
		}

		// 件数（实发数量），至少 1
		qty := 1
		if qtyStr := get(qtyCol); qtyStr != "" {
			if n, err := strconv.Atoi(extractLeadingInt(qtyStr)); err == nil && n > 0 {
				qty = n
			}
		}
		product := wuyinutil.NormalizeProductName(get(productCol))
		if product == "" {
			product = "香水柠檬" // 表头无物料列时的兜底（吾饮主品）
		}
		// 每件斤数：优先取规格型号（"9斤/件"），缺省 9（吾饮标准件）
		packJin := extractJin(get(specCol))
		if packJin <= 0 {
			packJin = 9
		}

		// N 件 → N 个斤条目（条目数即件数，结账按条目计数）
		specs := make(models.WuyinProductSpecsJSON, 0, qty)
		for i := 0; i < qty; i++ {
			specs = append(specs, models.WuyinProductSpec{Name: product, Weight: packJin})
		}

		list = append(list, models.WuyinAddressInfo{
			Recipient:    wuyinutil.ExtractRecipientFromAddress(customer),
			Phone:        get(phoneCol),
			Address:      wuyinutil.CleanCustomerAddress(customer),
			Province:     ordertutil.ExtractProvince(customer),
			ProductSpecs: specs,
		})
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("没有解析到有效数据行（客户列全为空）")
	}
	return list, nil
}

// extractLeadingInt 取字符串开头的整数（实发数量可能是 "1.0" 或 "1"）
func extractLeadingInt(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return "0"
	}
	return s[:end]
}

// extractJin 从规格型号取每件斤数（"9斤/件"→9；"9斤/件 "、"9斤"均可），无斤数返回 0
func extractJin(spec string) int {
	n, err := strconv.Atoi(extractLeadingInt(spec))
	if err != nil {
		return 0
	}
	return n
}
