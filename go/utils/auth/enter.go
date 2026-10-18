// Package authutil 汇集登录鉴权用到的密码学与令牌原语。
//
// 刻意不读 global.Config：所有密钥、有效期、签发者都由调用方显式传入，
// 这样签发与解析可以在单测里脱离配置文件与数据库直接验证。
package authutil

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultExpireHours 令牌默认有效期。12 小时覆盖一个工作日：
	// 短到离职/失窃的令牌不会长期可用，长到不会在正常工作中途把人踢出去。
	DefaultExpireHours = 12
	// DefaultIssuer 默认签发者标识。
	DefaultIssuer = "order_web"
	// MaxPasswordLen bcrypt 只取口令前 72 字节，超出部分被静默截断。
	// 与其让「改了末尾几个字符口令仍然一样」这种反直觉行为发生，
	// 不如在入口就拒绝超长口令。
	MaxPasswordLen = 72
)

var (
	// ErrPasswordTooLong 口令超过 bcrypt 的 72 字节有效长度。
	ErrPasswordTooLong = errors.New("口令长度超过上限")
	// ErrTokenInvalid 令牌缺失、格式错误、签名不符或签发者不匹配。
	ErrTokenInvalid = errors.New("令牌无效")
	// ErrTokenExpired 令牌已过期。与 ErrTokenInvalid 分开，
	// 前端据此把提示区分为「登录已过期，请重新登录」与「登录状态无效」。
	ErrTokenExpired = errors.New("令牌已过期")
)

// Identity 令牌中携带的用户身份，签发与解析共用同一结构。
//
// 只放渲染与鉴权必需的最小字段。令牌在客户端是明文可读的
// （签名只防篡改、不防窥视），因此口令哈希等敏感信息一律不进载荷。
type Identity struct {
	UserID   uint   `json:"uid"`
	Username string `json:"usr"`
	Nickname string `json:"nick"`
	Role     string `json:"role"`
}

// claims 内部载荷结构。内嵌 Identity 使其字段平铺进 JSON，
// 内嵌 RegisteredClaims 以满足 jwt/v5 的 Claims 接口（过期、签发时间等校验）。
type claims struct {
	Identity
	jwt.RegisteredClaims
}

// GenerateSecret 生成 32 字节的 URL-safe 随机密钥，供 Secret 留空时兜底。
func GenerateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashPassword 生成 bcrypt 加盐哈希。
//
// cost 用 bcrypt.DefaultCost(10)，单次约 60ms：对登录这种低频操作足够慢，
// 能显著抬高离线爆破成本，又不至于让交互产生可感知的卡顿。
func HashPassword(plain string) (string, error) {
	if len(plain) > MaxPasswordLen {
		return "", ErrPasswordTooLong
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验明文口令与哈希是否匹配。
// bcrypt.CompareHashAndPassword 内部即恒定时间比较，调用方无需再防时序侧信道。
// 哈希为空或格式非法时返回 false 而非 error——对调用方而言二者都是「不匹配」。
func VerifyPassword(hash, plain string) bool {
	if hash == "" || len(plain) > MaxPasswordLen {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// Sign 用 HS256 签发令牌。ttl 必须为正，否则返回错误——
// 非正 ttl 会签出一个签发即过期的令牌，属于调用方 bug，不该静默放过。
func Sign(secret, issuer string, id Identity, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("签名密钥为空")
	}
	if ttl <= 0 {
		return "", errors.New("令牌有效期必须为正")
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Identity: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	return token.SignedString([]byte(secret))
}

// Parse 校验并解析令牌。
//
// WithValidMethods 锁定 HS256：不限定算法会留下算法混淆的口子——
// 攻击者把 alg 改成 none 可绕过签名，或在服务同时持有公私钥时用
// HS256 把公钥当作 HMAC 密钥来伪造签名。
//
// 返回的 error 可用 errors.Is 判别是否为 ErrTokenExpired。
func Parse(secret, issuer, tokenStr string) (Identity, error) {
	var zero Identity
	if secret == "" {
		return zero, ErrTokenInvalid
	}

	parsed, err := jwt.ParseWithClaims(tokenStr, &claims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return zero, ErrTokenExpired
		}
		return zero, ErrTokenInvalid
	}

	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return zero, ErrTokenInvalid
	}
	return c.Identity, nil
}
