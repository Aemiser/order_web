package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order_server/go/common"

	"github.com/gin-gonic/gin"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/order/create", CreateOrder)
	r.POST("/api/order/recognition/stream", RecognitionStream)
	return r
}

func TestCreateOrderRejectsMissingDate(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/create", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestCreateOrderRejectsMalformedJSON(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/order/create", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}
