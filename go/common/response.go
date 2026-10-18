package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK       = 0     // 业务成功
	CodeBadParam = 40000 // 请求参数错误
	CodeError    = 50000 // 业务处理失败

	CodeUnauthorized    = 40100 // 未登录或登录已失效
	CodeForbidden       = 40300 // 已登录但被拒绝（如账号被禁用）
	CodeTooManyRequests = 42900 // 失败次数过多触发限流
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Ok 业务成功，HTTP 状态码固定 200
func Ok(c *gin.Context, data interface{}) {
	c.JSON(200, Response{Code: CodeOK, Msg: "ok", Data: data})
}

// Err 业务失败，HTTP 状态码固定 200，由 code 区分
func Err(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{Code: code, Msg: msg, Data: nil})
}

func BadParam(c *gin.Context, msg string) {
	Err(c, CodeBadParam, msg)
}

func Fail(c *gin.Context, err error) {
	Err(c, CodeError, err.Error())
}

// Unauthorized 鉴权失败，返回真实 HTTP 401 而非统一响应体的 200+code。
// 「未认证」是传输层拒绝而非业务结果（与 Healthz 打破响应体先例同理），
// 前端拦截器与中间代理都按状态码识别它。带 Abort：中间件里调用时
// 直接终止后续 handler。
func Unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
		Code: CodeUnauthorized,
		Msg:  msg,
	})
}
