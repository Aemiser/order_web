# Wails 桌面应用 → 前后端分离 Web 应用重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把基于 Wails v2 的桌面应用 `order_web` 重构为 Gin 后端 + Vue 3 前端完全分离的 Web 应用，业务行为不变。

**Architecture:** 后端引入 `router → handler → application → server` 四层单向依赖；`server` 层（原 `go/server/*`）去掉全部 Wails 耦合，保留纯领域逻辑；`handler` 负责 HTTP 出入参与 SSE，`application` 负责流程编排与全局状态、导出文件 URL；前端删除 `wailsjs` 绑定，改为 `src/api/` 下的 fetch 封装。

**Tech Stack:** Go 1.24 + Gin + GORM + excelize；Vue 3 + TypeScript + Vite 6 + Element Plus。

**参考规格：** `docs/superpowers/specs/2026-09-13-wails-to-web-refactor-design.md`

---

## 文件结构

### 后端新增

| 文件 | 职责 |
|---|---|
| `go/common/response.go` | 统一响应 `{code,msg,data}` 与 `Ok/Err/BadParam/Fail` 帮助函数 |
| `go/application/enter.go` | 导出目录保障 `ensureExportDir`、下载地址转换 `uploadURL` |
| `go/application/order.go` | 订单流程编排（识别/预览/入库/导出/查询） |
| `go/application/checkout.go` | 结账流程编排（日数据/商品归属/价格/导出） |
| `go/handler/system.go` | `GET /api/system/ai-models` |
| `go/handler/order.go` | 订单 8 个端点（含 SSE 流式识别） |
| `go/handler/checkout.go` | 结账 4 个端点 |
| `go/router/enter.go` | `Setup() *gin.Engine`：中间件 + 路由注册 + `/uploads` 静态 |
| `go/router/middleware.go` | CORS 中间件（白名单） |

### 后端修改

| 文件 | 改动 |
|---|---|
| `main.go` | 去掉 `embed`/`wails.Run`，改为初始化 + `router.Setup().Run()` |
| `go/conf/enter.go` | 新增 `Server` 段；删除 `VersionConfig` |
| `go/server/order/enter.go` | 删除 `EmitStreamData`/`EmitStreamDone`/`streamCtx`；`StreamOrderRec` 去掉 ctx 首参 |
| `go/server/order/save.go` | `SaveToExcel() (string, error)`，落到 `savePath.excel` 目录 |
| `go/server/checkout/enter.go` | 删除 `selectSaveDirectory`；`Checkout() (string, error)` |
| `settings.yaml` | 新增 `server` 段；删除 `version` 段 |

### 后端删除

`app.go`、`go/server/version/`、`go/server/config/`、`go/testdata/`

### 前端新增

| 文件 | 职责 |
|---|---|
| `src/api/request.ts` | fetch 封装：baseURL、统一解包、错误抛出 |
| `src/api/order.ts` | 订单 API |
| `src/api/checkout.ts` | 结账 API |
| `src/api/system.ts` | 系统 API |
| `src/api/stream.ts` | SSE 流式识别封装（本次不接入界面） |
| `src/vite-env.d.ts` | `vite/client` 类型引用（`import.meta.env` 需要） |

### 前端修改

`src/App.vue`、`src/main.ts`、`src/types.ts`、`src/components/Main/OrderTable.vue`、`src/components/Main/OrderInput.vue`、`src/components/Main/OrderDetailsView.vue`、`src/components/Main/Checkout/CheckoutView.vue`、`src/components/Main/Checkout/CheckQuery.vue`、`vite.config.ts`、`tsconfig.json`

### 前端删除

`frontend/wailsjs/`、`frontend/src/stores/`、`frontend/src/components/demo.vue`、`frontend/src/components/Header/index.js`、`frontend/vite.config.js`、`frontend/src/main.js`、`frontend/.env.example`、`frontend/dist/`

### 仓库根删除

`wails.json`、`build/`、根 `index.html`、根 `package.json`、根 `package-lock.json`、`.trae/`、`tmp/`

---

## Phase A：后端

### Task 1: 引入 Gin 依赖

**Files:**
- Modify: `go.mod`、`go.sum`

- [ ] **Step 1: 拉取 Gin**

在仓库根 `F:\project\go\order_web` 执行：

```bash
go get github.com/gin-gonic/gin@latest
```

- [ ] **Step 2: 验证依赖已写入**

Run: `grep -n "gin-gonic" go.mod`
Expected: 出现 `github.com/gin-gonic/gin v1.x.x` 一行

---

### Task 2: 统一响应封装

**Files:**
- Create: `go/common/response.go`
- Test: `go/common/response_test.go`

- [ ] **Step 1: 写失败测试**

创建 `go/common/response_test.go`：

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./go/common/ -run "TestOk|TestBadParam|TestFail" -v`
Expected: FAIL —`undefined: Ok`、`undefined: Response` 等编译错误

- [ ] **Step 3: 实现 response.go**

创建 `go/common/response.go`：

```go
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./go/common/ -run "TestOk|TestBadParam|TestFail" -v`
Expected: PASS（3 个测试）

- [ ] **Step 5: 提交**

```bash
git add go/common/response.go go/common/response_test.go go.mod go.sum
git commit -m "feat(common): 新增统一响应封装 {code,msg,data}"
```

---

### Task 3: 配置新增 Server 段

**Files:**
- Modify: `go/conf/enter.go`、`settings.yaml`

- [ ] **Step 1: 扩展 Config**

把 `go/conf/enter.go` 全文替换为：

```go
package conf

type Config struct {
	Log      Log           `yaml:"log"`
	DB       []DB          `yaml:"db"`
	Email    Email         `yaml:"email"`
	AiModel  AiModel       `yaml:"aimodel"`
	SavePath SavePath      `yaml:"savePath"`
	Version  VersionConfig `yaml:"version"`
	Server   Server        `yaml:"server"`
}

