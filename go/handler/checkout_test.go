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

func TestGetDayDataRejectsZeroTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/checkout/daydata", GetDayData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/daydata", strings.NewReader(`{"startTime":"0001-01-01T00:00:00Z","endTime":"0001-01-01T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestSetProductBelongBindsMap(t *testing.T) {
	t.Chdir(t.TempDir())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/checkout/product-belong", SetProductBelong)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/checkout/product-belong", strings.NewReader(`{"香水柠檬":["5斤柠檬"]}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeOK {
		t.Fatalf("code = %d, want %d, msg = %s", resp.Code, common.CodeOK, resp.Msg)
	}
}

// doSetPriceRequest 向 SetPrice 端点发送 body 并解析统一响应，返回 (code, msg)。
func doSetPriceRequest(t *testing.T, body string) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/checkout/price", SetPrice)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/price", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Code, resp.Msg
}

// TestSetPriceRejectsEmptyList 契约 §3.6：空数组 → 40000「价格列表不能为空」
func TestSetPriceRejectsEmptyList(t *testing.T) {
	code, msg := doSetPriceRequest(t, `[]`)
	if code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", code, common.CodeBadParam)
	}
	if msg != "价格列表不能为空" {
		t.Fatalf("msg = %q, want %q", msg, "价格列表不能为空")
	}
}

// TestSetPriceRejectsNonArrayBody 契约 §3.6：body 非数组（绑定失败）→ 40000「价格列表不能为空」
func TestSetPriceRejectsNonArrayBody(t *testing.T) {
	code, msg := doSetPriceRequest(t, `{"productName":"香水柠檬"}`)
	if code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", code, common.CodeBadParam)
	}
	if msg != "价格列表不能为空" {
		t.Fatalf("msg = %q, want %q", msg, "价格列表不能为空")
	}
}

// TestSetPriceRejectsMissingProductOrSpec 契约 §3.6：某项 productName/spec 为空 → 40000
func TestSetPriceRejectsMissingProductOrSpec(t *testing.T) {
	// 第 1 项缺 productName
	code, msg := doSetPriceRequest(t, `[{"spec":"5斤","pricePerKg":8.5}]`)
	if code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", code, common.CodeBadParam)
	}
	if msg != "第 1 项 productName 不能为空" {
		t.Fatalf("msg = %q, want %q", msg, "第 1 项 productName 不能为空")
	}

	// 第 2 项缺 spec
	code, msg = doSetPriceRequest(t, `[{"productName":"香水柠檬","spec":"5斤","pricePerKg":8.5},{"productName":"香水柠檬","pricePerKg":8.5}]`)
	if code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", code, common.CodeBadParam)
	}
	if msg != "第 2 项 spec 不能为空" {
		t.Fatalf("msg = %q, want %q", msg, "第 2 项 spec 不能为空")
	}
}

// TestSetPriceRejectsOversizedList M1 安全加固（2026-10-17）：
// 数组超过 500 条上限 → 40000「价格列表数量超过上限 500」（handler 层校验，
// 校验先于 application/server 调用，不触达 DB）。
func TestSetPriceRejectsOversizedList(t *testing.T) {
	var body strings.Builder
	body.WriteString(`[`)
	for i := 0; i < maxPriceConfigItems+1; i++ {
		if i > 0 {
			body.WriteString(",")
		}
		body.WriteString(`{"productName":"香水柠檬","spec":"5斤"}`)
	}
	body.WriteString(`]`)

	code, msg := doSetPriceRequest(t, body.String())
	if code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", code, common.CodeBadParam)
	}
	if msg != "价格列表数量超过上限 500" {
		t.Fatalf("msg = %q, want %q", msg, "价格列表数量超过上限 500")
	}
}
