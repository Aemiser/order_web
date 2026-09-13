package router

import (
	"net/http"
	"net/http/httptest"
	"order_server/go/conf"
	"order_server/go/global"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupGlobal(t *testing.T) {
	t.Helper()
	global.Config = &conf.Config{}
	global.Config.SavePath.Excel = t.TempDir()
	global.Config.Server.AllowOrigins = []string{"http://localhost:3000"}
}

func TestSetupRegistersAllEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	want := []string{
		"GET /api/system/ai-models",
		"POST /api/order/recognition",
		"POST /api/order/recognition/stream",
		"GET /api/order/preview",
		"PUT /api/order/preview",
		"DELETE /api/order/preview/:id",
		"POST /api/order/create",
		"POST /api/order/export",
		"POST /api/order/query",
		"POST /api/checkout/daydata",
		"PUT /api/checkout/product-belong",
		"POST /api/checkout/price",
		"POST /api/checkout/export",
	}

	got := make(map[string]bool)
	for _, r := range Setup().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("缺少路由 %s", w)
		}
	}
}

func TestCORSPreflightAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/checkout/daydata", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/checkout/daydata", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("未授权来源不应返回 CORS 头，得到 %q", got)
	}
}
