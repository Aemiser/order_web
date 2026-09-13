package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"order_server/go/conf"
	"order_server/go/global"
	"os"
	"path/filepath"
	"strings"
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

// 校验 /uploads 静态路由确实挂在导出目录上，且能通过百分号编码的中文文件名下载
func TestStaticServesEscapedChineseFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	name := "2026-09-01_2026-09-13结账.xlsx"
	content := []byte("xlsx-bytes")
	if err := os.WriteFile(filepath.Join(global.Config.SavePath.Excel, name), content, 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	r := Setup()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/uploads/"+url.PathEscape(name), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), content) {
		t.Fatalf("body = %q, want %q", w.Body.Bytes(), content)
	}
}

// 目录请求不得枚举文件名（gin.Static 内部使用 Dir(root, false) 屏蔽了 Readdir）
func TestStaticDoesNotListDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	name := "secret-order.xlsx"
	if err := os.WriteFile(filepath.Join(global.Config.SavePath.Excel, name), []byte("x"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	r := Setup()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/uploads/", nil)
	r.ServeHTTP(w, req)

	if strings.Contains(w.Body.String(), name) {
		t.Fatalf("目录响应不应列出文件名，body = %q", w.Body.String())
	}
}
