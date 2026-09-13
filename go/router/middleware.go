package router

import (
	"net/http"
	"order_server/go/global"

	"github.com/gin-gonic/gin"
)

// CORS 仅对 allowOrigins 白名单内的来源放开跨域
func CORS() gin.HandlerFunc {
	allowOrigins := global.Config.Server.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && contains(allowOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
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
