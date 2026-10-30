package wuyin

import (
	"fmt"
	"order_server/go/global"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

// SaveWuyinToExcel 吴饮预览数据导出 Excel，返回生成文件绝对路径。
// 列在一点点导出的基础上增加 省份/标签/发送人。
func SaveWuyinToExcel() (string, error) {
	addressList := global.WuyinAddressList
	if len(addressList) == 0 {
		return "", fmt.Errorf("预览没有数据")
	}
	filename := fmt.Sprintf("吴饮_%s.xlsx", time.Now().Format("2006-01-02"))
	filePath := filepath.Join(global.Config.SavePath.Excel, filename)

	f := excelize.NewFile()
	defer f.Close()

	const sheetName = "吴饮识别结果"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %w", err)
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return "", fmt.Errorf("创建表头样式失败: %w", err)
	}

	headers := []string{"序号", "收件人", "手机号", "收货地址", "省份", "商品规格", "标签", "发送人", "备注"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	for i, addr := range addressList {
		row := i + 2
		values := []interface{}{
			i + 1, addr.Recipient, addr.Phone, addr.Address, addr.Province,
			addr.ProductSpecsString(false), addr.Tag, addr.Sender, addr.Remarks,
		}
		for c, v := range values {
			cell := fmt.Sprintf("%c%d", 'A'+c, row)
			f.SetCellValue(sheetName, cell, v)
		}
	}

	widths := []float64{8, 25, 18, 80, 16, 28, 12, 30, 20}
	for c, w := range widths {
		f.SetColWidth(sheetName, string(rune('A'+c)), string(rune('A'+c)), w)
	}

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存 Excel 失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮预览导出", "保存路径：\n"+filePath)
	return filePath, nil
}