type VersionConfig struct {
	Repo string `yaml:"repo"`
}

type Server struct {
	Port         int      `yaml:"port"`
	AllowOrigins []string `yaml:"allowOrigins"`
}
```

> 注：`VersionConfig` 在 Task 9 删除（本任务先保留，保证期间可编译）。

- [ ] **Step 2: settings.yaml 新增 server 段**

在 `settings.yaml` 末尾追加：

```yaml
server:
    port: 8080
    allowOrigins:
        - "http://localhost:3000"
```

- [ ] **Step 3: 验证可编译**

Run: `go build ./go/conf/`
Expected: 无输出（成功）

---

### Task 4: server/order 去 Wails 化

**Files:**
- Modify: `go/server/order/enter.go`、`go/server/order/save.go`、`app.go`（临时兼容）

- [ ] **Step 1: 重写 enter.go**

把 `go/server/order/enter.go` 全文替换为：

```go
package order

import (
	"order_server/go/models/enum"
	"order_server/go/server/order/ai_model"
	ordertutil "order_server/go/utils/order"
)

func OrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	switch model {
	case enum.AIModelType_DeepSeek:
		return ai_model.OrderRecDeepseek(inputText, previewType)
	case enum.AIModelType_Qwen:
		return ai_model.OrderRecQwen(inputText, previewType)
	}
	return nil
}

type StreamCallback func(data string) error
type DoneCallback func()

