package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order_server/go/common"

	"github.com/gin-gonic/gin"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/order/create", CreateOrder)
	r.POST("/api/order/recognition", Recognition)
	r.POST("/api/order/recognition/stream", RecognitionStream)
	r.GET("/api/order/province-stats", ProvinceStats)
	r.PUT("/api/order/:id", UpdateOrder)
	r.DELETE("/api/order/:id", DeleteOrder)
	r.POST("/api/order/preview/audit", AuditPreview)
	return r
}

func TestCreateOrderRejectsMissingDate(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/create", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestCreateOrderRejectsMalformedJSON(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/create", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

// 以下 ProvinceStats 用例仅覆盖参数校验路径（契约 §3.1 错误码表）。
// 限制说明（任务 T12 / 风险 R10）：go test 环境无真实 DB（global.Db 为 nil），
// 合法参数会进入 DB 统计查询而 panic，故正例（无参全量/成对时间返回 code:0）
// 不在单测中覆盖，由启动服务后的 curl 三态联调验证（CP2）。
func TestProvinceStatsRejectsInvalidStartTime(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/order/province-stats?startTime=2026-10-01&endTime=2026-10-17T23:59:59.999Z", nil)

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam || resp.Msg != "startTime 时间格式错误" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "startTime 时间格式错误")
	}
}

func TestProvinceStatsRejectsInvalidEndTime(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/order/province-stats?startTime=2026-10-01T00:00:00.000Z&endTime=not-a-time", nil)

	r.ServeHTTP(w, req)

	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam || resp.Msg != "endTime 时间格式错误" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "endTime 时间格式错误")
	}
}

func TestProvinceStatsRejectsSingleParam(t *testing.T) {
	r := newRouter()

	// 只传 startTime
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/order/province-stats?startTime=2026-10-01T00:00:00.000Z", nil)
	r.ServeHTTP(w, req)
	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam || resp.Msg != "startTime 与 endTime 必须成对传入" {
		t.Fatalf("only startTime: code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "startTime 与 endTime 必须成对传入")
	}

	// 只传 endTime
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/order/province-stats?endTime=2026-10-17T23:59:59.999Z", nil)
	r.ServeHTTP(w2, req2)
	var resp2 common.Response
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Code != common.CodeBadParam || resp2.Msg != "startTime 与 endTime 必须成对传入" {
		t.Fatalf("only endTime: code = %d, msg = %q, want code %d msg %q", resp2.Code, resp2.Msg, common.CodeBadParam, "startTime 与 endTime 必须成对传入")
	}
}

// --- T13 UpdateOrder/DeleteOrder 用例 ---
// 限制说明（任务 T13 / 风险 R10）：go test 环境无真实 DB（global.Db 为 nil），成功用例与
// 「不存在 id → 50000 订单不存在」会进入 server 层 DB 访问而 panic，故本组仅覆盖 handler
// 参数校验路径（契约 §3.3/§3.4 错误码表 40000 部分）；DB 行为（真实 id 修改留痕、
// 重复删除 50000、删除后 query 不再现）由启动服务后的 curl 联调验证（任务验收标准⑤，CP3）。

func TestUpdateOrderRejectsInvalidID(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/order/abc", strings.NewReader(`{"备注":"测试"}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam || resp.Msg != "id 参数错误" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "id 参数错误")
	}
}

func TestUpdateOrderRejectsNonPositiveID(t *testing.T) {
	r := newRouter()

	// id=0 非正整数 → 40000
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/order/0", strings.NewReader(`{"备注":"测试"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam || resp.Msg != "id 参数错误" {
		t.Fatalf("id=0: code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "id 参数错误")
	}
}

