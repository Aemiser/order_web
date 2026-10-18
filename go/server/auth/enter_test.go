package auth

import (
	"testing"
	"time"

	"order_server/go/conf"
	"order_server/go/global"
	authutil "order_server/go/utils/auth"
)

// withConfig 每个用例自带配置隔离，结束后复位，
// 避免包内共享 global.Config 导致用例间相互污染。
func withConfig(t *testing.T, cfg *conf.Config) {
	t.Helper()
	old := global.Config
	global.Config = cfg
	t.Cleanup(func() { global.Config = old })
}

// TestSecretFallsBackToRandomAndStable 密钥未配置时兜底必须非空
// （空密钥意味着任何人都能伪造令牌），且进程内多次调用返回同一值
// ——中途换密钥会让所有已签发令牌立即失效。
func TestSecretFallsBackToRandomAndStable(t *testing.T) {
	withConfig(t, &conf.Config{})

	first := Secret()
	if first == "" {
		t.Fatalf("密钥未配置时兜底结果为空串")
	}
	for i := 0; i < 3; i++ {
		if got := Secret(); got != first {
			t.Fatalf("兜底密钥在多次调用间不稳定")
		}
	}
}

func TestSecretFromConfig(t *testing.T) {
	withConfig(t, &conf.Config{Auth: conf.Auth{Secret: "configured-secret"}})

	if got := Secret(); got != "configured-secret" {
		t.Fatalf("Secret() = %q, want 配置值", got)
	}
}

func TestTTLAndIssuerDefaults(t *testing.T) {
	withConfig(t, &conf.Config{})

	if got := TTL(); got != time.Duration(authutil.DefaultExpireHours)*time.Hour {
		t.Fatalf("TTL() 默认值 = %v", got)
	}
	if got := Issuer(); got != authutil.DefaultIssuer {
		t.Fatalf("Issuer() 默认值 = %q", got)
	}
}

func TestTTLAndIssuerFromConfig(t *testing.T) {
	withConfig(t, &conf.Config{Auth: conf.Auth{ExpireHours: 24, Issuer: "custom"}})

	if got := TTL(); got != 24*time.Hour {
		t.Fatalf("TTL() = %v, want 24h", got)
	}
	if got := Issuer(); got != "custom" {
		t.Fatalf("Issuer() = %q, want custom", got)
	}
}

// TestTTLRejectsBadConfig 非正 expireHours 回退默认而非签出立即过期的令牌。
func TestTTLRejectsBadConfig(t *testing.T) {
	withConfig(t, &conf.Config{Auth: conf.Auth{ExpireHours: -5}})

	if got := TTL(); got <= 0 {
		t.Fatalf("TTL() = %v, 非正有效期会签出签发即过期的令牌", got)
	}
}