func StreamOrderRec(inputText string, model enum.AIModelType, previewType enum.RecPreviewType, callback StreamCallback, doneCallback DoneCallback) {
	switch model {
	case enum.AIModelType_DeepSeek:
		ai_model.StreamRecDeepseek(inputText, previewType, callback, doneCallback)
	case enum.AIModelType_Qwen:
		ai_model.StreamRecQwen(inputText, previewType, callback, doneCallback)
	}
}
```

变更点：删除 `context` / `runtime` import、`streamCtx` / `streamEventName` 变量、`EmitStreamData` / `EmitStreamDone` 函数；`StreamOrderRec` 去掉 `ctx context.Context` 首参。

- [ ] **Step 2: save.go 改为写导出目录并返回路径**

把 `go/server/order/save.go` 中的 `SaveToExcel` 函数（第 15-29 行）替换为：

```go
func SaveToExcel() (string, error) {
	addressList := global.AddressList
	now := time.Now()
	filename := fmt.Sprintf("%s.xlsx", now.Format("2006-01-02"))
	// 保存为Excel文件
	excelFile, err := save.SaveToExcel(addressList, filepath.Join(global.Config.SavePath.Excel, filename))
	if err != nil {
		logrus.Errorf("保存Excel文件失败: %v", err)
		alog.Logaction(enum.LogErrLevel, "保存Excel文件失败", err.Error())
		return "", err
	}
	logrus.Infof("Excel文件已保存: %s\n", excelFile)
	logtent := "保存路径：\n" + excelFile
	alog.Logaction(enum.LogInfoLevel, "保存Excel文件成功", logtent)
	return excelFile, nil
}
```

并在该文件 import 块中加入 `"path/filepath"`。

- [ ] **Step 3: 临时兼容 app.go**

`app.go:52` 的调用去掉首个 ctx 参数：

```go
	order.StreamOrderRec(inputText, aimodel, previewType, func(data string) error {
```

（其余三行不变；`order.SaveToExcel()` 忽略返回值在 Go 中合法，无需改动。app.go 将在 Task 9 整体删除。）

- [ ] **Step 4: 验证可编译**

Run: `go build ./go/server/order/ ./...`
Expected: 无输出（成功）

- [ ] **Step 5: 提交**

```bash
git add go/server/order/enter.go go/server/order/save.go app.go go/conf/enter.go settings.yaml
git commit -m "refactor(order): 移除 Wails 耦合，导出改写入 savePath.excel"
```

---

### Task 5: server/checkout 去 Wails 化

**Files:**
- Modify: `go/server/checkout/enter.go`、`app.go`（临时兼容）

- [ ] **Step 1: 调整 import 块**

`go/server/checkout/enter.go` 顶部 import 改为（删除 `context`、`os`、`github.com/wailsapp/wails/v2/pkg/runtime`，新增 `path/filepath`）：

```go
import (
	"fmt"
	"order_server/go/core"
	"order_server/go/global"
	"order_server/go/models"
	ordertutil "order_server/go/utils/order"
	time2 "order_server/go/utils/time"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)
```

- [ ] **Step 2: 改写 Checkout 函数**

把 `go/server/checkout/enter.go` 第 145-203 行的 `Checkout` 函数替换为：

```go
// Checkout 生成财务账单 Excel，返回生成文件的绝对路径
func Checkout() (string, error) {
	fmt.Println("========== 商品销售统计Excel生成器 ==========")

	// 创建规格管理器
	sm := NewSpecManager()

	// 添加数据
	AddcheckoutData(sm)
	fmt.Println(sm.Orders)
	// 打印统计摘要
	sm.PrintSummary()

	// 生成Excel文件
	fmt.Println("\n========== 生成Excel文件 ==========")
	fmt.Println("begtime:", begtime2)
	fmt.Println("endtime:", endtime2)

	// 生成带时间戳的文件名，输出到配置的导出目录
	filename := fmt.Sprintf("%s_%s结账.xlsx", begtime2, endtime2)
	filePath := filepath.Join(global.Config.SavePath.Excel, filename)

	if err := sm.GenerateExcel(filePath); err != nil {
		fmt.Printf("❌ 生成Excel失败: %v\n", err)
		return "", err
	}
	fmt.Printf("✅ Excel生成成功！文件: %s\n", filePath)

	// 显示一些统计信息
	fmt.Println("\n========== 统计信息 ==========")
	fmt.Printf("总订单数: %d\n", len(sm.Orders))
	fmt.Printf("总商品种类: %d\n", len(sm.ProductMap))
	fmt.Printf("总销售天数: %d\n", len(sm.DateMap))

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return filePath, nil
	}
	return absPath, nil
}
```

- [ ] **Step 3: 删除 selectSaveDirectory**

删除 `go/server/checkout/enter.go` 第 232-278 行整个 `selectSaveDirectory` 函数（含其上方注释 `// 选择保存目录`）。

- [ ] **Step 4: 临时兼容 app.go**

`app.go:151` 改为：

```go
	checkout.Checkout()
```

（`SaveCheackToExcel` 方法体只保留这一行；返回值忽略合法。app.go 将在 Task 9 整体删除。）

- [ ] **Step 5: 验证可编译**

Run: `go build ./...`
Expected: 无输出（成功）

- [ ] **Step 6: 提交**

```bash
git add go/server/checkout/enter.go app.go
git commit -m "refactor(checkout): 移除目录选择弹窗，输出到 savePath.excel"
```

---

### Task 6: application 应用服务层

**Files:**
- Create: `go/application/enter.go`、`go/application/order.go`、`go/application/checkout.go`
- Test: `go/application/enter_test.go`

- [ ] **Step 1: 写失败测试**

创建 `go/application/enter_test.go`：

```go
package application

import (
	"net/url"
	"order_server/go/conf"
	"order_server/go/global"
	"os"
	"path/filepath"
	"testing"
)

func setupConfig(t *testing.T, exportDir string) {
	t.Helper()
	global.Config = &conf.Config{}
	global.Config.SavePath.Excel = exportDir
}

func TestUploadURLEscapesChineseFilename(t *testing.T) {
	name := "2026年9月13日_2026年9月20日结账.xlsx"
	got := uploadURL(filepath.Join(`C:\data\uploads`, name))
	want := "/uploads/" + url.PathEscape(name)
	if got != want {
		t.Fatalf("uploadURL = %q, want %q", got, want)
	}
}

func TestEnsureExportDirCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "uploads")
	setupConfig(t, dir)

	if err := ensureExportDir(); err != nil {
		t.Fatalf("ensureExportDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("目录未创建: %v", err)
	}
}

func TestEnsureExportDirRejectsEmptyConfig(t *testing.T) {
	setupConfig(t, "")

	if err := ensureExportDir(); err == nil {
		t.Fatal("空目录配置应返回错误")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./go/application/ -v`
Expected: FAIL —`undefined: uploadURL`、`undefined: ensureExportDir`

- [ ] **Step 3: 实现 enter.go**

创建 `go/application/enter.go`：

```go
package application

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"order_server/go/global"
)

// ensureExportDir 确保导出目录存在
func ensureExportDir() error {
	dir := global.Config.SavePath.Excel
	if dir == "" {
		return fmt.Errorf("未配置导出目录 savePath.excel")
	}
	return os.MkdirAll(dir, 0755)
}

// uploadURL 把本地导出文件路径转换为 /uploads/ 下的下载地址
func uploadURL(path string) string {
	return "/uploads/" + url.PathEscape(filepath.Base(path))
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./go/application/ -v`
Expected: PASS（3 个测试）

- [ ] **Step 5: 实现 order.go**

创建 `go/application/order.go`：

```go
package application

import (
	"order_server/go/global"
	"order_server/go/models/enum"
	"order_server/go/server/order"
	ordertutil "order_server/go/utils/order"
	"time"
)

// OrderQueryRequest 订单详情查询入参（与 server 层同类型）
type OrderQueryRequest = order.OrderQueryRequest

// OrderQueryResponse 订单详情查询出参
type OrderQueryResponse = order.OrderQueryResponse

// ListAIModels 返回已配置的大模型
func ListAIModels() map[string]int {
	return global.AiMap
}

// Recognition 同步订单识别
func Recognition(text string, model enum.AIModelType, previewType enum.RecPreviewType) []ordertutil.ExceladdressInfo {
	return order.OrderRec(text, model, previewType)
}

// StreamRecognition 流式订单识别，回调由 handler 注入（SSE 写入）
func StreamRecognition(text string, model enum.AIModelType, previewType enum.RecPreviewType, callback order.StreamCallback, done order.DoneCallback) {
	order.StreamOrderRec(text, model, previewType, callback, done)
}

// GetPreview 获取预览列表
func GetPreview() []ordertutil.ExceladdressInfo {
	return order.GetpreviewData()
}

// SetPreview 修改一条预览数据
func SetPreview(address ordertutil.ExceladdressInfo) error {
	return order.SetpreviewData(address)
}

// DeletePreview 删除一条预览数据
func DeletePreview(id uint) error {
	return order.DeletepreviewData(id)
}

// CreateOrder 把预览数据导入数据库
func CreateOrder(date time.Time) error {
	return order.CreateOrder(date)
}

// ExportOrder 导出预览数据为 Excel，返回下载地址
func ExportOrder() (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := order.SaveToExcel()
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}

// OrderQuery 分页查询历史订单
func OrderQuery(req order.OrderQueryRequest) order.OrderQueryResponse {
	return order.OrderQuery(req)
}
```

- [ ] **Step 6: 实现 checkout.go**

创建 `go/application/checkout.go`：

```go
package application

import (
	"order_server/go/server/checkout"
	"time"
)

// GetDayData 按日期范围查询结账数据
func GetDayData(start, end time.Time) checkout.CheckoutResponse {
	return checkout.GetDaydata(start, end)
}

// SetProductBelong 保存商品归属配置
func SetProductBelong(mapp map[string][]string) {
	checkout.SetFieldList(mapp)
}

// SetPrice 保存价格并返回更新的结账数据
func SetPrice(list []checkout.ProPriceRequest) checkout.CheckoutResponse {
	return checkout.SetPrice(list)
}

// ExportCheckout 生成财务账单 Excel，返回下载地址
func ExportCheckout() (string, error) {
	if err := ensureExportDir(); err != nil {
		return "", err
	}
	path, err := checkout.Checkout()
	if err != nil {
		return "", err
	}
	return uploadURL(path), nil
}
```

- [ ] **Step 7: 验证编译与测试**

Run: `go test ./go/application/ -v && go build ./...`
Expected: PASS，且 build 无输出

- [ ] **Step 8: 提交**

```bash
git add go/application/
git commit -m "feat(application): 新增应用服务层，编排订单与结账流程"
```

---

### Task 7: handler HTTP 层

**Files:**
- Create: `go/handler/system.go`、`go/handler/order.go`、`go/handler/checkout.go`
- Test: `go/handler/order_test.go`、`go/handler/checkout_test.go`

- [ ] **Step 1: 写失败测试**

创建 `go/handler/order_test.go`：

```go
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
```

创建 `go/handler/checkout_test.go`：

```go
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
```

> 注：`TestSetProductBelongBindsMap` 会真实写入 `productbelong.json`（`SetFieldList` 内部调用 `core.WriteProductBelongMap`）。测试结束请用 `git checkout -- productbelong.json` 还原。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./go/handler/ -v`
Expected: FAIL —`undefined: CreateOrder`、`undefined: GetDayData` 等

- [ ] **Step 3: 实现 system.go**

创建 `go/handler/system.go`：

```go
package handler

import (
	"order_server/go/application"
	"order_server/go/common"

	"github.com/gin-gonic/gin"
)

// GetAIModels 已配置的大模型列表
func GetAIModels(c *gin.Context) {
	common.Ok(c, application.ListAIModels())
}
```

- [ ] **Step 4: 实现 order.go**

创建 `go/handler/order.go`：

```go
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
```

- [ ] **Step 5: 实现 checkout.go**

创建 `go/handler/checkout.go`：

```go
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
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./go/handler/ -v`
Expected: PASS（4 个测试）

- [ ] **Step 7: 还原测试副作用**

Run: `git checkout -- productbelong.json`
Expected: 无输出

- [ ] **Step 8: 提交**

```bash
git add go/handler/
git commit -m "feat(handler): 新增 HTTP 层，含 SSE 流式识别端点"
```

---

### Task 8: router 路由层

**Files:**
- Create: `go/router/enter.go`、`go/router/middleware.go`
- Test: `go/router/enter_test.go`

- [ ] **Step 1: 写失败测试**

创建 `go/router/enter_test.go`：

```go
package router

import (
	"net/http"
	"net/http/httptest"
	"order_server/go/conf"
	"order_server/go/global"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupGlobal(t *testing.T) {
	t.Helper()
	global.Config = &conf.Config{}
	global.Config.SavePath.Excel = t.TempDir()
	global.Config.Server.AllowOrigins = []string{"http://localhost:3000"}
}

func TestSetupRegistersAllEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)

	want := []string{
		"GET /api/system/ai-models",
		"POST /api/order/recognition",
		"POST /api/order/recognition/stream",
		"GET /api/order/preview",
		"PUT /api/order/preview",
		"DELETE /api/order/preview/:id",
		"POST /api/order/create",
		"POST /api/order/export",
		"POST /api/order/query",
		"POST /api/checkout/daydata",
		"PUT /api/checkout/product-belong",
		"POST /api/checkout/price",
		"POST /api/checkout/export",
	}

	got := make(map[string]bool)
	for _, r := range Setup().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("缺少路由 %s", w)
		}
	}
}

func TestCORSPreflightAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/checkout/daydata", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGlobal(t)
	r := Setup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/checkout/daydata", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("未授权来源不应返回 CORS 头，得到 %q", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./go/router/ -v`
Expected: FAIL —`undefined: Setup`

- [ ] **Step 3: 实现 middleware.go**

创建 `go/router/middleware.go`：

```go
package router

import (
	"net/http"
	"order_server/go/global"

	"github.com/gin-gonic/gin"
)

// CORS 仅对 allowOrigins 白名单内的来源放开跨域
func CORS() gin.HandlerFunc {
	allowOrigins := global.Config.Server.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && contains(allowOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 实现 enter.go**

创建 `go/router/enter.go`：

```go
package router

import (
	"order_server/go/global"
	"order_server/go/handler"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CORS())

	api := r.Group("/api")
	{
		api.GET("/system/ai-models", handler.GetAIModels)

		orderGroup := api.Group("/order")
		{
			orderGroup.POST("/recognition", handler.Recognition)
			orderGroup.POST("/recognition/stream", handler.RecognitionStream)
			orderGroup.GET("/preview", handler.GetPreview)
			orderGroup.PUT("/preview", handler.SetPreview)
			orderGroup.DELETE("/preview/:id", handler.DeletePreview)
			orderGroup.POST("/create", handler.CreateOrder)
			orderGroup.POST("/export", handler.ExportOrder)
			orderGroup.POST("/query", handler.OrderQuery)
		}

		checkoutGroup := api.Group("/checkout")
		{
			checkoutGroup.POST("/daydata", handler.GetDayData)
			checkoutGroup.PUT("/product-belong", handler.SetProductBelong)
			checkoutGroup.POST("/price", handler.SetPrice)
			checkoutGroup.POST("/export", handler.ExportCheckout)
		}
	}

	// 导出文件静态下载
	r.Static("/uploads", global.Config.SavePath.Excel)

	return r
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./go/router/ -v`
Expected: PASS（3 个测试）

- [ ] **Step 6: 提交**

```bash
git add go/router/
git commit -m "feat(router): 新增路由层与 CORS 中间件"
```

---

### Task 9: main.go 启动 Gin + 删除 Wails 与死代码

**Files:**
- Modify: `main.go`、`go/conf/enter.go`、`settings.yaml`
- Delete: `app.go`、`go/server/version/`、`go/server/config/`、`go/testdata/`

- [ ] **Step 1: 重写 main.go**

把 `main.go` 全文替换为：

```go
package main

