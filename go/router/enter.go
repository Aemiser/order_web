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

	// 存活探针保持匿名：nginx 与部署流水线不持令牌。
	r.GET("/healthz", handler.Healthz)

	api := r.Group("/api")
	{
		// 登录本身匿名，挂在 /api/auth 前缀下而非 /api/order 下：
		// order 组里 PUT/DELETE /:id 是参数路由，再加字面量子路径
		// 会复现 /preview/:id 与 /:id 那类共存的匹配麻烦。
		api.POST("/auth/login", handler.Login)

		// 其余全部接口必须携带有效令牌。分组挂中间件而非引擎级：
		// /healthz 与 /api/auth/login 需要留在门外。
		authed := api.Group("", AuthRequired())
		{
			authed.GET("/auth/me", handler.Me)
			authed.POST("/auth/logout", handler.Logout)
			authed.GET("/system/ai-models", handler.GetAIModels)

			orderGroup := authed.Group("/order")
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

			checkoutGroup := authed.Group("/checkout")
			{
				checkoutGroup.POST("/daydata", handler.GetDayData)
				checkoutGroup.PUT("/product-belong", handler.SetProductBelong)
				checkoutGroup.POST("/price", handler.SetPrice)
				checkoutGroup.POST("/export", handler.ExportCheckout)
			}
		}
	}

	// 导出文件静态下载；目录未配置时不挂载，避免 http.Dir("") 退化为服务当前工作目录。
	// 纳入鉴权（原先匿名可下载，而导出的 Excel 含收件人姓名手机号地址）。
	// 不用 r.Static：它无法在路由组上挂中间件；serveUpload 走同一套
	// http.FileServer 语义，gin.Dir(dir,false) 关闭目录列举。
	if dir := global.Config.SavePath.Excel; dir != "" {
		uploads := r.Group("/uploads", AuthRequired())
		uploads.GET("/*filepath", serveUpload(dir))
	}

	return r
}

// serveUpload 构造 /uploads/*filepath 的文件服务 handler。
// 单独成函数而非内联闭包：文件名转义与目录列举屏蔽这些行为
// 需要不带鉴权中间件的单测覆盖，解耦后测试可以直接挂在裸 engine 上。
func serveUpload(dir string) gin.HandlerFunc {
	fs := gin.Dir(dir, false)
	return func(c *gin.Context) {
		c.FileFromFS(c.Param("filepath"), fs)
	}
}
