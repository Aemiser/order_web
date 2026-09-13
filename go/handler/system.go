package handler

import (
	"order_server/go/application"
	"order_server/go/common"

	"github.com/gin-gonic/gin"
)

// GetAIModels 已配置的大模型列表
func GetAIModels(c *gin.Context) {
	common.Ok(c, application.ListAIModels())
}
