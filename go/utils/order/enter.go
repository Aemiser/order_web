package ordertutil

import (
	"fmt"
	"order_server/go/models"
	"regexp"
	"strconv"
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
	for i := range infolist {
		addr := &infolist[i] // 获取元素的指针
		addr.PreviewID = uint(i + 1)
	}
	return infolist
}
