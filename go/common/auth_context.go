package common

import (
	authutil "order_server/go/utils/auth"

	"github.com/gin-gonic/gin"
)

// ctxIdentityKey 鉴权中间件写入用户身份用的 context key。
// 放在 common 而非 router 包：router 依赖 handler，handler 若反向引用 router
// 的导出常量会成环；common 是两层都能引用的共享叶子包。
const ctxIdentityKey = "authIdentity"

// SetIdentity 由鉴权中间件在请求校验通过后写入用户身份。
func SetIdentity(c *gin.Context, id authutil.Identity) {
	c.Set(ctxIdentityKey, id)
}

// CurrentIdentity 取当前请求的用户身份。
// 未经鉴权中间件覆盖的路径（如 /healthz、登录接口本身）返回零值。
func CurrentIdentity(c *gin.Context) authutil.Identity {
	if v, ok := c.Get(ctxIdentityKey); ok {
		if id, ok := v.(authutil.Identity); ok {
			return id
		}
	}
	return authutil.Identity{}
}
