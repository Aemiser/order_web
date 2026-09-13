# Wails 桌面应用 → 前后端分离 Web 应用重构设计

- 日期：2026-09-13
- 状态：已确认，待实施
- 范围：`F:\project\go\order_web` 全仓库

## 1. 背景与目标

### 1.1 现状

`order_web` 是基于 **Wails v2** 的桌面应用「订单识别 / 结账系统」，业务链路：

> 粘贴订单文本 → AI（DeepSeek / Qwen）结构化识别 → 预览编辑（内存态） → 导入 MySQL → 结账统计（按日期范围） → 导出 Excel / 生成财务账单

### 1.2 目标

重构为**前后端分离的 Web 应用**：

- Go 后端提供 HTTP API（Gin），前端 Vue 3 独立部署，通过 CORS 授权访问
- 删除全部 Wails 依赖代码与构建产物，仓库仅保留 `frontend/`、`go/` 及必要配置/数据文件
- 业务逻辑尽可能原样复用，重构不改业务行为

### 1.3 已确认的技术决策

| 决策点 | 结论 | 理由 |
|---|---|---|
| HTTP 框架 | **Gin** | 中文社区主流、中间件齐全 |
| 流式推送 | **SSE**（后端实现，前端本次不接入） | 单向文本流天然匹配；保留能力备用 |
| 部署形态 | **完全分离** | 前端独立部署（Nginx 等），后端纯 API + CORS |
| 版本更新模块 | **整体删除** | 桌面自更新（下载 exe → bat 替换重启）在 Web 无意义 |

## 2. 现状分析

### 2.1 可复用资产（保持不动）

| 目录 | 内容 |
|---|---|
| `go/conf/` | 配置结构体（Log / DB / Email / AiModel / SavePath；Version 段随更新模块删除） |
| `go/core/` | 初始化：读配置、连 DB、日志、`aimap.json`、`productbelong.json`、`systemPrompt.txt` |
| `go/global/` | 全局变量：`Config`、`Db`、`AddressList`（预览数据）、`ProductBelongMap`、`AiMap`、`SystemPrompt` |
| `go/flags/` | 命令行参数（`-f` 配置文件、`-db` 数据迁移） |
| `go/models/` | GORM 模型（`AddressInfo`、`Order`、`LogModels` 等） |
| `go/utils/` | AI 请求、Excel 生成、订单文本处理、时间工具 |
| `go/common/` | 泛型分页查询 `ListQuery` |
| `go/server/order/` | 识别（同步/流式）、预览 CRUD、入库、查询、Excel 保存 |
| `go/server/checkout/` | 销量统计、价格计算、账单生成、规格管理器 |
| `go/server/log/` | 操作日志（带 1 分钟去重） |

### 2.2 Wails 耦合点（全部需处理）

**后端：**

| 位置 | 耦合内容 | 处理方式 |
|---|---|---|
| `main.go` | `wails.Run` + `embed frontend/dist` | 改为启动 Gin |
| `app.go`（全部） | 17 个绑定方法、`runtime.EventsEmit` / `OpenDirectoryDialog` / `BrowserOpenURL` | 逻辑迁入 handler/application 后删除文件 |
| `go/server/order/enter.go:39-49` | `EmitStreamData` / `EmitStreamDone` 依赖 `runtime.EventsEmit` | 删除这两个函数；回调机制保留，由 handler 注入 SSE 写入 |
| `go/server/checkout/enter.go:145,233-278` | `Checkout(ctx)` + `selectSaveDirectory` 用 `OpenDirectoryDialog` | 删除弹窗函数；保存目录改读 `settings.yaml` → `savePath.excel`；`Checkout()` 去掉 ctx |
| `go/server/version/`（整个包） | GitHub Release 下载 + bat 自我替换重启 | 整包删除 |

**前端：**

| 位置 | 耦合内容 | 处理方式 |
|---|---|---|
| `frontend/wailsjs/`（整个目录） | 自动生成的 JS/TS 绑定 | 删除，新增 `src/api/` fetch 封装 |
| `src/App.vue:16` | `import ... from '../wailsjs/go/main/App'` | 改为 api 模块 |
| `src/App.vue:17,78-102` | `EventsOn('download_progress'/'download_complete'/'download_error')` | 随更新模块删除 |
| `src/App.vue:307,403,450` | `window['go']['main']['App'][...]` 动态调用 | 改为 api 模块 |
| `src/main.ts:7-12` | `EventsOn('update_available')` | 删除 |
| 组件内 5 处 | `SetpreviewData` / `GetAIModel` / `OrderQuery` / `SetProductBelong` / `GetChackDaydata` | 改为 api 模块 |

