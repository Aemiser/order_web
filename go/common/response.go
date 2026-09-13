package common

import "github.com/gin-gonic/gin"

const (
	CodeOK       = 0     // 业务成功
	CodeBadParam = 40000 // 请求参数错误
	CodeError    = 50000 // 业务处理失败
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
