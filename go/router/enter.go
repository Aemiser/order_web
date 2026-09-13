package router

import (
	"order_server/go/global"
	"order_server/go/handler"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CORS())

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
