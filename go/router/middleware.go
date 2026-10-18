package router

import (
	"errors"
	"net/http"
	"order_server/go/application"
	"order_server/go/common"
	"order_server/go/global"
	authutil "order_server/go/utils/auth"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AccessLog 记录访问日志；走 logrus 以便与应用其余日志写入同一份文件。
// 日志格式化器只输出 Message，因此明细拼进消息文本而非 Fields。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logrus.Infof("访问日志 %s %s %d %s %s",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(),
			time.Since(start).String(), c.ClientIP())
	}
}

// AuthRequired 校验 Authorization: Bearer <JWT>，并确认账号当前仍启用。
//
// fail-closed：缺令牌、格式错、签名不符、已过期、账号不存在、被禁用、
// 查库失败，一律 401 拒绝。宁可让合法请求看到明确报错，不可放行未验证
// 请求——查库失败时若放行，「数据库挂了」会演变成「鉴权整体失效」。
//
// 每请求查一次库是刻意的：令牌无状态，签发之后账号被禁用或删除无法
// 从令牌本身感知（见 server/auth.FindEnabledByID 说明）。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		identity, err := authutil.Parse(application.AuthSecret(), application.AuthIssuer(), token)
		if err != nil {
			abortUnauthorized(c, err)
			return
		}

		user, err := application.CurrentUser(identity.UserID)
		if err != nil || user == nil || !user.Enabled {
			// 记 warn 便于发现拿着旧令牌乱撞的已禁用账号；
			// 提示语保持含糊，不区分「不存在」与「被禁用」。
			logrus.Warnf("鉴权拒绝 用户不存在或已禁用 uid=%d ip=%s", identity.UserID, c.ClientIP())
			abortUnauthorized(c, errors.New("账号不可用"))
			return
		}

		// 身份取库里的现值而非令牌快照：角色、昵称的变更即时生效。
		common.SetIdentity(c, authutil.Identity{
			UserID:   user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Role:     user.Role,
		})
		c.Next()
	}
}

// abortUnauthorized 401 拒绝。文案区分「过期」与「无效」，
// 细节错误不外泄——解析失败的原因属于内部信息。
func abortUnauthorized(c *gin.Context, err error) {
	msg := "请先登录"
	if errors.Is(err, authutil.ErrTokenExpired) {
		msg = "登录已过期，请重新登录"
	}
	c.Header("WWW-Authenticate", `Bearer realm="order_web"`)
	common.Unauthorized(c, msg)
}

// bearerToken 提取 Authorization 头里的 Bearer 令牌。
// 方案名按 RFC 6750 大小写不敏感；无 Bearer 前缀的裸值不接受——
// 显式前缀避免把任意头内容当令牌去解析。
func bearerToken(header string) string {
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// CORS 仅对 allowOrigins 白名单内的来源放开跨域
func CORS() gin.HandlerFunc {
	allowOrigins := global.Config.Server.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")
		if origin != "" && contains(allowOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
