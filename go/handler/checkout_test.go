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

func TestGetDayDataRejectsZeroTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/checkout/daydata", GetDayData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/daydata", strings.NewReader(`{"startTime":"0001-01-01T00:00:00Z","endTime":"0001-01-01T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeBadParam {
		t.Fatalf("code = %d, want %d", resp.Code, common.CodeBadParam)
	}
}

func TestSetProductBelongBindsMap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/checkout/product-belong", SetProductBelong)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/checkout/product-belong", strings.NewReader(`{"香水柠檬":["5斤柠檬"]}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	var resp common.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != common.CodeOK {
		t.Fatalf("code = %d, want %d, msg = %s", resp.Code, common.CodeOK, resp.Msg)
	}
}
