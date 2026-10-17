package ordertutil

import "testing"

// TestExtractProvinceFullNames 34 个省级行政区全称逐一命中（全称匹配不限出现位置：开头与居中两种形态）。
func TestExtractProvinceFullNames(t *testing.T) {
	for _, p := range provinceTable {
		if got := ExtractProvince(p.Full + "某市某区幸福路1号"); got != p.Full {
			t.Errorf("全称开头: ExtractProvince(%q...) = %q, want %q", p.Full, got, p.Full)
		}
		if got := ExtractProvince("收件人张三 " + p.Full + "某市某区幸福路1号"); got != p.Full {
			t.Errorf("全称居中: ExtractProvince(...%q...) = %q, want %q", p.Full, got, p.Full)
		}
	}
}

// TestExtractProvinceShortNames 34 个简称逐一命中（简称位于地址开头且不含任何全称，走简称兜底轮；
// 构造时简称后接「某市」，避免「北京」+「市」拼出全称「北京市」而误走全称轮）。
func TestExtractProvinceShortNames(t *testing.T) {
	for _, p := range provinceTable {
		addr := p.Short + "某市某区幸福路1号"
		if got := ExtractProvince(addr); got != p.Full {
			t.Errorf("简称开头: ExtractProvince(%q) = %q, want %q", addr, got, p.Full)
		}
	}
}

// TestExtractProvinceShortPrefixNoFullName 省份简称开头、不含全称的真实订单地址形态（简称兜底轮），
// 含「北京市密云区…」同类场景（去掉「市」字后无省份全称）。
func TestExtractProvinceShortPrefixNoFullName(t *testing.T) {
	cases := []struct{ name, addr, want string }{
		{"北京密云区开头无省份全称", "北京密云区示例镇示例景区-某茶馆", "北京市"},
		{"内蒙古开头", "内蒙古呼和浩特市赛罕区大学东街88号", "内蒙古自治区"},
		{"新疆开头", "新疆乌鲁木齐市天山区人民路100号", "新疆维吾尔自治区"},
		{"香港开头", "香港中环皇后大道中1号", "香港特别行政区"},
		{"广西开头", "广西南宁市青秀区民族大道100号", "广西壮族自治区"},
	}
	for _, c := range cases {
		if got := ExtractProvince(c.addr); got != c.want {
			t.Errorf("%s: ExtractProvince(%q) = %q, want %q", c.name, c.addr, got, c.want)
		}
	}
}

// TestExtractProvinceNoFalsePositive 「城市名/区县名 + 他省简称路名」不得误判（简称轮的市/区/县前置排除）；
// 以及无省份信息返回空串。
func TestExtractProvinceNoFalsePositive(t *testing.T) {
	cases := []struct{ name, addr, want string }{
		{"南京市上海路不误判为上海", "南京市上海路5号", ""},
		{"青岛市广西路不误判为广西", "青岛市广西路12号", ""},
		{"完整地址中的上海路由全称轮命中江苏省", "江苏省南京市鼓楼区上海路5号", "江苏省"},
		{"完整地址中的广西路由全称轮命中山东省", "山东省青岛市市南区广西路12号", "山东省"},
		{"无省级行政区信息返回空串", "杭州市西湖区文三路100号", ""},
		{"空地址返回空串", "", ""},
	}
	for _, c := range cases {
		if got := ExtractProvince(c.addr); got != c.want {
			t.Errorf("%s: ExtractProvince(%q) = %q, want %q", c.name, c.addr, got, c.want)
		}
	}
}

// TestExtractProvinceMultipleHitsByTableOrder 地址含多个省份字样时，按 provinceTable 表序取第一个命中
// （而非地址中的出现顺序）。
func TestExtractProvinceMultipleHitsByTableOrder(t *testing.T) {
	cases := []struct{ name, addr, want string }{
		{"全称多命中-地址顺序与表序一致", "福建省厦门市思明区，后转运至广东省深圳市南山区", "福建省"},
		{"全称多命中-地址中广东在前但表序福建在前", "广东省深圳市南山区，后转运至福建省厦门市思明区", "福建省"},
		{"简称多命中-表序福建先于广东", "广东福建交界处某市幸福路1号", "福建省"},
	}
	for _, c := range cases {
		if got := ExtractProvince(c.addr); got != c.want {
			t.Errorf("%s: ExtractProvince(%q) = %q, want %q", c.name, c.addr, got, c.want)
		}
	}
}
