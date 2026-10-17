# 订单识别与结账系统

订单识别 / 结账系统：Gin 后端（Go）+ Vue 3 前端（Vite + TypeScript）。

## 目录结构

```
go/
  router/       路由注册
  handler/      HTTP 处理层
  application/  业务逻辑层
  server/       数据服务层
  conf/ core/ global/ flags/ models/ utils/ common/   配置、初始化与公共组件
  config/       运行时配置（settings.yaml、aimap.json、productbelong.json、systemPrompt.txt）
web/            Vue 3 + Vite 前端
main.go         程序入口
```

## 开发运行

后端（监听 `:8080`）：

```bash
go run main.go
```

前端（开发服务器 `:3000`，已代理 `/api` 与 `/uploads` 到 `:8080`）：

```bash
cd web
npm install
npm run dev
```

## 构建

后端：`go build`。前端：`cd web && npm run build`（产物输出到 `web/dist/`）。

生产部署时，前端静态产物可与后端分域部署：构建前设置环境变量 `VITE_API_BASE` 指向后端地址（如 `VITE_API_BASE=https://api.example.com npm run build`），并把前端域名加入后端 `go/config/settings.yaml` 的 `server.allowOrigins`。同域部署（由后端托管前端产物）无需设置该变量。

## 配置说明

配置集中在 `go/config/settings.yaml`（不入库；模板见 `go/config/settings.example.yaml`，复制为 `settings.yaml` 并填入真实值），包含：

- `log`：日志级别、应用名与日志目录；
- `db`：数据库连接列表（主机、端口、库名、账号密码、连接池与 debug 开关）；
- `email`：SMTP 邮件发送配置（服务器、端口、发件账号与授权码、TLS/SSL 开关）；
- `savePath`：导出文件（Excel、JSON）的保存目录；
- `aimodel`：AI 模型配置（按模型名分组，含 apikey、apiurl、超时与模型名）；
- `server`：服务监听端口与 CORS 允许来源。

`settings.yaml` 含真实密钥（数据库密码、SMTP 授权码、AI apikey），已被 `.gitignore` 忽略，请勿提交到公开仓库；仓库内仅保留占位符模板 `go/config/settings.example.yaml`。