### 2.3 关键数据流（重构后保持一致）

- **识别**：`OrderRec*` 内部按 `previewType`（追加 / 覆盖）写入 `global.AddressList`，并返回 `AddressToExcelAddress(global.AddressList)` 全量预览列表；流式实现在完成时同样更新时间 `global.AddressList`。
- **入库**：`CreateOrder(date)` 把 `global.AddressList` 中 `Isstorage == false` 的记录写入 DB，并回写标记。
- **结账**：`GetDaydata(start, end)` 查询区间数据写入包级变量 `addresinfolist` / `gspecsMap` / `checkoutresp`，`SetPrice` 基于这些状态计算金额，`Checkout` 基于状态生成 Excel。

## 3. 目标架构

### 3.1 目录结构

```
order_web/
├── frontend/                    # Vue 3 + Vite（独立部署）
│   ├── src/
│   │   ├── api/                 # 【新增】
│   │   │   ├── request.ts       #   fetch 封装：baseURL、统一解包、错误处理
│   │   │   ├── order.ts         #   订单 API
│   │   │   ├── checkout.ts      #   结账 API
│   │   │   ├── system.ts        #   系统 API
│   │   │   └── stream.ts        #   SSE 流式识别（fetch + ReadableStream）
│   │   ├── components/          #   现有组件（仅改调用点）
│   │   ├── App.vue / main.ts    #   删除更新模块相关代码
│   │   └── types.ts             #   删除 window.go 声明
│   └── vite.config.ts           #   开发代理 /api → :8080
│
├── go/
│   ├── router/                  # 【新增】路由注册、中间件挂载
│   │   └── enter.go
│   ├── handler/                 # 【新增】HTTP 层
│   │   ├── order.go
│   │   ├── checkout.go
│   │   └── system.go
│   ├── application/             # 【新增】应用服务层
│   │   ├── order.go
│   │   └── checkout.go
│   ├── server/                  # 领域层（保留，去 Wails）
│   │   ├── order/  checkout/  log/
│   ├── common/                  # 保留 + 新增 response.go
│   ├── conf/ core/ global/ flags/ models/ utils/   # 原样保留
│
├── main.go                      # 改为：初始化 + 启动 Gin
├── settings.yaml                # 保留 + 新增 server 段
├── aimap.json / productbelong.json / systemPrompt.txt   # 运行时数据，保留
├── uploads/                     # 导出文件落地目录
└── logs/                        # 运行日志
```

### 3.2 分层职责

| 层 | 职责 | 禁止 |
|---|---|---|
| `router/` | URL 定义、分组、中间件（CORS / Recover / 日志） | 写业务逻辑 |
| `handler/` | 解析请求（JSON / query / path）、参数校验、调用 application、按统一格式写响应、SSE 时注入 emit 回调 | 直接操作 `global`、直接调 server 层 |
| `application/` | 业务流程编排、全局状态读写（`global.AddressList`、价格副本）、返回下载 URL 等 Web 语义 | 感知 `*gin.Context` |
| `server/` | 纯领域逻辑：AI 调用、统计计算、Excel 生成、DB 读写 | 感知 HTTP / Wails |

依赖方向单向：`router → handler → application → server`。

### 3.3 分层示例

**导出 Excel**

```
router   POST /api/order/export
handler  调 application.ExportOrder()
application  调 server/order.SaveToExcel()（落到 uploads/），返回下载 URL
server   生成 xlsx（纯文件操作）
```

**流式识别**

```
router   POST /api/order/recognition/stream
handler  设置 SSE 头（text/event-stream, flush），传入 emit 回调
application  调 server/order.StreamOrderRec(..., emit, done)
server   现有回调实现原样复用 → handler 逐段写 SSE
```

## 4. API 设计

### 4.1 通用约定

- 前缀 `/api`，请求/响应均为 JSON（SSE 与文件下载除外）
- 统一响应：`{ "code": 0, "msg": "ok", "data": ... }`；`code != 0` 表示业务失败，`msg` 为错误信息
- 前端 `request.ts` 负责解包 `data` 并在 `code != 0` 时抛错，组件拿到的即业务数据
- 时间参数：JSON 中为 RFC3339 字符串，Go 侧解析为 `time.Time`

### 4.2 端点清单

