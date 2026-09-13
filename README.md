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
frontend/       Vue 3 + Vite 前端
main.go         程序入口
settings.yaml   主配置（数据库、服务端口、AI 模型、邮件等）
aimap.json      AI 模型映射
productbelong.json  产品归属映射
systemPrompt.txt    识别提示词
```

## 开发运行

后端（监听 `:8080`）：

```bash
go run main.go
```

前端（开发服务器 `:3000`，已代理 `/api` 与 `/uploads` 到 `:8080`）：

```bash
cd frontend
npm install
npm run dev
```

## 构建

后端：`go build`。前端：`cd frontend && npm run build`（产物输出到 `frontend/dist/`）。

生产部署时，前端静态产物可与后端分域部署，此时需自行处理跨域：后端已配置 CORS 允许来源列表，见 `settings.yaml` 的 `server.allowOrigins`。

## 配置说明

配置集中在根目录 `settings.yaml`，包含：

- `log`：日志级别、应用名与日志目录；
- `db`：数据库连接列表（主机、端口、库名、账号密码、连接池与 debug 开关）；
- `email`：SMTP 邮件发送配置（服务器、端口、发件账号与授权码、TLS/SSL 开关）；
- `savePath`：导出文件（Excel、JSON）的保存目录；
- `aimodel`：AI 模型配置（按模型名分组，含 apikey、apiurl、超时与模型名）；
- `server`：服务监听端口与 CORS 允许来源。

请勿将含有真实密钥的 `settings.yaml` 提交到公开仓库。
