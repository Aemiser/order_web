package wuyinutil

import (
	"testing"

	"order_server/go/models"
)

func TestSpecItemsProcessing(t *testing.T) {
	cases := []struct {
		in        string
		wantLen   int
		wantFirst models.WuyinProductSpec
	}{
		{"【香水柠檬9斤】", 1, models.WuyinProductSpec{Name: "香水柠檬", Weight: 9}},
		{"【黄柠檬5斤】【香水柠檬9斤】", 2, models.WuyinProductSpec{Name: "黄柠檬", Weight: 5}},
		{"【香水柠檬9】", 1, models.WuyinProductSpec{Name: "香水柠檬", Weight: 9}}, // 单位可省
		// 两件 = 两个 9 斤条目（条目数即件数）
		{"【香水柠檬9斤】【香水柠檬9斤】", 2, models.WuyinProductSpec{Name: "香水柠檬", Weight: 9}},
	}
	for _, c := range cases {
		got, err := SpecItemsProcessing(c.in)
		if err != nil {
			t.Fatalf("SpecItemsProcessing(%q) 报错: %v", c.in, err)
		}
		if len(got) != c.wantLen || got[0] != c.wantFirst {
			t.Errorf("SpecItemsProcessing(%q) = %v, want 首项 %v 长 %d", c.in, got, c.wantFirst, c.wantLen)
		}
	}
	// 口径修正：商品规格只认斤，"件"形态必须报错
	if got, err := SpecItemsProcessing("【香水柠檬1件】"); err == nil {
		t.Errorf("件单位规格应报错，得到 %v", got)
	}
	if _, err := SpecItemsProcessing("没有规格"); err == nil {
		t.Error("无规格内容应报错")
	}
}

func TestExtractRecipientFromAddress(t *testing.T) {
	cases := map[string]string{
		"广西梧州市梧州学院（第二食堂）（陈洪生）吾饮":     "陈洪生", // 取最后一个括号
		"广西百色市平果平铝集团综合楼第5间铺面（潘雪晶）吾饮": "潘雪晶",
		"江苏省徐州市铜山区望城村168号（王柱、张超）吾饮":  "王柱、张超",
		"没有任何括号的地址吾饮":                "",
	}
	for in, want := range cases {
		if got := ExtractRecipientFromAddress(in); got != want {
			t.Errorf("ExtractRecipientFromAddress(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanCustomerAddress(t *testing.T) {
	if got := CleanCustomerAddress("广西百色市平果某铺面（潘雪晶）吾饮"); got != "广西百色市平果某铺面（潘雪晶）" {
		t.Errorf("去尾吾饮失败: %q", got)
	}
	if got := CleanCustomerAddress("  前后空白  "); got != "前后空白" {
		t.Errorf("去空白失败: %q", got)
	}
}

func TestNormalizeProductName(t *testing.T) {
	cases := map[string]string{
		"香水柠檬（9斤）":    "香水柠檬",
		"黄柠檬":         "黄柠檬",
		" 尤力克青柠 (规格)": "尤力克青柠",
	}
	for in, want := range cases {
		if got := NormalizeProductName(in); got != want {
			t.Errorf("NormalizeProductName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRespBodyToJsonWuyin(t *testing.T) {
	// 数组形态
	list, err := RespBodyToJsonWuyin(`[{"收件人":"潘雪晶","手机号":"18277682337","商品规格":[{"name":"香水柠檬","weight":9}],"发送人":"广西～南昌订货微信"}]`)
	if err != nil || len(list) != 1 {
		t.Fatalf("数组解析失败: %v %v", list, err)
	}
	if list[0].Recipient != "潘雪晶" || list[0].ProductSpecs[0].Weight != 9 || list[0].Sender != "广西～南昌订货微信" {
		t.Errorf("数组字段不符: %+v", list[0])
	}
	// markdown 包裹 + 单对象
	one, err := RespBodyToJsonWuyin("```json\n{\"收件人\":\"胡文建\"}\n```")
	if err != nil || len(one) != 1 || one[0].Recipient != "胡文建" {
		t.Errorf("单对象解析失败: %v %v", one, err)
	}
}

func TestExtractSender(t *testing.T) {
	cases := map[string]string{
		"[2026-10-29 16:52] 广西～南昌订货微信: 广西百色市（潘雪晶）吾饮 18277682337": "广西～南昌订货微信",
		"合肥仓工作微信: 安徽省马鞍山（谢煜）吾饮 18056898922":                      "合肥仓工作微信",
		"没有发送人前缀的裸文本":                                            "",
	}
	for in, want := range cases {
		if got := ExtractSender(in); got != want {
			t.Errorf("ExtractSender(%q) = %q, want %q", in, got, want)
		}
	}
}
