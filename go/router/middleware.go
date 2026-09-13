package router

import (
	"net/http"
	"order_server/go/global"
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
