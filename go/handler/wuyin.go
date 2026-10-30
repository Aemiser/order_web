package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"order_server/go/application"
	"order_server/go/common"
	"order_server/go/models/enum"
	wuyinutil "order_server/go/utils/wuyin"

	"github.com/gin-gonic/gin"
)

type wuyinRecognitionRequest struct {
	Text        string              `json:"text"`
	AIModel     enum.AIModelType    `json:"aimodel"`
	PreviewType enum.RecPreviewType `json:"previewType"`
}

type wuyinCreateOrderRequest struct {
	Date time.Time `json:"date"`
}

// checkWuyinDB 吴饮端点统一前置：未配置独立库时 50000
func checkWuyinDB(c *gin.Context) bool {
	if err := application.WuyinCheckDB(); err != nil {
		common.Fail(c, err)
		return false
	}
	return true
}

// WuyinRecognition 吴饮同步订单识别
func WuyinRecognition(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req wuyinRecognitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if !checkRecognitionText(c, req.Text) {
		return
	}
	common.Ok(c, application.WuyinRecognition(req.Text, req.AIModel, req.PreviewType))
}

// WuyinRecognitionStream 吴饮 SSE 流式订单识别（SSE 写入逻辑与一点点端点同构）
func WuyinRecognitionStream(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req wuyinRecognitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
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

	application.WuyinStreamRecognition(req.Text, req.AIModel, req.PreviewType,
		func(data string) error { return writeSSE(data) },
		func() { _ = writeSSE("DONE") },
	)
}

// GetWuyinPreview 吴饮预览列表
func GetWuyinPreview(c *gin.Context) {
	common.Ok(c, application.GetWuyinPreview())
}

// SetWuyinPreview 修改一条吴饮预览数据
func SetWuyinPreview(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var address wuyinutil.WuyinExceladdressInfo
	if err := c.ShouldBindJSON(&address); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if err := application.SetWuyinPreview(address); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// DeleteWuyinPreview 删除一条吴饮预览数据
func DeleteWuyinPreview(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.BadParam(c, "id 参数错误")
		return
	}
	if err := application.DeleteWuyinPreview(uint(id)); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// WuyinImportExcel 表格导入（multipart：file=表格文件，tag=统一标签，可选）
func WuyinImportExcel(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.BadParam(c, "缺少上传文件（表单字段 file）")
		return
	}
	reader, err := fileHeader.Open()
	if err != nil {
		common.Fail(c, err)
		return
	}
	defer reader.Close()

	list, err := application.ImportWuyinExcel(reader, fileHeader.Filename, c.PostForm("tag"))
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, list)
}

// CreateWuyinOrder 吴饮预览数据导入独立库
func CreateWuyinOrder(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req wuyinCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if req.Date.IsZero() {
		common.BadParam(c, "date 不能为空")
		return
	}
	if err := application.CreateWuyinOrder(req.Date); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// ExportWuyinOrder 吴饮预览导出 Excel
func ExportWuyinOrder(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	url, err := application.ExportWuyinOrder()
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, gin.H{"url": url})
}

// WuyinOrderQuery 吴饮订单分页查询
func WuyinOrderQuery(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req application.WuyinOrderQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	resp, err := application.WuyinOrderQuery(req)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// WuyinTagStats 标签订单数统计（startTime/endTime 可选且成对，同 ProvinceStats 约定）
func WuyinTagStats(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	startStr, endStr := c.Query("startTime"), c.Query("endTime")
	var start, end time.Time
	switch {
	case startStr == "" && endStr == "":
		// 都不传：全量统计
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
	resp, err := application.WuyinTagStats(start, end)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// ListWuyinTagConfigs 标签列表
func ListWuyinTagConfigs(c *gin.Context) {
	common.Ok(c, application.ListWuyinTagConfigs())
}

// CreateWuyinTagConfig 新增标签
func CreateWuyinTagConfig(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req application.WuyinTagConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if err := application.CreateWuyinTagConfig(req); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// UpdateWuyinTagConfig 修改标签
func UpdateWuyinTagConfig(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	var req application.WuyinTagConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if err := application.UpdateWuyinTagConfig(uint(id), req); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// DeleteWuyinTagConfig 删除标签
func DeleteWuyinTagConfig(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	if err := application.DeleteWuyinTagConfig(uint(id)); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// UpdateWuyinOrder 修改吴饮历史订单
func UpdateWuyinOrder(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	var req application.UpdateWuyinOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if err := application.UpdateWuyinOrder(uint(id), req); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

// DeleteWuyinOrder 删除吴饮历史订单
func DeleteWuyinOrder(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.BadParam(c, "id 参数错误")
		return
	}
	if err := application.DeleteWuyinOrder(uint(id)); err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, nil)
}

type wuyinDayDataRequest struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Tags      []string  `json:"tags"`
}

type wuyinExportRequest struct {
	Tags []string `json:"tags"`
}

// GetWuyinDayData 按标签分组的结账数据
func GetWuyinDayData(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req wuyinDayDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		common.BadParam(c, "startTime 与 endTime 不能为空")
		return
	}
	resp, err := application.GetWuyinDayData(req.StartTime, req.EndTime, req.Tags)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// SetWuyinPrice 保存价格并重算
func SetWuyinPrice(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req []application.WuyinPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadParam(c, "请求参数错误: "+err.Error())
		return
	}
	resp, err := application.SetWuyinPrice(req)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, resp)
}

// ExportWuyinCheckout 结账导出（tags 可选，空=全部标签）
func ExportWuyinCheckout(c *gin.Context) {
	if !checkWuyinDB(c) {
		return
	}
	var req wuyinExportRequest
	_ = c.ShouldBindJSON(&req) // body 可为空对象
	url, err := application.ExportWuyinCheckout(req.Tags)
	if err != nil {
		common.Fail(c, err)
		return
	}
	common.Ok(c, gin.H{"url": url})
}
