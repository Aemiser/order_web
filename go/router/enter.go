package router

import (
	"order_server/go/global"
	"order_server/go/handler"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(AccessLog())
	r.Use(CORS())

	r.GET("/healthz", handler.Healthz)

	api := r.Group("/api")
	{
		api.GET("/system/ai-models", handler.GetAIModels)

		orderGroup := api.Group("/order")
		{
			orderGroup.POST("/recognition", handler.Recognition)
			orderGroup.POST("/recognition/stream", handler.RecognitionStream)
			orderGroup.GET("/preview", handler.GetPreview)
			orderGroup.PUT("/preview", handler.SetPreview)
			orderGroup.DELETE("/preview/:id", handler.DeletePreview)
			orderGroup.POST("/create", handler.CreateOrder)
			orderGroup.POST("/export", handler.ExportOrder)
			orderGroup.POST("/query", handler.OrderQuery)
			orderGroup.GET("/province-stats", handler.ProvinceStats)
			// T15：预览审核端点（契约 §3.2，空 JSON 入参；原始文本由后端 LastInputText 暂存）
			orderGroup.POST("/preview/audit", handler.AuditPreview)
			// T13：历史订单删改端点（契约 §3.3/§3.4）。:id 为 address_infos 主键，
			// 与既有 /preview/:id（预览 VID 删除）共存：gin 静态路由优先匹配，
			// /api/order/preview 仍命中预览端点，其余命中 :id。
			orderGroup.PUT("/:id", handler.UpdateOrder)
			orderGroup.DELETE("/:id", handler.DeleteOrder)
		}

		checkoutGroup := api.Group("/checkout")
		{
			checkoutGroup.POST("/daydata", handler.GetDayData)
			checkoutGroup.PUT("/product-belong", handler.SetProductBelong)
			checkoutGroup.POST("/price", handler.SetPrice)
			checkoutGroup.POST("/export", handler.ExportCheckout)
		}
	}

	// 导出文件静态下载；目录未配置时不挂载，避免 http.Dir("") 退化为服务当前工作目录
	if dir := global.Config.SavePath.Excel; dir != "" {
		r.Static("/uploads", dir)
	}

	return r
}