import (
	"fmt"
	"order_server/go/core"
	"order_server/go/flags"
	"order_server/go/global"
	"order_server/go/router"

	"github.com/sirupsen/logrus"
)

func main() {
	flags.Parse()
	global.Config = core.ReadConf()
	core.InitLogrus()
	global.Db = core.InitDB()
	global.ProductBelongMap = core.ReadProductBelongMap()
	global.AiMap = core.ReadAiMap()
	global.SystemPrompt = core.Readsystemprompt()
	flags.Run()

	port := global.Config.Server.Port
	if port == 0 {
		port = 8080
	}
	addr := fmt.Sprintf(":%d", port)

	logrus.Infof("HTTP 服务启动: %s", addr)
	if err := router.Setup().Run(addr); err != nil {
		logrus.Fatalf("HTTP 服务启动失败: %v", err)
	}
}
```

- [ ] **Step 2: 删除 Wails 绑定层与死代码**

```bash
git rm app.go wails.json
git rm -r go/server/version go/server/config go/testdata
```

- [ ] **Step 3: 删除 VersionConfig**

把 `go/conf/enter.go` 全文替换为：

```go
package conf

type Config struct {
	Log      Log      `yaml:"log"`
	DB       []DB     `yaml:"db"`
	Email    Email    `yaml:"email"`
	AiModel  AiModel  `yaml:"aimodel"`
	SavePath SavePath `yaml:"savePath"`
	Server   Server   `yaml:"server"`
}

type Server struct {
	Port         int      `yaml:"port"`
	AllowOrigins []string `yaml:"allowOrigins"`
}
```

- [ ] **Step 4: settings.yaml 删除 version 段**

删除 `settings.yaml` 中这三行：

```yaml
version:
    repo: "Aemiser/order_manager"
```

- [ ] **Step 5: 整理依赖**

```bash
go mod tidy
```

- [ ] **Step 6: 验证无 Wails 残留且全仓可编译**

Run: `go build ./... && go vet ./...`
Expected: 两条命令均无输出

Run: `grep -rn "wails" --include="*.go" .`
Expected: 无匹配（若 `go.sum` 中残留无关紧要）

Run: `go test ./...`
Expected: PASS（common / application / handler / router 四个包）

- [ ] **Step 7: 提交**

```bash
git add -A
git commit -m "refactor: main 改为启动 Gin，删除 Wails 绑定层与版本更新模块"
```

---

### Task 10: 后端启动自检

**Files:** 无（人工验证）

- [ ] **Step 1: 启动服务**

Run: `go run main.go`
Expected: 日志出现 `读取配置文件 settings.yaml 成功` 与 `HTTP 服务启动: :8080`，进程保持运行

> 若数据库不可达会 panic，属既有依赖，非本次改动引入。

- [ ] **Step 2: 验证只读端点**

Run: `curl -s http://localhost:8080/api/system/ai-models`
Expected: `{"code":0,"msg":"ok","data":{"Deepseek":1,"Qwen":2}}`

- [ ] **Step 3: 验证预览端点与 CORS 头**

Run: `curl -s -i -H "Origin: http://localhost:3000" http://localhost:8080/api/order/preview | head -20`
Expected: HTTP 200，含 `Access-Control-Allow-Origin: http://localhost:3000`

- [ ] **Step 4: 验证静态目录**

Run: `mkdir -p uploads && echo hi > uploads/probe.txt && curl -s http://localhost:8080/uploads/probe.txt && rm uploads/probe.txt`
Expected: 输出 `hi`

- [ ] **Step 5: 停止服务**

在 `go run` 所在终端按 `Ctrl+C`。

---

## Phase B：前端

### Task 11: 前端 API 层

**Files:**
- Create: `frontend/src/api/request.ts`、`order.ts`、`checkout.ts`、`system.ts`、`stream.ts`、`frontend/src/vite-env.d.ts`

- [ ] **Step 1: request.ts**

创建 `frontend/src/api/request.ts`：

```ts
export const BASE = import.meta.env.VITE_API_BASE ?? '';

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }

  const payload = (await res.json()) as ApiResponse<T>;
  if (payload.code !== 0) {
    throw new Error(payload.msg || '请求失败');
  }
  return payload.data;
}

export const get = <T>(path: string) => request<T>('GET', path);
export const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body);
export const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body);
export const del = <T>(path: string) => request<T>('DELETE', path);
```

- [ ] **Step 2: order.ts**

创建 `frontend/src/api/order.ts`：

```ts
import { del, get, post, put } from './request';

export interface ExcelAddressInfo {
  createdAt?: string;
  VID?: number;
  PreviewID?: number;
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  商品规格?: string;
  备注?: string;
}

export interface OrderQueryResponse {
  list: ExcelAddressInfo[];
  count: number;
}

export interface RecognitionResult {
  createdAt?: string;
  PreviewID?: number;
  VID?: number;
  Recipient?: string;
  Phone?: string;
  Address?: string;
  ProductSpecs?: string;
  Remarks?: string;
  [key: string]: unknown;
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function recognition(text: string, aimodel: number, previewType: number) {
  return post<RecognitionResult[]>('/api/order/recognition', { text, aimodel, previewType });
}

export function getPreview() {
  return get<ExcelAddressInfo[]>('/api/order/preview');
}

export function setPreview(address: ExcelAddressInfo) {
  return put<null>('/api/order/preview', address);
}

export function deletePreview(id: number) {
  return del<null>(`/api/order/preview/${id}`);
}

export function createOrder(date: Date | string) {
  return post<null>('/api/order/create', { date: toISO(date) });
}

export function exportOrder() {
  return post<{ url: string }>('/api/order/export');
}

export function queryOrders(params: Record<string, unknown>) {
  return post<OrderQueryResponse>('/api/order/query', params);
}
```

