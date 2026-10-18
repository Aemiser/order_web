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

// testSecret 单测用的固定签名密钥。经由 global.Config.Auth.Secret 注入，
// 中间件与登录路径都从 Secret() 读取它；留空会走「进程内随机密钥」兜底，
// 每个用例各自随机的话，签出的令牌就无法互验了。
const testSecret = "unit-test-secret-do-not-use-in-prod"

func setupGlobal(t *testing.T) {
	t.Helper()
	global.Config = &conf.Config{}
	global.Config.SavePath.Excel = t.TempDir()
	global.Config.Server.AllowOrigins = []string{"http://localhost:3000"}
	global.Config.Auth.Secret = testSecret
	global.Config.Auth.Issuer = "order_web"
}

// TestSetupRegistersAllEndpoints 集合相等断言：既防止漏注册（want ⊆ got
// 能查出来的），也防止悄悄多出匿名可访问的路由（got ⊆ want）。
// 后者正是本次接入鉴权时最危险的回归形态——路由挂错组就是开了个匿名后门。
func TestSetupRegistersAllEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	want := []string{
		"GET /healthz",
		"POST /api/auth/login",
		"GET /api/auth/me",
		"POST /api/auth/logout",
		"GET /api/system/ai-models",
		"POST /api/order/recognition",
		"POST /api/order/recognition/stream",
		"GET /api/order/preview",
		"PUT /api/order/preview",
		"DELETE /api/order/preview/:id",
		"POST /api/order/create",
		"POST /api/order/export",
		"POST /api/order/query",
		"GET /api/order/province-stats",
		"POST /api/order/preview/audit",
		"PUT /api/order/:id",
		"DELETE /api/order/:id",
		"POST /api/checkout/daydata",
		"PUT /api/checkout/product-belong",
		"POST /api/checkout/price",
		"POST /api/checkout/export",
		"GET /uploads/*filepath",
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
	for g := range got {
		found := false
		for _, w := range want {
			if g == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("出现未登记的额外路由 %s", g)
		}
	}
}

func TestHealthzReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %s", w.Body.String())
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
		t.Fatalf("Access-Control-Allow-Origin = %q, got = %q", got, w.Code)
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

// TestUploadsAnonymousRejected /uploads 纳入鉴权后的接线验证：
// 不带令牌访问导出文件必须 401。走完整 Setup() 而非裸中间件，
// 确认中间件真的挂在了这个路由组上（挂漏了就是匿名下载回归）。
// 带有效令牌的正例依赖数据库查账号现状，与既有测试策略一致，
// 由启动服务后的 curl 联调验证（见 handler 测试的说明）。
func TestUploadsAnonymousRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	name := "2026-09-01_2026-09-13结账.xlsx"
	if err := os.WriteFile(filepath.Join(global.Config.SavePath.Excel, name), []byte("xlsx-bytes"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	r := Setup()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/uploads/"+url.PathEscape(name), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（/uploads 已纳入鉴权）", w.Code)
	}
	if w.Body.Len() != 0 && bytes.Contains(w.Body.Bytes(), []byte("xlsx-bytes")) {
		t.Fatalf("未认证请求不应拿到文件内容")
	}
}

// TestUploadsAnonymousAPIRejected 任意 API 不带令牌同样 401。
// 单独列出而不并入上面的用例：API 与静态文件走不同注册路径，
// 两处都可能各自挂漏。
func TestUploadsAnonymousAPIRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/system/ai-models", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（API 已纳入鉴权）", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":40100`) {
		t.Fatalf("响应体应携带统一业务码 40100，got = %s", w.Body.String())
	}
}

// TestServeUploadEscapedFilename 校验文件服务 handler 能通过百分号编码的
// 中文文件名下载。直接挂 serveUpload（不含鉴权中间件）以聚焦文件服务行为
// 本身；鉴权接线由 TestUploadsAnonymousRejected 覆盖。
func TestServeUploadEscapedFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	name := "2026-09-01_2026-09-13结账.xlsx"
	content := []byte("xlsx-bytes")
	if err := os.WriteFile(filepath.Join(global.Config.SavePath.Excel, name), content, 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	r := gin.New()
	r.GET("/uploads/*filepath", serveUpload(global.Config.SavePath.Excel))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/uploads/"+url.PathEscape(name), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), content) {
		t.Fatalf("body = %q, want = %q", w.Body.Bytes(), content)
	}
}

// TestServeUploadDoesNotListDirectory 目录请求不得枚举文件名
// （gin.Dir(root,false) 屏蔽了 Readdir，改造前 r.Static 同款行为）。
func TestServeUploadDoesNotListDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	name := "secret-order.xlsx"
	if err := os.WriteFile(filepath.Join(global.Config.SavePath.Excel, name), []byte("x"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	r := gin.New()
	r.GET("/uploads/*filepath", serveUpload(global.Config.SavePath.Excel))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/", nil))

	if strings.Contains(w.Body.String(), name) {
		t.Fatalf("目录响应不应列出文件名，body = %q", w.Body.String())
	}
}

// TestServeUploadRejectsTraversal 路径穿越防护：.. 序列经 http.FileServer
// 的 path.Clean 规整后不允许逃出导出目录。
func TestServeUploadRejectsTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	outside := filepath.Join(filepath.Dir(global.Config.SavePath.Excel), "outside-secret.txt")
	if err := os.WriteFile(outside, []byte("leaked"), 0644); err != nil {
		t.Fatalf("写入目录外测试文件失败: %v", err)
	}
	t.Cleanup(func() { os.Remove(outside) })

	r := gin.New()
	r.GET("/uploads/*filepath", serveUpload(global.Config.SavePath.Excel))

	// httptest.NewRequest 会把 URL 里的 .. 当相对段规整掉，
	// 所以这里用编码后的形态直接构造。
	for _, p := range []string{"/uploads/..%2Foutside-secret.txt", "/uploads/%2e%2e%2foutside-secret.txt"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		r.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			t.Fatalf("路径 %s 不应命中目录外文件", p)
		}
		if bytes.Contains(w.Body.Bytes(), []byte("leaked")) {
			t.Fatalf("路径 %s 泄露了目录外文件内容", p)
		}
	}
}
