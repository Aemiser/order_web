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

// maxRecognitionTextBytes 识别请求 text 长度上限（M2 安全加固，2026-10-17，200KB）：
// 识别文本是审核对比基准 global.LastInputText 的唯一入口（AuditRequest 按契约 §8.1
// 为空对象、不回传文本），超限文本会原样进入正则解析与 AI 复核全量外发（audit.go
// aiReviewItems），故在入口直接 40000 拒绝。
const maxRecognitionTextBytes = 200 << 10 // 200KB

// checkRecognitionText M2：识别文本长度校验，超限返回 40000（同步/流式两个端点共用）。
func checkRecognitionText(c *gin.Context, text string) bool {
	if len(text) > maxRecognitionTextBytes {
		common.BadParam(c, "text 长度超过上限 200KB")
		return false
	}
	return true
}

// Recognition 同步订单识别
func Recognition(c *gin.Context) {
	var req recognitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if !checkRecognitionText(c, req.Text) {
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
	// M2：text 超限在进入 SSE 之前返回统一 JSON 错误（契约 §5 第 2 条）
	if !checkRecognitionText(c, req.Text) {
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
	resp, err := application.OrderQuery(req)
	if err != nil {
		// 契约 §3.5 变更4：查询失败返回 50000（原吞错静默返回空列表）
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// ProvinceStats 省份统计（GET /api/order/province-stats，契约 §3.1）。
// startTime/endTime 可选 RFC3339 且必须成对传入：都不传全量统计；
// 只传其一 40000；格式非 RFC3339 40000。
func ProvinceStats(c *gin.Context) {
	startStr, endStr := c.Query("startTime"), c.Query("endTime")

	var start, end time.Time
	switch {
	case startStr == "" && endStr == "":
		// 都不传：全量统计（零值时间透传）
	case startStr == "" || endStr == "":
		common.BadParam(c, "startTime 与 endTime 必须成对传入")
		return
	default:
		var err error
		if start, err = time.Parse(time.RFC3339, startStr); err != nil {
			common.BadParam(c, "startTime 时间格式错误")
			return
		}
		if end, err = time.Parse(time.RFC3339, endStr); err != nil {
			common.BadParam(c, "endTime 时间格式错误")
			return
		}
	}

	resp, err := application.ProvinceStats(start, end)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// UpdateOrder 修改历史订单（PUT /api/order/:id，契约 §3.3）。
// 三段式：id 非十进制正整数 → 40000 id 参数错误；body 绑定失败 → 40000；
// 六字段全未传 → 40000（契约 §3.3 错误码表，handler 层判定）；
// server 层错误（订单不存在/规格解析失败/更新失败）→ Fail 50000。
func UpdateOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	var req application.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if req.Recipient == nil && req.Phone == nil && req.Address == nil &&
		req.Province == nil && req.ProductSpecs == nil && req.Remarks == nil {
		common.BadParam(c, "至少提供一项需要修改的字段")
		return
	}
	// M3 留痕：操作来源 IP 经 application → server 透传，写入 LogModels 留痕
	if err := application.UpdateOrder(uint(id), req, c.ClientIP()); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// DeleteOrder 软删除历史订单（DELETE /api/order/:id，契约 §3.4）。
// id 非十进制正整数 → 40000；server 层错误（重复删除返回订单不存在/删除失败）→ Fail 50000。
// 非幂等：前端删除成功后应移除该行，避免重放（契约 §2 幂等性总览）。
func DeleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	// M3 留痕：操作来源 IP 经 application → server 透传，写入 LogModels 留痕
	if err := application.DeleteOrder(uint(id), c.ClientIP()); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// AuditPreview 预览审核（POST /api/order/preview/audit，契约 §3.2）。
// body 缺失/非 JSON → 40000；server 层错误（预览态为空/原始文本为空/内部错误）→ Fail 50000。
// AI 复核失败不阻塞响应：aiFailed=true + 程序对比结果仍返回（契约 §3.2 约定2）。
func AuditPreview(c *gin.Context) {
	var req application.AuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	resp, err := application.AuditPreview(req)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}
