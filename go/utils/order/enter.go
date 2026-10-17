package ordertutil

import (
	"encoding/json"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"regexp"
	"strconv"
	"strings"
)

func Specprocessing(input string) (models.ProductSpecsJSON, error) {
	//	【】
	// 更通用的正则表达式：
	// 匹配：【名称数字斤】 或 【名称数字】 格式
	// 解释：
	// \[ 匹配左括号
	// ([^】]+?) 非贪婪匹配名称（不含】的字符）
	// (\d+) 匹配数字（数量）
	// (?:斤)? 可选的"斤"字（非捕获组）
	// \] 匹配右括号
	re := regexp.MustCompile(`【([^】]+?)(\d+)(?:斤)?】`)

	matches := re.FindAllStringSubmatch(input, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("没有找到匹配的内容")
	}

	var items models.ProductSpecsJSON

	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1]
			numberStr := match[2]

			// 转换数字
			number, err := strconv.Atoi(numberStr)
			if err != nil {
				return nil, fmt.Errorf("转换数量失败: %v", err)
			}

			items = append(items, models.ProductSpec{
				Name:   name,
				Weight: number,
			})
		}
	}

	return items, nil
}

func SetVid(infolist []models.AddressInfo) []models.AddressInfo {
	length := len(global.AddressList)
	for i := range infolist {
		addr := &infolist[i] // 获取元素的指针
		addr.PreviewID = uint(length + i + 1)
	}
	return infolist
}

// 检查是否存在新字段
func Specchecking(list []models.AddressInfo) (newSpecList []string) {
	newSpecMap := make(map[string]int, 0)
	SpecMap := make(map[string]int, 0)
	// 把map拉长为一维
	for k, v := range global.ProductBelongMap {
		SpecMap[k] = 1
		for _, vv := range v {
			SpecMap[vv] = 1
		}
	}

	for _, info := range list {
		for _, spec := range info.ProductSpecs {
			if _, ok := SpecMap[spec.Name]; !ok {
				newSpecMap[spec.Name] = 1
			}
		}
	}
	// map转一维切片
	for k, _ := range newSpecMap {
		newSpecList = append(newSpecList, k)
	}
	fmt.Println("检查到新商品：", newSpecList)
	return
}

func GetNumberInSpec(s string) int {
	// 创建一个正则表达式模式
	re := regexp.MustCompile(`\d+`)

	// 使用正则表达式模式匹配字符串
	matches := re.FindString(s)

	// 检查匹配结果
	if len(matches) > 0 {
		// 获取匹配到的第一个数字
		num, err := strconv.Atoi(matches)
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

func RespBodyToJson(content string) ([]models.AddressInfo, error) {

	// 清理可能存在的Markdown代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// 解析JSON数组
	var addressList []models.AddressInfo
	if err := json.Unmarshal([]byte(content), &addressList); err != nil {
		// 尝试解析为单个对象
		var singleAddr models.AddressInfo
		if err2 := json.Unmarshal([]byte(content), &singleAddr); err2 == nil {
			addressList = []models.AddressInfo{singleAddr}
		} else {
			return nil, fmt.Errorf("地址信息解析失败: %w\n原始返回: %s", err, content)
		}
	}
	return addressList, nil
}

// provinceTable 中国 34 个省级行政区（全称+简称）。有序切片而非 map，保证匹配结果确定。
// 全称为存储值（契约 §1.5.3），简称仅作匹配兜底。
var provinceTable = []struct{ Full, Short string }{
	{"北京市", "北京"}, {"天津市", "天津"}, {"上海市", "上海"}, {"重庆市", "重庆"},
	{"河北省", "河北"}, {"山西省", "山西"}, {"辽宁省", "辽宁"}, {"吉林省", "吉林"},
	{"黑龙江省", "黑龙江"}, {"江苏省", "江苏"}, {"浙江省", "浙江"}, {"安徽省", "安徽"},
	{"福建省", "福建"}, {"江西省", "江西"}, {"山东省", "山东"}, {"河南省", "河南"},
	{"湖北省", "湖北"}, {"湖南省", "湖南"}, {"广东省", "广东"}, {"海南省", "海南"},
	{"四川省", "四川"}, {"贵州省", "贵州"}, {"云南省", "云南"}, {"陕西省", "陕西"},
	{"甘肃省", "甘肃"}, {"青海省", "青海"}, {"台湾省", "台湾"},
	{"内蒙古自治区", "内蒙古"}, {"广西壮族自治区", "广西"}, {"西藏自治区", "西藏"},
	{"宁夏回族自治区", "宁夏"}, {"新疆维吾尔自治区", "新疆"},
	{"香港特别行政区", "香港"}, {"澳门特别行政区", "澳门"},
}

// ExtractProvince 从地址字符串提取省级行政区全称；未识别返回空串。
// 两轮匹配（契约 §1.5.3）：① 全称——34 个全称互不包含、无歧义，不限出现位置，按表序取第一个命中；
// ② 简称兜底——须出现在地址前 6 个字符内（≈ 前 18 字节，省份惯例位于中文地址开头），且简称
// 不得紧邻在行政区划后缀（市/区/县）之后——后者规避「南京市上海路」「青岛市广西路」类
// 「城市名/区县名+路名」误判（契约 §1.5.3 注释声称的设计意图；其示例代码的纯 Contains 无法
// 规避路名恰在前 6 字符内的场景，故按注释意图补充该排除条件）。
// 残余误判（如地址开头即他省简称）为已接受风险（契约 §5 Q8），由改单端点人工兜底。
func ExtractProvince(address string) string {
	for _, p := range provinceTable {
		if strings.Contains(address, p.Full) {
			return p.Full
		}
	}
	head := address
	if len(head) > 18 { // UTF-8 中文每字 3 字节，前 6 个字符 ≈ 前 18 字节，粗截断即可
		head = head[:18]
	}
	for _, p := range provinceTable {
		idx := strings.Index(head, p.Short)
		if idx < 0 {
			continue
		}
		if idx >= len("市") { // 前一字符为完整中文且是行政区划后缀时，视为路名/地名成分，跳过
			if prev := head[idx-3 : idx]; prev == "市" || prev == "区" || prev == "县" {
				continue
			}
		}
		return p.Full
	}
	return ""
}

// ProvinceShortName 省级行政区全称 → 简称（provinceTable 映射，契约 §1.5.3）。
// 供省份统计端点（server/order/stats.go）填充 shortName；provinceTable 为有序切片且
// 全称互不包含，首个命中即唯一。非表内全称（如人工改单填入的其他值）原样返回。
func ProvinceShortName(fullName string) string {
	for _, p := range provinceTable {
		if p.Full == fullName {
			return p.Short
		}
	}
	return fullName
}
