package handler

import (
	"fmt"
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

// maxPriceConfigItems 价格列表长度上限（M1 安全加固，2026-10-17）：
// 防止无上限数组进入逐项 upsert 事务；超限 40000。HTTP 语义校验在 handler 层，
// 不下放 server 层（B1 分层）。
const maxPriceConfigItems = 500

// SetPrice 保存价格并返回更新的结账数据（契约 §3.6：新增校验 + 持久化）
func SetPrice(c *gin.Context) {
	var list []checkout.ProPriceRequest
	// 非数组（绑定失败）或空数组统一 40000（契约 §3.6 校验表）
	if err := c.ShouldBindJSON(&list); err != nil || len(list) == 0 {
		common.BadParam(c, "价格列表不能为空")
		return
	}
	// M1：数组长度上限校验（先于逐项校验，超限快速失败）
	if len(list) > maxPriceConfigItems {
		common.BadParam(c, fmt.Sprintf("价格列表数量超过上限 %d", maxPriceConfigItems))
		return
	}
	for i := range list {
		if list[i].ProductName == "" {
			common.BadParam(c, fmt.Sprintf("第 %d 项 productName 不能为空", i+1))
			return
		}
		if list[i].Spec == "" {
			common.BadParam(c, fmt.Sprintf("第 %d 项 spec 不能为空", i+1))
			return
		}
	}
	resp, err := application.SetPrice(list)
	if err != nil {
		// 契约 §3.6：事务失败/DB 错误统一 50000
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
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