| 方法 | 路由 | 原 Wails 方法 | 入参 | 出参（data） |
|---|---|---|---|---|
| GET | `/api/system/ai-models` | GetAIModel | — | `{"Deepseek":1,"Qwen":2}` |
| POST | `/api/order/recognition` | OrderRecognition | `{text, aimodel, previewType}` | `ExceladdressInfo[]` |
| POST | `/api/order/recognition/stream` | StartStreamRecognition | `{text, aimodel, previewType}` | SSE 流（见 4.3） |
| GET | `/api/order/preview` | GetpreviewData | — | `ExceladdressInfo[]` |
| PUT | `/api/order/preview` | SetpreviewData | `ExceladdressInfo` | null |
| DELETE | `/api/order/preview/:id` | DeletepreviewData | path `id` | null |
| POST | `/api/order/create` | CreateOrder | `{date}` | null |
| POST | `/api/order/export` | SaveToExcel | — | `{url}` |
| POST | `/api/order/query` | OrderQuery | `OrderQueryRequest` | `{list, count}` |
| POST | `/api/checkout/daydata` | GetChackDaydata + GetDaydata（同一实现，合并） | `{startTime, endTime}` | `CheckoutResponse` |
| PUT | `/api/checkout/product-belong` | SetProductBelong | `{...}` 商品归属 map | null |
| POST | `/api/checkout/price` | SetPriceData | `ProPriceRequest[]` | `CheckoutResponse` |
| POST | `/api/checkout/export` | SaveCheackToExcel | — | `{url}` |
| GET | `/uploads/*` | — | — | 静态文件下载 |

删除的 Wails 方法：`Greet`（demo）、`DownloadUpdate`、`RestartApplication`（随 version 模块删除）。

### 4.3 SSE 端点规范

- 请求：`POST /api/order/recognition/stream`，用 POST 规避长文本 URL 长度限制
- 响应头：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`Connection: keep-alive`
- 事件格式：每个 chunk 一条 `data: <内容>\n\n`；结束时发送 `data: DONE\n\n`
- 内容语义与现有回调一致：中间为 AI 增量文本，完成时推送一次全量预览 JSON 后发 `DONE`
- 前端 `api/stream.ts` 用 `fetch` + `ReadableStream` 手动解析（`EventSource` 不支持 POST），本次仅提供封装，界面不接入

## 5. 后端改造细节

1. **`main.go`**：`flags.Parse()` → 读配置 → 初始化日志/DB/映射表 → 启动 Gin（端口读配置）
2. **`setupRoutes`**（`go/router/enter.go`）：`gin.New()` + Recover/日志中间件 + CORS + 路由分组（`/api/system`、`/api/order`、`/api/checkout`）+ `/uploads` 静态目录
3. **`go/conf`**：`Config` 增加 `Server` 段：
   ```yaml
   server:
       port: 8080
       allowOrigins:
           - "http://localhost:3000"
   ```
4. **`go/common/response.go`**：统一响应 `{code,msg,data}` 与 `Ok/Err` 帮助函数
5. **`go/server/order/enter.go`**：删除 `EmitStreamData` / `EmitStreamDone` 与 wails import；`StreamOrderRec` 去掉 `ctx context.Context` 首参
6. **`go/server/checkout/enter.go`**：删除 `selectSaveDirectory`、wails import 与 `context` 参数；`Checkout()` 直接读 `global.Config.SavePath.Excel` 作为输出目录
7. **删除** `go/server/version/`、`go/server/config/`（整体已注释）、`go/testdata/`
8. **`go.mod`**：移除 `github.com/wailsapp/wails/v2`，新增 `github.com/gin-gonic/gin`，`go mod tidy`

> 注：`go/server/version` 删除后，`conf.VersionConfig` 与 `settings.yaml` 的 `version` 段一并移除。

## 6. 前端改造细节

1. **新增 `src/api/`**：
   - `request.ts`：`const BASE = import.meta.env.VITE_API_BASE ?? ''`；封装 `get/post/put/del`，统一解包与错误抛出
   - `order.ts` / `checkout.ts` / `system.ts`：按端点导出函数，签名与旧 Wails 绑定保持一致（降低调用点改动量）
   - `stream.ts`：SSE 读取封装（本次保留不接入）