- [ ] **Step 3: checkout.ts**

创建 `frontend/src/api/checkout.ts`：

```ts
import { post, put } from './request';

export interface Spec {
  name: string;
  pricePerKg: number;
  pricePerItem: number;
  total: number;
}

export interface ProSPecs {
  SID: number;
  Name: string;
  Specs: Spec[];
}

export interface CheckoutResponse {
  FieldList: string[];
  PBMap: Record<string, string[]>;
  Count: number;
  ProSPecs: ProSPecs[];
  TableData: string[];
  TotalRow: number[];
  PricePerKgRow: number[];
  PricePerItemRow: number[];
  TotalAmountRow?: number[];
  totalAmountRow?: number[];
}

export interface ProPriceRequest {
  productName: string;
  spec: string;
  total: number;
  pricePerKg: number;
  pricePerItem: number;
  totalAmount: number;
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function getDayData(startTime: Date | string, endTime: Date | string) {
  return post<CheckoutResponse>('/api/checkout/daydata', {
    startTime: toISO(startTime),
    endTime: toISO(endTime),
  });
}

export function setProductBelong(mapp: Record<string, string[]>) {
  return put<null>('/api/checkout/product-belong', mapp);
}

export function setPrice(list: ProPriceRequest[]) {
  return post<CheckoutResponse>('/api/checkout/price', list);
}

export function exportCheckout() {
  return post<{ url: string }>('/api/checkout/export');
}
```

- [ ] **Step 4: system.ts**

创建 `frontend/src/api/system.ts`：

```ts
import { get } from './request';

export function getAIModels() {
  return get<Record<string, number>>('/api/system/ai-models');
}
```

- [ ] **Step 5: stream.ts（SSE 封装，本次不接入界面）**

创建 `frontend/src/api/stream.ts`：

