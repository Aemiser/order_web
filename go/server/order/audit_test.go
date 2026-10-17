package order

import (
	"sync/atomic"
	"testing"

	"order_server/go/global"
	"order_server/go/models"
)

// saveAuditGlobals 测试隔离：保存审核所依赖的内存态（B6：AddressList / LastInputText /
// RecognitionInFlight），结束后恢复，避免污染同包其他用例。
func saveAuditGlobals(t *testing.T) {
	t.Helper()
	savedList := global.AddressList
	savedText := global.LastInputText
	savedInFlight := global.RecognitionInFlight
	t.Cleanup(func() {
		global.AddressList = savedList
		global.LastInputText = savedText
		atomic.StoreInt32(&global.RecognitionInFlight, savedInFlight)
	})
}

// TestAuditPreviewDataRejectsInFlight 缺陷2回归（2026-10-17）：识别在途时审核必须拒绝。
// 场景：识别入口已先行更新 LastInputText（追加批判入基准文本），但 AddressList 尚未
// 写入新增批（AI 未返回）——若不拒绝，程序对比会把整批未落定订单判为 missing 误报。
func TestAuditPreviewDataRejectsInFlight(t *testing.T) {
	saveAuditGlobals(t)
	// 模拟在途：已有 1 行预览 + 基准文本已追加新批、预览表尚无对应行
	global.AddressList = []models.AddressInfo{
		{PreviewID: 1, Recipient: "张三", Phone: "13800000001", Address: "北京市东城区1号"},
	}
	global.LastInputText = "张三 13800000001\n北京市东城区1号\n李四 13900000002\n上海市黄浦区2号"
	atomic.StoreInt32(&global.RecognitionInFlight, 1)

	_, err := AuditPreviewData(AuditRequest{})
	if err == nil {
		t.Fatal("识别在途时审核必须返回错误，实际返回 nil")
	}
	if err.Error() != "识别进行中，请稍后再审核" {
		t.Fatalf("err = %q, want %q", err.Error(), "识别进行中，请稍后再审核")
	}
}

// TestAuditPreviewDataNormalWhenIdle 缺陷2验收：识别不在途（计数归零 = 识别已落定）时，
// 一致的预览态与基准文本对比全部 normal，不产生 missing/wrong/extra 误报。
// 构造的用例无可疑项 → 不触发 AI 复核，全程无 DB / 无外部调用。
func TestAuditPreviewDataNormalWhenIdle(t *testing.T) {
	saveAuditGlobals(t)
	atomic.StoreInt32(&global.RecognitionInFlight, 0)
	global.AddressList = []models.AddressInfo{
		{PreviewID: 1, Recipient: "张三", Phone: "13800000001", Address: "北京市东城区1号"},
	}
	global.LastInputText = "张三 13800000001\n北京市东城区1号"

	resp, err := AuditPreviewData(AuditRequest{})
	if err != nil {
		t.Fatalf("不在途时审核不应报错: %v", err)
	}
	if resp.AiFailed {
		t.Fatal("无可疑项不应触发 AI 复核，aiFailed 应为 false")
	}
	if resp.Summary.Missing != 0 {
		t.Fatalf("missing = %d, want 0（一致状态不得误报缺单）", resp.Summary.Missing)
	}
	if resp.Summary.Wrong != 0 {
		t.Fatalf("wrong = %d, want 0", resp.Summary.Wrong)
	}
	if resp.Summary.Extra != 0 {
		t.Fatalf("extra = %d, want 0", resp.Summary.Extra)
	}
	if resp.Summary.Normal != 1 {
		t.Fatalf("normal = %d, want 1", resp.Summary.Normal)
	}
	if len(resp.Items) != 1 || resp.Items[0].DiffType != diffTypeNormal {
		t.Fatalf("items = %+v, want 1 条 normal", resp.Items)
	}
}
