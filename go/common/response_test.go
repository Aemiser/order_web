package common

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestOkWritesUnifiedResponse(t *testing.T) {
	c, w := newTestContext()

	Ok(c, map[string]int{"Deepseek": 1})

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != CodeOK {
		t.Fatalf("code = %d, want %d", resp.Code, CodeOK)
	}
	if resp.Msg != "ok" {
		t.Fatalf("msg = %q, want ok", resp.Msg)
	}
}

func TestBadParamKeepsHTTP200(t *testing.T) {
	c, w := newTestContext()

	BadParam(c, "参数错误")

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != CodeBadParam || resp.Msg != "参数错误" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestFailCarriesErrorMessage(t *testing.T) {
	c, w := newTestContext()

	Fail(c, errTest{})

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != CodeError || resp.Msg != "boom" {
		t.Fatalf("resp = %+v", resp)
	}
}

type errTest struct{}

func (errTest) Error() string { return "boom" }