2. **调用点替换（7 处）**：
   - `App.vue`：`OrderRecognition` → `order.recognition`；`GetpreviewData` → `order.getPreview`；`DeletepreviewData` → `order.deletePreview`；`CreateOrder` → `order.create`；`window['go']...SetProductBelong` → `checkout.setProductBelong`；`SetPriceData` → `checkout.setPrice`；`SaveCheackToExcel` → `checkout.exportExcel`（成功后 `window.open(data.url)` 下载）
   - `OrderTable.vue`：`SetpreviewData` → `order.setPreview`
   - `OrderDetailsView.vue`：`OrderQuery` → `order.query`
   - `OrderInput.vue`：`GetAIModel` → `system.getAIModels`
   - `CheckoutView.vue`：`SetProductBelong` → `checkout.setProductBelong`
   - `CheckQuery.vue`：`GetChackDaydata` → `checkout.getDayData`
3. **删除更新模块**：`App.vue` 中下载状态变量（59-72 行）、事件监听（77-102 行）、更新处理函数（490-514 行）、两个弹窗模板（725-824 行）；`main.ts` 事件监听；`stores/update.ts`；`stores/` 目录
4. **`vite.config.ts`**：新增 `server.proxy` 将 `/api` 与 `/uploads` 代理到 `http://localhost:8080`（开发环境）
5. **导出下载**：后端返回 `{url}`，前端 `window.open(url)`；生产环境需前端能访问后端 `/uploads`（同一域名或 CORS/代理）

## 7. 配置、运行与部署

### 7.1 settings.yaml

```yaml
log: ...
db: ...              # 保持不变
email: ...
savePath:
    excel: ./uploads   # 导出目录（结账账单、识别结果 Excel）
    json: ./uploads
aimodel: ...
server:                # 【新增】
    port: 8080
    allowOrigins:
        - "http://localhost:3000"
```

### 7.2 运行

- **开发**：后端 `go run main.go`（:8080）；前端 `npm run dev`（:3000，Vite 代理 `/api`、`/uploads` → :8080）
- **生产**：后端二进制 + `settings.yaml` + 三个数据文件（`aimap.json`、`productbelong.json`、`systemPrompt.txt`）独立运行；前端 `npm run build` 产物交 Nginx 托管，通过 `allowOrigins` 授权

## 8. 删除清单

| 路径 | 原因 |
|---|---|
| `app.go` | Wails 绑定层，逻辑迁入 handler/application |
| `wails.json` | Wails 构建配置 |
| `build/` | Wails 构建产物（含旧 exe、配置副本、日志） |
| `go/server/version/` | 桌面自更新，Web 无意义 |
| `go/server/config/` | 整体已注释的死代码 |
| `go/testdata/` | 13 个 `package main` 实验脚本 |
| 根目录 `index.html` | 旧版 CDN 单页演示（41KB） |
| 根目录 `package.json` / `package-lock.json` | 旧残留（仅 tailwind postcss） |
| `frontend/wailsjs/` | 生成的绑定 |
| `frontend/vite.config.js` | 与 `.ts` 重复 |
| `frontend/src/main.js` | 与 `main.ts` 重复 |
| `frontend/src/stores/` | 更新状态，随模块删除 |
| `frontend/src/components/demo.vue` | 演示组件（App.vue 中已注释） |
| `frontend/src/components/Header/index.js` | 空导出残留 |
| `frontend/.env.example` | Gemini AI Studio 残留 |
| `.trae/` | IDE 规则残留 |
| `tmp/` | 空目录残留 |
| `frontend/dist/` | 旧构建产物（重建） |

**保留**：`frontend/`、`go/`、`main.go`、`settings.yaml`、`aimap.json`、`productbelong.json`、`systemPrompt.txt`、`logs/`、`uploads/`、`.git/`、`.idea/`、`README.md`

## 9. 已知限制

`global.AddressList`（预览数据）与结账价格（`proPrice`、`checkoutresp`）是**全局单例状态**，并发用户会互相覆盖。本次重构保持单用户/单会话假设（与桌面版行为一致）；多用户隔离需另行引入会话机制，不在本次范围。

## 10. 验收标准

1. `go build ./...` 通过，`go.mod` 中无 Wails 依赖；全仓库不再出现 `wails` 引用
2. `go vet ./...` 通过
3. 后端启动后 13 个业务端点 + `/uploads` 静态服务均可访问，统一响应格式正确
4. 前端 `npm run build` 与 `npm run lint`（vue-tsc）通过；全仓库不再出现 `wailsjs` / `window['go']`
5. 端到端手工验证（开发模式）：
   - 识别订单 → 预览列表出现、可编辑、可删除
   - 确认入库 → 数据库新增记录
   - 导出 Excel → 浏览器成功下载文件
   - 结账页按日期查询 → 字段配置 / 价格编辑 / 财务结算导出均正常
   - 订单详情页分页查询正常
