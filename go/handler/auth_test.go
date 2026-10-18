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

func newAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/login", Login)
	return r
}

func doLogin(r *gin.Engine, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) common.Response {
	t.Helper()
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, w.Body.String())
	}
	return resp
}

// 与 order_test 同一策略：只覆盖进入业务层（也就是触库）之前的参数校验路径，
// 正例由启动服务后的 curl 联调验证。
func TestLoginRejectsMalformedJSON(t *testing.T) {
	r := newAuthRouter()

	resp := decode(t, doLogin(r, `not-json`))
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestLoginRejectsEmptyCredentials(t *testing.T) {
	r := newAuthRouter()

	cases := []string{
		`{}`,
		`{"username":"","password":"x"}`,
		`{"username":"u","password":""}`,
	}
	for _, body := range cases {
		resp := decode(t, doLogin(r, body))
		if resp.Code != common.CodeBadParam {
			t.Errorf("body %s: code = %d, want %d", body, resp.Code, common.CodeBadParam)
		}
	}
}

// TestLoginRejectsOverlongInput 长度校验先于查库与 bcrypt：
// 超长输入进 bcrypt 是纯浪费（且超 72 字节的部分本就会被截断）。
func TestLoginRejectsOverlongInput(t *testing.T) {
	r := newAuthRouter()

	longUser := `{"username":"` + strings.Repeat("u", 65) + `","password":"x"}`
	longPass := `{"username":"u","password":"` + strings.Repeat("p", 73) + `"}`
	for _, body := range []string{longUser, longPass} {
		resp := decode(t, doLogin(r, body))
		if resp.Code != common.CodeBadParam {
			t.Errorf("超长输入 code = %d, want %d", resp.Code, common.CodeBadParam)
		}
	}
}

// TestLoginStaysHTTP200 登录失败走统一响应体（HTTP 200 + code）而非 401：
// 前端对 401 的全局动作是踢回登录页，若登录接口自身返回 401 会在登录页
// 内部触发全局拦截。此形态固化后，将来有人「顺手统一」成 401 时测试会变红。
// 单测不触库，因此这里用参数校验路径（空凭据/超长）验证 HTTP 200 语义；
// 用户名不存在等真实业务失败由启动服务后的 curl 联调验证。
func TestLoginStaysHTTP200(t *testing.T) {
	r := newAuthRouter()

	cases := []string{
		`{}`,
		`{"username":"","password":"x"}`,
		`{"username":"` + strings.Repeat("u", 65) + `","password":"x"}`,
	}
	for _, body := range cases {
		w := doLogin(r, body)
		if w.Code != http.StatusOK {
			t.Fatalf("登录接口 HTTP 状态 = %d, want 200（业务码区分失败）", w.Code)
		}
	}
}
