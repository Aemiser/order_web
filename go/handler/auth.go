package handler

import (
	"errors"
	"fmt"

	"order_server/go/application"
	"order_server/go/common"
	authutil "order_server/go/utils/auth"

	"github.com/gin-gonic/gin"
)

const (
	maxUsernameLen = 64
	maxPasswordLen = authutil.MaxPasswordLen
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginUser struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// Login POST /api/auth/login。
//
// 本接口的失败一律走统一响应体（HTTP 200 + code）而非 401：
// 前端对 HTTP 401 的全局动作是「踢回登录页」，若登录接口本身返回 401，
// 会在登录页内部触发那次全局拦截，把用户正在看的错误提示冲掉。
// 「登录没成功」是业务结果，「已登录但凭据失效」才是传输层拒绝。
func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	// 长度先于任何查库与 bcrypt 截断：超长输入除了让 bcrypt 白跑 60ms 没有任何用途。
	if req.Username == "" || len(req.Username) > maxUsernameLen {
		common.BadParam(c, "用户名不能为空且长度不超过 64")
		return
	}
	if req.Password == "" || len(req.Password) > maxPasswordLen {
		common.BadParam(c, "口令不能为空且长度不超过 72")
		return
	}

	result, err := application.Login(req.Username, req.Password, c.ClientIP())
	if err != nil {
		var authErr *application.AuthError
		if errors.As(err, &authErr) {
			common.Err(c, authErr.Code, authErr.Msg)
			return
		}
		common.Fail(c, err)
		return
	}

	common.Ok(c, gin.H{
		"token":     result.Token,
		"expiresIn": int(result.ExpiresIn.Seconds()),
		"user": loginUser{
			Username: result.Username,
			Nickname: result.Nickname,
			Role:     result.Role,
		},
	})
}

// Me GET /api/auth/me。返回最近一次中间件查库的账号现状
// （昵称、角色改动能即时反映，不依赖令牌签发时的快照）。
func Me(c *gin.Context) {
	identity := common.CurrentIdentity(c)
	user, err := application.CurrentUser(identity.UserID)
	if err != nil {
		common.Fail(c, fmt.Errorf("账号信息读取失败: %w", err))
		return
	}
	common.Ok(c, loginUser{
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
	})
}

// Logout POST /api/auth/logout。服务端只做审计留痕，令牌的真正失效
// 由前端丢弃完成（见 server/auth.Logout 的取舍说明）。
func Logout(c *gin.Context) {
	application.Logout(common.CurrentIdentity(c), c.ClientIP())
	common.Ok(c, nil)
}
