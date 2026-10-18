package router

import (
	"net/http"
	"net/http/httptest"
	authutil "order_server/go/utils/auth"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// newAuthProbeEngine 构造仅含 AuthRequired 与探针 handler 的裸 engine。
// 探针 handler 只有在校验通过后才会执行——到达它即证明中间件放行。
// 本文件全部用例都选在中间件触达数据库之前结束（缺令牌/坏令牌/过期/错密钥），
// 与既有测试策略一致：涉及数据库的正例由启动服务后的 curl 联调验证。
func newAuthProbeEngine() (*gin.Engine, *string) {
	reached := ""
	r := gin.New()
	r.GET("/probe", AuthRequired(), func(c *gin.Context) {
		reached = "hit"
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r, &reached
}

func probeWith(t *testing.T, r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestAuthRequiredRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	w := probeWith(t, r, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if *reached != "" {
		t.Fatalf("无令牌不应到达业务 handler")
	}
	if !strings.Contains(w.Body.String(), `"code":40100`) {
		t.Fatalf("响应体应携带业务码 40100，got = %s", w.Body.String())
	}
}

func TestAuthRequiredRejectsGarbageToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	for _, h := range []string{
		"Bearer not-a-jwt",
		"Bearer ",
		"bearer", // 只有方案名没有令牌
	} {
		w := probeWith(t, r, h)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Authorization=%q: status = %d, want 401", h, w.Code)
		}
		if *reached != "" {
			t.Fatalf("Authorization=%q 不应到达业务 handler", h)
		}
	}
}

func TestAuthRequiredAcceptsCaseInsensitiveBearer(t *testing.T) {
	// 方案名大小写不敏感是 RFC 6750 的规定，拒绝 "bearer " 小写形态会是兼容性缺陷。
	// 该用例会走到数据库查询（global.Db 为 nil → gin.Recovery 捕获 panic 返回 500），
	// 所以只断言「不是 401」——足以证明前缀解析已放行。
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	token, err := authutil.Sign(testSecret, "order_web", authutil.Identity{UserID: 1, Username: "u"}, time.Hour)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/probe", AuthRequired(), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Fatalf("小写 bearer 前缀被拒：令牌格式合法，401 说明前缀解析大小写敏感了")
	}
}

func TestAuthRequiredRejectsExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	// 1ns 有效期：签发与解析两次取时之间必然超过 1ns。
	token, err := authutil.Sign(testSecret, "order_web", authutil.Identity{UserID: 1}, time.Nanosecond)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}

	w := probeWith(t, r, "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if *reached != "" {
		t.Fatalf("过期令牌不应到达业务 handler")
	}
	if !strings.Contains(w.Body.String(), "登录已过期") {
		t.Fatalf("过期场景应给出可区分的文案，got = %s", w.Body.String())
	}
}

func TestAuthRequiredRejectsForeignSecretToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	token, err := authutil.Sign("attacker-known-secret", "order_web", authutil.Identity{UserID: 1}, time.Hour)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}

	w := probeWith(t, r, "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（密钥不符不得放行）", w.Code)
	}
	if *reached != "" {
		t.Fatalf("密钥不符的令牌不应到达业务 handler")
	}
}

func TestAuthRequiredRejectsForeignIssuerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	token, err := authutil.Sign(testSecret, "some-other-app", authutil.Identity{UserID: 1}, time.Hour)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}

	w := probeWith(t, r, "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（签发者不符不得放行）", w.Code)
	}
	if *reached != "" {
		t.Fatalf("签发者不符的令牌不应到达业务 handler")
	}
}

// TestAuthRequiredRejectsAlgNoneToken 算法混淆攻击：把头里的 alg 改成 none，
// 期望服务端拒绝。jwt 库的 WithValidMethods(HS256) 负责挡住，
// 此用例固化该防线——将来有人删掉那个参数时测试会变红。
func TestAuthRequiredRejectsAlgNoneToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"uid": 1,
		"usr": "attacker",
		"iss": "order_web",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("构造 none 算法令牌失败: %v", err)
	}

	w := probeWith(t, r, "Bearer "+signed)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（alg=none 必须拒绝）", w.Code)
	}
	if *reached != "" {
		t.Fatalf("alg=none 令牌不应到达业务 handler")
	}
}

// TestAuthRequiredRejectsTamperedPayload 篡改载荷：拿合法令牌，
// 翻转载荷 base64 的一个字符再拼回原签名，期望签名校验失败。
func TestAuthRequiredRejectsTamperedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r, reached := newAuthProbeEngine()

	token, err := authutil.Sign(testSecret, "order_web", authutil.Identity{UserID: 1, Role: "operator"}, time.Hour)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("令牌形态异常: %s", token)
	}
	// 头与载荷之间调换顺序，签名必然不再匹配任一被认可的内容。
	tampered := parts[1] + "." + parts[0] + "." + parts[2]

	w := probeWith(t, r, "Bearer "+tampered)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401（篡改载荷必须拒绝）", w.Code)
	}
	if *reached != "" {
		t.Fatalf("被篡改的令牌不应到达业务 handler")
	}
}
