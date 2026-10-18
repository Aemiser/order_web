package authutil

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testSecret = "unit-test-secret-0123456789abcdef"
	testIssuer = "order_web_test"
)

var testIdentity = Identity{UserID: 7, Username: "张三", Nickname: "三哥", Role: "operator"}

func TestSignParseRoundtrip(t *testing.T) {
	token, err := Sign(testSecret, testIssuer, testIdentity, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if token == "" || strings.Count(token, ".") != 2 {
		t.Fatalf("令牌形态异常: %q", token)
	}

	got, err := Parse(testSecret, testIssuer, token)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got != testIdentity {
		t.Fatalf("解析结果 = %+v, want %+v", got, testIdentity)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	token, _ := Sign(testSecret, testIssuer, testIdentity, time.Hour)

	if _, err := Parse("another-secret-entirely", testIssuer, token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("错密钥应返回 ErrTokenInvalid，got = %v", err)
	}
}

func TestParseRejectsWrongIssuer(t *testing.T) {
	token, _ := Sign(testSecret, testIssuer, testIdentity, time.Hour)

	if _, err := Parse(testSecret, "some-other-app", token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("错签发者应返回 ErrTokenInvalid，got = %v", err)
	}
}

func TestParseRejectsExpired(t *testing.T) {
	// 1ns 有效期：签发与解析两次取时之间必然超过 1ns。
	token, _ := Sign(testSecret, testIssuer, testIdentity, time.Nanosecond)

	_, err := Parse(testSecret, testIssuer, token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("过期令牌应返回 ErrTokenExpired，got = %v", err)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	cases := []string{"", "not-a-token", "a.b", "a.b.c.d", "。。。"}
	for _, c := range cases {
		if _, err := Parse(testSecret, testIssuer, c); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("Parse(%q) 应返回 ErrTokenInvalid，got = %v", c, err)
		}
	}
}

func TestParseRejectsEmptySecretConfig(t *testing.T) {
	token, _ := Sign(testSecret, testIssuer, testIdentity, time.Hour)

	if _, err := Parse("", testIssuer, token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("空密钥应拒绝而非放行，got = %v", err)
	}
}

func TestSignRejectsBadInputs(t *testing.T) {
	if _, err := Sign("", testIssuer, testIdentity, time.Hour); err == nil {
		t.Fatalf("空密钥签发应报错")
	}
	if _, err := Sign(testSecret, testIssuer, testIdentity, 0); err == nil {
		t.Fatalf("零有效期签发应报错")
	}
	if _, err := Sign(testSecret, testIssuer, testIdentity, -time.Hour); err == nil {
		t.Fatalf("负有效期签发应报错")
	}
}

func TestHashVerifyPassword(t *testing.T) {
	hash, err := HashPassword("正确口令示例123")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	if len(hash) < 50 {
		t.Fatalf("bcrypt 哈希长度异常: %d", len(hash))
	}
	// 相同口令两次哈希结果不同（盐随机）。
	hash2, _ := HashPassword("正确口令示例123")
	if hash == hash2 {
		t.Fatalf("两次哈希结果相同，盐未生效")
	}

	if !VerifyPassword(hash, "正确口令示例123") {
		t.Fatalf("正确口令校验失败")
	}
	if VerifyPassword(hash, "错误口令") {
		t.Fatalf("错误口令不应通过")
	}
	if VerifyPassword(hash, "") {
		t.Fatalf("空口令不应通过")
	}
	if VerifyPassword("", "任意") {
		t.Fatalf("空哈希不应通过")
	}
	if VerifyPassword("not-a-bcrypt-hash", "任意") {
		t.Fatalf("非法哈希不应通过")
	}
}

// TestHashPasswordRejectsOver72Bytes bcrypt 只取口令前 72 字节，
// 超出部分被静默截断——「改了末尾口令却仍然能登录」比拒绝更反直觉，
// 因此在入口直接拒绝。
func TestHashPasswordRejectsOver72Bytes(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", MaxPasswordLen+1)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("超长口令应返回 ErrPasswordTooLong，got = %v", err)
	}
	// 恰好 72 字节仍然可用。
	if _, err := HashPassword(strings.Repeat("a", MaxPasswordLen)); err != nil {
		t.Fatalf("72 字节口令不应被拒: %v", err)
	}
	if VerifyPassword("x", strings.Repeat("a", MaxPasswordLen+1)) {
		t.Fatalf("超长口令在 Verify 侧也应直接判不匹配")
	}
}

func TestGenerateSecret(t *testing.T) {
	a, err := GenerateSecret()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	b, _ := GenerateSecret()
	if len(a) < 40 {
		t.Fatalf("随机密钥长度不足: %d", len(a))
	}
	if a == b {
		t.Fatalf("两次生成的密钥相同")
	}
}
