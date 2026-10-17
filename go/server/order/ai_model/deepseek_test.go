package ai_model

import (
	"testing"

	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	ordertutil "order_server/go/utils/order"
)

// resetPreviewState 测试隔离：保存并清空内存预览态（B6），测试结束后恢复，
// 避免污染同包其他用例（mergeIntoPreview 直接写 global.AddressList）。
func resetPreviewState(t *testing.T) {
	t.Helper()
	saved := global.AddressList
	global.AddressList = nil
	t.Cleanup(func() { global.AddressList = saved })
}

// TestMergeIntoPreviewAddAssignsNonZeroVid 缺陷1回归（2026-10-17）：
// 追加模式（previewType=ADD）旧写法「先 append 后 SetVid」因切片元素值拷贝，
// 使写入 global.AddressList 的新增行 PreviewID 恒为 0（返回前台的 VID 也为 0），
// 编辑/删除/审核修正按 VID 定位错行。
// 修复后必须满足：RE 批与 ADD 批的 PreviewID 全部非 0、全局唯一、从既有长度继续递增
// （与 qwen.go 非流式分支行为一致），且前台转换后的 VID 与 global 中一致。
func TestMergeIntoPreviewAddAssignsNonZeroVid(t *testing.T) {
	resetPreviewState(t)

	// 首次识别：清空重新识别（RE），2 行
	batch1 := []models.AddressInfo{{Recipient: "张三"}, {Recipient: "李四"}}
	mergeIntoPreview(batch1, enum.RecPreviewType_RE)
	if len(global.AddressList) != 2 {
		t.Fatalf("RE 后 len(global.AddressList) = %d, want 2", len(global.AddressList))
	}

	// 追加识别（ADD），3 行——缺陷1触发场景
	batch2 := []models.AddressInfo{{Recipient: "王五"}, {Recipient: "赵六"}, {Recipient: "钱七"}}
	mergeIntoPreview(batch2, enum.RecPreviewType_ADD)
	if len(global.AddressList) != 5 {
		t.Fatalf("ADD 后 len(global.AddressList) = %d, want 5", len(global.AddressList))
	}

	// 全部行 PreviewID 非 0 且全局唯一
	seen := make(map[uint]bool, len(global.AddressList))
	for i, row := range global.AddressList {
		if row.PreviewID == 0 {
			t.Fatalf("第 %d 行 PreviewID = 0：追加模式下新增行 VID 不得为 0（缺陷1回归）", i)
		}
		if seen[row.PreviewID] {
			t.Fatalf("PreviewID %d 重复，VID 必须全局唯一", row.PreviewID)
		}
		seen[row.PreviewID] = true
	}

	// VID 序列：首批 1,2；追加批 3,4,5（从既有长度继续递增，与 qwen.go 一致）
	for i, want := range []uint{1, 2, 3, 4, 5} {
		if got := global.AddressList[i].PreviewID; got != want {
			t.Fatalf("global.AddressList[%d].PreviewID = %d, want %d", i, got, want)
		}
	}

	// 再次追加（ADD），1 行：VID 继续递增为 6
	batch3 := []models.AddressInfo{{Recipient: "孙八"}}
	mergeIntoPreview(batch3, enum.RecPreviewType_ADD)
	if got := global.AddressList[5].PreviewID; got != 6 {
		t.Fatalf("再次追加行 PreviewID = %d, want 6", got)
	}

	// 前台返回视图（ExceladdressInfo.VID）与 global 中存储值一致
	excel := ordertutil.AddressToExcelAddress(global.AddressList)
	if len(excel) != 6 {
		t.Fatalf("len(excel) = %d, want 6", len(excel))
	}
	for i, want := range []uint{1, 2, 3, 4, 5, 6} {
		if got := excel[i].PreviewID; got != want {
			t.Fatalf("excel[%d].VID = %d, want %d", i, got, want)
		}
	}
}
