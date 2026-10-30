package wuyin

import (
	"encoding/json"
	"errors"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	wuyinutil "order_server/go/utils/wuyin"
)

// Package wuyin 吴饮（吾饮良品）品牌链路的业务层，与 server/order（一点点）平行。
// 决策（2026-10-29）：纯件计价、独立数据库 global.DbWuyin、标签按发送人关键词匹配。

// ErrDBNotConfigured 所有 wuyin 端点的前置校验统一错误
var ErrDBNotConfigured = errors.New("吴饮数据库未配置（settings.yaml dbWuyin）")

// CheckDB wuyin 端点入口校验：未配置独立库时统一拒绝
func CheckDB() error {
	if global.DbWuyin == nil {
		return ErrDBNotConfigured
	}
	return nil
}

// OrderRecWuyin 吴饮识别统一入口（同步）
func OrderRecWuyin(inputText string, model enum.AIModelType, previewType enum.RecPreviewType) []wuyinutil.WuyinExceladdressInfo {
	addressList, err := recognizeWuyin(model, inputText)
	if err != nil {
		fmt.Printf("吴饮识别失败: %v\n", err)
		return wuyinutil.WuyinToExcelAddress(global.WuyinAddressList)
	}
	logWuyinRec("吴饮 API识别", inputText, addressList)

	mergeIntoWuyinPreview(applyTags(addressList), previewType)
	return wuyinutil.WuyinToExcelAddress(global.WuyinAddressList)
}

type StreamCallback func(data string) error
type DoneCallback func()

// StreamOrderRecWuyin 吴饮识别统一入口（SSE 流式）。
// 增量片段解析出完整记录即回调（与一点点 StreamRecDeepseek 行为同构），
// 流结束回调全量列表 + DONE 由 handler 补发。
func StreamOrderRecWuyin(inputText string, model enum.AIModelType, previewType enum.RecPreviewType, callback StreamCallback, doneCallback DoneCallback) {
	streamRecognizeWuyin(model, inputText, func(content string) error {
		addressList, err := wuyinutil.RespBodyToJsonWuyin(content)
		if err != nil {
			return nil
		}
		if len(addressList) > 0 {
			mergeIntoWuyinPreview(applyTags(addressList), previewType)
			excelData := wuyinutil.WuyinToExcelAddress(global.WuyinAddressList)
			jsonData, _ := json.Marshal(excelData)
			return callback(string(jsonData))
		}
		return nil
	}, func() {
		excelData := wuyinutil.WuyinToExcelAddress(global.WuyinAddressList)
		jsonData, _ := json.Marshal(excelData)
		callback(string(jsonData))
		doneCallback()
	})
}

// mergeIntoWuyinPreview 按 RE/ADD 写入预览（VID 先赋值再写入，同一点点 mergeIntoPreview 的缺陷修复结论）
func mergeIntoWuyinPreview(addressList []models.WuyinAddressInfo, previewType enum.RecPreviewType) {
	addressList = wuyinutil.SetVidWuyin(addressList)
	if previewType == enum.RecPreviewType_ADD {
		global.WuyinAddressList = append(global.WuyinAddressList, addressList...)
	} else {
		global.WuyinAddressList = addressList
	}
}
