package application

import (
	"errors"
	"fmt"
	"time"

	"order_server/go/common"
	"order_server/go/models"
	"order_server/go/server/auth"
	authutil "order_server/go/utils/auth"
)

// AuthError 把 server/auth 的失败翻译成带统一业务码的结构。
// handler 只认 application 层的错误类型（handler 不 import server 包，
// 与既有分层一致），码值映射收在这个边界上。
type AuthError struct {
	Code int
	Msg  string
}

func (e *AuthError) Error() string { return e.Msg }

// AuthSecret / AuthIssuer / AuthTTL 供鉴权中间件取生效配置。
func AuthSecret() string { return auth.Secret() }
func AuthIssuer() string { return auth.Issuer() }
func AuthTTL() time.Duration { return auth.TTL() }

// LoginResult 对前端暴露的登录产物。User 字段是视图模型而非 models.User：
// 尽管该模型的 Password 已 json:"-"，在边界上再收敛一次字段集，
// 响应体里出现什么就不再取决于模型定义。
type LoginResult struct {
	Token     string
	ExpiresIn time.Duration
	Username  string
	Nickname  string
	Role      string
}

// Login 登录。
func Login(username, password, ip string) (*LoginResult, error) {
	result, err := auth.Login(username, password, ip)
	if err != nil {
		var locked *auth.LockedError
		switch {
		case errors.As(err, &locked):
			return nil, &AuthError{Code: common.CodeTooManyRequests, Msg: locked.Error()}
		case errors.Is(err, auth.ErrBadCredentials):
			return nil, &AuthError{Code: common.CodeUnauthorized, Msg: err.Error()}
		case errors.Is(err, auth.ErrDisabled):
			return nil, &AuthError{Code: common.CodeForbidden, Msg: err.Error()}
		}
		return nil, fmt.Errorf("登录失败: %w", err)
	}
	return &LoginResult{
		Token:     result.Token,
		ExpiresIn: result.ExpiresIn,
		Username:  result.User.Username,
		Nickname:  result.User.Nickname,
		Role:      result.User.Role,
	}, nil
}

// CurrentUser 按用户主键取当前账号。中间件与 /auth/me 共用。
func CurrentUser(id uint) (*models.User, error) {
	return auth.FindEnabledByID(id)
}

// Logout 退出留痕。
func Logout(id authutil.Identity, ip string) {
	auth.Logout(id.Username, id.Role, ip)
}
