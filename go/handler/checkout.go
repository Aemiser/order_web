package handler

import (
	"time"

	"order_server/go/application"
	"order_server/go/common"
	"order_server/go/server/checkout"

	"github.com/gin-gonic/gin"
)

type dayDataRequest struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// GetDayData 按日期范围查询结账数据
func GetDayData(c *gin.Context) {
	var req dayDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		common.BadParam(c, "startTime/endTime 不能为空")
		return
	}
	common.Ok(c, application.GetDayData(req.StartTime, req.EndTime))
}

// SetProductBelong 保存商品归属配置
func SetProductBelong(c *gin.Context) {
	var mapp map[string][]string
	if err := c.ShouldBindJSON(&mapp); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	application.SetProductBelong(mapp)
	common.Ok(c, nil)
}

// SetPrice 保存价格并返回更新的结账数据
func SetPrice(c *gin.Context) {
	var list []checkout.ProPriceRequest
	if err := c.ShouldBindJSON(&list); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	common.Ok(c, application.SetPrice(list))
}

// ExportCheckout 生成财务账单 Excel，返回下载地址
func ExportCheckout(c *gin.Context) {
	url, err := application.ExportCheckout()
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, gin.H{"url": url})
}