```ts
import { BASE } from './request';

export interface StreamHandlers {
  onData: (data: string) => void;
  onDone?: () => void;
}

// EventSource 不支持 POST，改用 fetch + ReadableStream 手动解析 SSE
export async function streamRecognition(
  text: string,
  aimodel: number,
  previewType: number,
  handlers: StreamHandlers,
): Promise<void> {
  const res = await fetch(`${BASE}/api/order/recognition/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, aimodel, previewType }),
  });

  if (!res.ok || !res.body) {
    throw new Error(`HTTP ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let idx: number;
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      const raw = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 2);
      const data = raw
        .split('\n')
        .filter(line => line.startsWith('data: '))
        .map(line => line.slice(6))
        .join('\n');
      if (data === 'DONE') {
        handlers.onDone?.();
        return;
      }
      if (data) handlers.onData(data);
    }
  }
  handlers.onDone?.();
}
```

- [ ] **Step 6: vite-env.d.ts**

创建 `frontend/src/vite-env.d.ts`：

```ts
/// <reference types="vite/client" />
```

- [ ] **Step 7: 提交**

```bash
git add frontend/src/api frontend/src/vite-env.d.ts
git commit -m "feat(frontend): 新增 api 层 fetch 封装与 SSE 客户端"
```

---

### Task 12: App.vue 替换调用点并删除更新模块

**Files:**
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: 替换 import 块（第 2、3、8、16-18 行）**

第 2-3 行改为：

```ts
import { ref, computed } from 'vue';
import { Info, Download, RotateCw, SlidersHorizontal, Wallet, DollarSign } from 'lucide-vue-next';
```

删除第 8 行 `import demo from './components/demo.vue';`

第 16-18 行三行整体替换为：

```ts
import * as orderApi from './api/order';
import * as checkoutApi from './api/checkout';
```

- [ ] **Step 2: 删除版本更新状态与事件监听（第 58-102 行）**

删除从 `// 版本更新状态` 到 `});`（`onUnmounted` 块结束）的整段共 45 行，即：

```ts
// 版本更新状态
const showDownloadDialog = ref(false);
... （downloadProgress / downloadTotal / downloadPercentage / downloadSpeed /
     downloadComplete / downloadFilePath / downloadErrors / downloadError /
     logDownloadError / unsubscribers / onMounted / onUnmounted 全部）
});

```

- [ ] **Step 3: 替换识别调用（第 119 行）**

```ts
    const result = await orderApi.recognition(inputText.value, modelId, previewType);
```

- [ ] **Step 4: 替换删除预览调用（第 164-180 行 deleteOrder 整体）**

```ts
const deleteOrder = async (id: string) => {
  try {
    await orderApi.deletePreview(parseInt(id));
    statusMessage.value = '删除成功';

    // 刷新数据
    await refreshData();

    // 3秒后恢复默认状态
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('删除失败:', error);
    statusMessage.value = '删除出错：后端服务异常';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  }
};
```

- [ ] **Step 5: 替换预览查询（第 211 行）**

```ts
    const result = await orderApi.getPreview();
```

- [ ] **Step 6: 替换入库调用（第 273-274 行）**

```ts
    await orderApi.createOrder(selectedDateTime.value);
    statusMessage.value = '订单导入成功';
```

- [ ] **Step 7: 替换字段配置保存（第 302-317 行 handleSaveFieldConfig 整体，改为 async）**

```ts
const handleSaveFieldConfig = async (fieldMap: Record<string, string[]>) => {
  console.log('保存字段配置:', fieldMap);

  try {
    await checkoutApi.setProductBelong(fieldMap);
    console.log('字段配置保存成功');
    statusMessage.value = '字段配置保存成功';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('保存字段配置失败:', error);
    alert('保存字段配置失败，请稍后重试');
  }

  showFieldDialog.value = false;
};
```

- [ ] **Step 8: 替换价格保存（第 403 行）**

```ts
    const response = await checkoutApi.setPrice(proPriceRequests.value);
```

- [ ] **Step 9: 替换账单生成（第 450 行，含下载跳转）**

```ts
    const response = await checkoutApi.exportCheckout();
    console.log('结算账单成功:', response);
    if (response?.url) {
      window.open(response.url, '_blank');
    }
    statusMessage.value = '财务账单生成成功';
```

- [ ] **Step 10: 删除版本处理函数（第 489-514 行）**

删除从 `// 处理版本更新` 到 `const handleCloseDownload = () => { showDownloadDialog.value = false; };` 的整段（`handleUpdateDismiss` / `handleStartUpdate` / `handleRestart` / `handleCloseDownload`）。

- [ ] **Step 11: 删除两个版本弹窗模板（第 724-824 行）**

删除从 `<!-- Version Update Notification Dialog -->` 开始，到 `</div>`（`showDownloadDialog` 弹窗结束）为止的整段模板，保留其后的空行与 `</main>`。

- [ ] **Step 12: 提交**

```bash
git add frontend/src/App.vue
git commit -m "refactor(frontend): App.vue 改用 api 层并删除版本更新模块"
```

---

### Task 13: 组件调用点替换

**Files:**
- Modify: `frontend/src/components/Main/OrderInput.vue`、`OrderTable.vue`、`OrderDetailsView.vue`、`Checkout/CheckoutView.vue`、`Checkout/CheckQuery.vue`

- [ ] **Step 1: OrderInput.vue**

第 3 行后新增 import：

```ts
import * as systemApi from '../../../api/system';
```

第 31 行改为：

```ts
    const models = await systemApi.getAIModels();
```

- [ ] **Step 2: OrderTable.vue**

第 4 行

```ts
import { SetpreviewData } from '../../../wailsjs/go/main/App';
```

替换为

```ts
import * as orderApi from '../../../api/order';
```

第 50 行改为：

```ts
      await orderApi.setPreview(excelAddressInfo);
```

- [ ] **Step 3: OrderDetailsView.vue**

第 19 行

```ts
import { OrderQuery } from '../../../wailsjs/go/main/App';
```

替换为

```ts
import * as orderApi from '../../../api/order';
```

第 250 行与第 331 行的 `await OrderQuery(requestParams);` 均改为：

```ts
    const result = await orderApi.queryOrders(requestParams);
```

- [ ] **Step 4: CheckoutView.vue**

在 `<script setup>` 的 import 区（第 1-5 行附近）新增：

```ts
import * as checkoutApi from '../../../../api/checkout';
```

第 182-196 行 `handleSaveFieldConfig` 整体替换为：

```ts
const handleSaveFieldConfig = async (fieldMap: Record<string, string[]>) => {
  console.log('保存字段配置:', fieldMap);

  try {
    await checkoutApi.setProductBelong(fieldMap);
    console.log('字段配置保存成功');
  } catch (error) {
    console.error('保存字段配置失败:', error);
    alert('保存字段配置失败，请稍后重试');
  }

  showFieldDialog.value = false;
};
```

- [ ] **Step 5: CheckQuery.vue**

在 `<script setup>` 的 import 区新增：

```ts
import * as checkoutApi from '../../../../api/checkout';
```

第 153-157 行替换为：

```ts
    // 调用后端日数据接口
    const response = await checkoutApi.getDayData(startDate, endDate);
```

- [ ] **Step 6: 提交**

```bash
git add frontend/src/components
git commit -m "refactor(frontend): 组件调用点改用 api 层"
```

---

### Task 14: 前端清理（main.ts / types.ts / tsconfig / wailsjs / stores）

**Files:**
- Modify: `frontend/src/main.ts`、`frontend/src/types.ts`、`frontend/tsconfig.json`
- Delete: `frontend/wailsjs/`、`frontend/src/stores/`

- [ ] **Step 1: main.ts 全文替换**

```ts
import { createApp } from 'vue';
import App from './App.vue';
import './index.css';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn';

const app = createApp(App);
app.use(ElementPlus, {
  locale: zhCn
});
app.mount('#root');
```

- [ ] **Step 2: types.ts 删除 window.go 声明**

删除 `frontend/src/types.ts` 中第 10 行起的注释 `// 声明window.go对象的类型` 及其后的整个 `declare global { ... }` 块，仅保留 `Order` 接口。

- [ ] **Step 3: tsconfig.json 去掉 wailsjs 排除**

删除 `frontend/tsconfig.json` 中的这一行：

```json
    "exclude": ["wailsjs"]
```

> 若删除后 `exclude` 变空数组则一并删除该键；注意保持 JSON 合法（去掉前一行的尾逗号）。

- [ ] **Step 4: 删除 Wails 生成物与更新 store**

```bash
git rm -r frontend/wailsjs frontend/src/stores
```

- [ ] **Step 5: 类型检查**

Run: `cd frontend && npm run lint`
Expected: 无错误输出（`vue-tsc --noEmit --skipLibCheck`）

- [ ] **Step 6: 提交**

```bash
git add frontend/src/main.ts frontend/src/types.ts frontend/tsconfig.json
git commit -m "chore(frontend): 删除 wailsjs 绑定与更新 store，清理入口"
```

---

### Task 15: Vite 开发代理

**Files:**
- Modify: `frontend/vite.config.ts`

- [ ] **Step 1: 加 proxy**

把 `frontend/vite.config.ts` 全文替换为：

```ts
import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';
import {defineConfig} from 'vite';

export default defineConfig(() => {
  return {
    plugins: [vue(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
        '/uploads': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
  };
});
```

- [ ] **Step 2: 提交**

```bash
git add frontend/vite.config.ts
git commit -m "chore(frontend): 开发环境代理 /api 与 /uploads 到后端"
```

---

### Task 16: 仓库清理（Wails 与旧残留）

**Files:**
- Delete: 根 `wails.json` 已在 Task 9 删除；本任务删除其余残留

- [ ] **Step 1: 逐项确认存在后再删除**

```bash
ls -d build .trae tmp frontend/dist frontend/.env.example frontend/vite.config.js frontend/src/main.js frontend/src/components/demo.vue frontend/src/components/Header/index.js index.html package.json package-lock.json 2>/dev/null
```

Expected: 列出这些路径中实际存在的项；对不存在的项跳过，不报错。

- [ ] **Step 2: 删除**

```bash
git rm -r build .trae tmp frontend/dist frontend/.env.example frontend/vite.config.js frontend/src/main.js frontend/src/components/demo.vue frontend/src/components/Header/index.js index.html package.json package-lock.json 2>/dev/null
rm -rf build .trae tmp frontend/dist
```

> `tmp/`、`logs/`、`uploads/` 若未被 git 跟踪，`git rm` 会报错，用 `rm -rf` 兜底即可；`logs/` 与 `uploads/` 保留。

- [ ] **Step 3: 确认保留项仍在**

Run: `ls`
Expected: 至少包含 `frontend/`、`go/`、`main.go`、`settings.yaml`、`aimap.json`、`productbelong.json`、`systemPrompt.txt`、`logs/`、`uploads/`、`README.md`

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "chore: 删除 Wails 构建产物与旧版残留文件"
```

---

### Task 17: 前端整体校验

**Files:** 无（验证任务）

- [ ] **Step 1: 全仓无 wails 引用**

Run: `grep -rn "wailsjs\|wails" frontend/src frontend/*.ts frontend/*.json go main.go 2>/dev/null`
Expected: 无匹配

- [ ] **Step 2: 全仓无 window['go']**

Run: `grep -rn "window\['go'\]\|window\.go" frontend/src`
Expected: 无匹配

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npm run lint`
Expected: 无错误

- [ ] **Step 4: 构建**

Run: `cd frontend && npm run build`
Expected: 构建成功，产物生成在 `frontend/dist/`

- [ ] **Step 5: 提交（如有产物变更以外的改动）**

```bash
git status --short
```

若仅有 `frontend/dist/`（已被 gitignore 或已删除）则无需提交；若 `package-lock.json` 等被重新生成，按需提交。

---

### Task 18: 端到端手工验证

**Files:** 无（人工验证）

前置：一个终端跑 `go run main.go`（:8080），另一个终端 `cd frontend && npm run dev`（:3000），浏览器打开 `http://localhost:3000`。

- [ ] **Step 1: 识别 → 预览**

在「订单录入」页粘贴订单文本，选择模型，点识别。
Expected: 预览列表出现识别结果；「追加」模式下再次识别会累加而不是覆盖。

- [ ] **Step 2: 预览编辑与删除**

在预览表格中编辑一条记录并保存；删除一条记录。
Expected: 编辑后刷新列表仍为新值；删除后记录消失，且 `global.AddressList` 同步（切换页面再回来数据不变）。

- [ ] **Step 3: 入库**

点「导入数据库」，在确认弹窗选择日期并确认。
Expected: 提示导入成功；重复导入同一批数据不会产生重复记录。

- [ ] **Step 4: 导出 Excel（服务端）**

Run: `curl -s -X POST http://localhost:8080/api/order/export`
Expected: 返回 `{"code":0,"msg":"ok","data":{"url":"/uploads/....xlsx"}}`
Run: `curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/uploads/<上一步的文件名>`
Expected: `200`

- [ ] **Step 5: 结账查询 → 字段配置 → 价格 → 账单导出**

在「结账」页按日期范围查询。
Expected: 表格数据与字段数据出现；未分类字段可配置归属（保存后重新查询仍生效）。
编辑价格并保存。
Expected: 每斤/元、每件/元、总金额行随价格刷新。
点「财务结算」。
Expected: 浏览器打开 `/uploads/xxxx结账.xlsx` 下载地址（中文文件名可正常下载）。

- [ ] **Step 6: 订单详情分页**

在「订单详情」页查询、翻页、导出。
Expected: 分页数据正常；导出为前端 XLSX 本地下载。

- [ ] **Step 7: 记录验证结果**

验证通过后，按需提交任何在此期间产生的修复。

---

## 自审记录

- **规格覆盖**：§3 目录分层 → Task 6/7/8；§4 API 端点 → Task 7/8（13 个端点全部注册并测试）；§4.3 SSE → Task 7 Step 4；§5 后端改造 1-8 项 → Task 4/5/9；§6 前端改造 1-5 项 → Task 11-16；§7 配置 → Task 3/15；§8 删除清单 → Task 9/14/16；§10 验收标准 → Task 10/17/18。
- **类型一致性**：`uploadURL` / `ensureExportDir`（application）、`StreamCallback` / `DoneCallback`（server/order）、`RecognitionStream`（handler）、`Setup`（router）、`ExcelAddressInfo` / `CheckoutResponse`（前端 api）在各任务中命名一致。
- **已知取舍**：`/api/order/export` 与 `/api/checkout/export` 的下载 URL 用 `url.PathEscape` 编码中文文件名；前端 `window.open` 直接跳转，生产环境依赖同域或反代暴露 `/uploads`。
