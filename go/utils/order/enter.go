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
