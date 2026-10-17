package handler

import (
	"net/http"
	"order_server/go/application"
	"order_server/go/common"

	"github.com/gin-gonic/gin"
)

// GetAIModels 已配置的大模型列表
func GetAIModels(c *gin.Context) {
	common.Ok(c, application.ListAIModels())
}

// Healthz 存活探针，供 nginx 与部署流水线校验进程已就绪。
// 刻意返回裸 JSON 而非统一响应体，探测方无需理解业务码。
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
