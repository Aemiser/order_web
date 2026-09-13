package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"order_server/go/application"
	"order_server/go/common"
	"order_server/go/models/enum"
	"order_server/go/server/order"
	ordertutil "order_server/go/utils/order"

	"github.com/gin-gonic/gin"
)

type recognitionRequest struct {
	Text        string              `json:"text"`
	AIModel     enum.AIModelType    `json:"aimodel"`
	PreviewType enum.RecPreviewType `json:"previewType"`
}

type createOrderRequest struct {
	Date time.Time `json:"date"`
}

// Recognition 同步订单识别
func Recognition(c *gin.Context) {
	var req recognitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	common.Ok(c, application.Recognition(req.Text, req.AIModel, req.PreviewType))
}

// RecognitionStream SSE 流式订单识别
func RecognitionStream(c *gin.Context) {
	var req recognitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	writeSSE := func(data string) error {
		for _, line := range strings.Split(data, "\n") {
			if _, err := c.Writer.WriteString("data: " + line + "\n"); err != nil {
				return err
			}
		}
		if _, err := c.Writer.WriteString("\n"); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	application.StreamRecognition(req.Text, req.AIModel, req.PreviewType,
		func(data string) error { return writeSSE(data) },
		func() { _ = writeSSE("DONE") },
	)
}

// GetPreview 预览列表
func GetPreview(c *gin.Context) {
	common.Ok(c, application.GetPreview())
}

// SetPreview 修改一条预览数据
func SetPreview(c *gin.Context) {
	var address ordertutil.ExceladdressInfo
	if err := c.ShouldBindJSON(&address); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if err := application.SetPreview(address); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// DeletePreview 删除一条预览数据
func DeletePreview(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.BadParam(c, "id 参数错误")
		return
	}
	if err := application.DeletePreview(uint(id)); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// CreateOrder 预览数据导入数据库
func CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if req.Date.IsZero() {
		common.BadParam(c, "date 不能为空")
		return
	}
	if err := application.CreateOrder(req.Date); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// ExportOrder 导出预览数据 Excel，返回下载地址
func ExportOrder(c *gin.Context) {
	url, err := application.ExportOrder()
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, gin.H{"url": url})
}

// OrderQuery 分页查询历史订单
func OrderQuery(c *gin.Context) {
	var req order.OrderQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	common.Ok(c, application.OrderQuery(req))
}
