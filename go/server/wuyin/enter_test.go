package wuyin

import (
	"testing"

	"order_server/go/global"
	"order_server/go/models"

	"github.com/xuri/excelize/v2"
)

func TestMatchTag(t *testing.T) {
	global.WuyinTagConfigs = []models.WuyinTagConfig{
		{Name: "合肥仓", Keywords: models.WuyinKeywordsJSON{"合肥"}, Sort: 1},
		{Name: "广西仓", Keywords: models.WuyinKeywordsJSON{"广西", "南昌"}, Sort: 2},
		{Name: "云南仓", Keywords: models.WuyinKeywordsJSON{"云南", "长春"}, Sort: 3},
	}
	cases := map[string]string{
		"合肥仓工作微信":   "合肥仓",
		"广西～南昌订货微信": "广西仓",
		"云南仓+长春仓推单": "云南仓",
		"完全无关的昵称":   DefaultTag,
		"":          DefaultTag,
	}
	for sender, want := range cases {
		if got := MatchTag(sender); got != want {
			t.Errorf("MatchTag(%q) = %q, want %q", sender, got, want)
		}
	}
}

// buildWuyinXlsx 按吾饮仓日报真实列结构生成内存 xlsx
func buildWuyinXlsx(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetList()[0]
	header := []string{"日期", "客户编码", "客户", "电话", "物料编码", "规格型号", "物料名称", "库存单位", "实发数量"}
	for i, h := range header {
		if err := f.SetCellValue(sheet, cell(1, i), h); err != nil {
			t.Fatal(err)
		}
	}
	rows := [][]interface{}{
		{"2026/3/8", "WGUX07747627", "广西梧州市梧州学院（第二食堂）（陈洪生）吾饮", "13687741290", "WXGWXGWY0005", "9斤/件", "香水柠檬（9斤）", "件", 1},
		{"2026/3/8", "WGUX07762005", "广西省百色市田东县东宁东路31号（鲍安腾）吾饮", "13471647768/18277636045", "WXGWXGWY0005", "9斤/件", "香水柠檬（9斤）", "件", 2},
	}
	for r, row := range rows {
		for c, v := range row {
			if err := f.SetCellValue(sheet, cell(r+2, c), v); err != nil {
				t.Fatal(err)
			}
		}
	}
	return f
}

func cell(row, col int) string {
	return string(rune('A'+col)) + string(rune('0'+row))
}

func TestParseWuyinExcel(t *testing.T) {
	f := buildWuyinXlsx(t)
	list, err := parseWuyinExcel(f)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应解析 2 行，得到 %d", len(list))
	}
	first := list[0]
	if first.Recipient != "陈洪生" {
		t.Errorf("收件人 = %q, want 陈洪生（取最后一个括号）", first.Recipient)
	}
	if first.Phone != "13687741290" {
		t.Errorf("电话 = %q", first.Phone)
	}
	if first.Address != "广西梧州市梧州学院（第二食堂）（陈洪生）" {
		t.Errorf("地址去尾吾饮失败: %q", first.Address)
	}
	if first.Province != "广西壮族自治区" {
		t.Errorf("省份 = %q", first.Province)
	}
	// 第 1 行：实发数量 1 × 规格型号"9斤/件" → 1 个 9 斤条目
	if len(first.ProductSpecs) != 1 || first.ProductSpecs[0].Name != "香水柠檬" || first.ProductSpecs[0].Weight != 9 {
		t.Errorf("商品规格不符: %+v", first.ProductSpecs)
	}
	// 第 2 行：实发数量 2 → 2 个 9 斤条目（条目数即件数）
	second := list[1].ProductSpecs
	if len(second) != 2 || second[0].Weight != 9 || second[1].Weight != 9 {
		t.Errorf("第二行应展开为 2 个 9 斤条目: %+v", second)
	}
}

func TestParseWuyinExcelMissingCustomerColumn(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetList()[0]
	_ = f.SetCellValue(sheet, "A1", "无关表头")
	if _, err := parseWuyinExcel(f); err == nil {
		t.Error("缺客户列应报错")
	}
}

// TestImportExcelMergePreview ImportExcel 全链路（打标 + 追加预览）——
// 不触库：日志写库路径依赖 global.Db，本用例只验证 merge 与 tag 逻辑，
// 故直接调用 ImportExcel 前先把日志依赖绕开是不可行的；改为只验证纯解析已覆盖，
// 此处验证 mergeIntoWuyinPreview 的 ADD/RE 语义。
func TestMergeIntoWuyinPreview(t *testing.T) {
	global.WuyinAddressList = nil
	batch1 := []models.WuyinAddressInfo{{Recipient: "a"}, {Recipient: "b"}}
	mergeIntoWuyinPreview(batch1, 0) // RE
	if len(global.WuyinAddressList) != 2 || global.WuyinAddressList[0].PreviewID != 1 {
		t.Fatalf("RE 首批 VID 赋值错误: %+v", global.WuyinAddressList)
	}
	batch2 := []models.WuyinAddressInfo{{Recipient: "c"}}
	mergeIntoWuyinPreview(batch2, 1) // ADD
	if len(global.WuyinAddressList) != 3 || global.WuyinAddressList[2].PreviewID != 3 {
		t.Fatalf("ADD 追加或 VID 续号错误: %+v", global.WuyinAddressList)
	}
	mergeIntoWuyinPreview(batch2, 0) // RE 覆盖：与一点点 SetVid 同款行为，VID 从当前列表长度续号（4）而非归 1
	if len(global.WuyinAddressList) != 1 || global.WuyinAddressList[0].PreviewID != 4 {
		t.Fatalf("RE 覆盖错误: %+v", global.WuyinAddressList)
	}
	global.WuyinAddressList = nil
}