func TestUpdateOrderRejectsEmptyBody(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	// 空 body：ShouldBindJSON 读到 EOF → 40000 请求参数错误（契约 §3.3 错误码表）
	req := httptest.NewRequest(http.MethodPut, "/api/order/1", nil)

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestUpdateOrderRejectsNoFields(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	// {} 绑定成功但六个指针字段全未传 → 40000（契约 §3.3，handler 层判定）
	req := httptest.NewRequest(http.MethodPut, "/api/order/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam || resp.Msg != "至少提供一项需要修改的字段" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "至少提供一项需要修改的字段")
	}
}

func TestDeleteOrderRejectsInvalidID(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/order/abc", nil)

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam || resp.Msg != "id 参数错误" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "id 参数错误")
	}
}

// --- T15 AuditPreview 用例 ---
// 契约 §3.2 错误码表：body 缺失或非 JSON → 40000；预览态为空 → 50000「请先识别订单信息」。
// 预览态为空用例不依赖 DB（audit server 的空态检查在访问 DB 之前，测试进程
// global.AddressList 为 nil），可直接在单测覆盖；识别后含数据的审核联调
// 由启动服务后的 curl 验证（任务 T15 验收标准④，CP3）。

func TestAuditPreviewRejectsMalformedJSON(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	// 非 JSON body → 40000（契约 §3.2 错误码表：body 缺失或非 JSON 对象）
	req := httptest.NewRequest(http.MethodPost, "/api/order/preview/audit", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestAuditPreviewRejectsEmptyBody(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	// 空 body：ShouldBindJSON 读到 EOF → 40000（契约 §3.2：body 缺失 → 40000）
	req := httptest.NewRequest(http.MethodPost, "/api/order/preview/audit", nil)

	r.ServeHTTP(w, req)

	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestAuditPreviewEmptyPreviewState(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	// {} 绑定成功但测试进程 global.AddressList 为 nil（预览态为空）→ 50000
	// 「请先识别订单信息」（契约 §3.2 错误码表）。此路径不碰 DB。
	req := httptest.NewRequest(http.MethodPost, "/api/order/preview/audit", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeError || resp.Msg != "请先识别订单信息" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeError, "请先识别订单信息")
	}
}

// --- M2 安全加固用例（2026-10-17）---
// 识别请求 text 是审核对比基准 global.LastInputText 的唯一入口（AuditRequest 按契约
// §8.1 为空对象、不回传文本），超 200KB 的文本在 handler 层直接 40000 拒绝。
// 仅覆盖拒绝路径：合法长度文本会触发真实 AI 外呼，不在单测中覆盖（R10 同类限制）。

// oversizedTextBody 构造 text 字段超过 200KB 的 JSON 请求体（UTF-8 每汉字 3 字节）。
func oversizedTextBody(t *testing.T) string {
	t.Helper()
	big := strings.Repeat("甲", maxRecognitionTextBytes/3+1)
	body, err := json.Marshal(map[string]string{"text": big})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if n := len(big); n <= maxRecognitionTextBytes {
		t.Fatalf("构造的文本仅 %d 字节，未超过上限 %d", n, maxRecognitionTextBytes)
	}
	return string(body)
}

// TestRecognitionRejectsOversizedText M2：同步识别 text 超 200KB → 40000（不触达 application 层）。
func TestRecognitionRejectsOversizedText(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/recognition", strings.NewReader(oversizedTextBody(t)))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam || resp.Msg != "text 长度超过上限 200KB" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "text 长度超过上限 200KB")
	}
}

// TestRecognitionStreamRejectsOversizedText M2：流式识别 text 超 200KB → 在进入 SSE 之前
// 返回统一 JSON 错误（HTTP 200 + content-type application/json + code=40000，契约 §5 第 2 条）。
func TestRecognitionStreamRejectsOversizedText(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/recognition/stream", strings.NewReader(oversizedTextBody(t)))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json（参数错误必须在进入 SSE 前返回 JSON）", ct)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam || resp.Msg != "text 长度超过上限 200KB" {
		t.Fatalf("code = %d, msg = %q, want code %d msg %q", resp.Code, resp.Msg, common.CodeBadParam, "text 长度超过上限 200KB")
	}
}
