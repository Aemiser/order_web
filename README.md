# 订单识别与结账系统

订单识别 / 结账系统：Gin 后端（Go）+ Vue 3 前端（Vite + TypeScript）。

双品牌：**一点点**（原有链路，斤计价）与 **吴饮**（吾饮良品香水柠檬，2026-10 新增，
独立数据库、件计价、按仓库标签分组结账）。前端侧边栏顶部切换品牌。

## 吴饮（吾饮良品）链路

- **数据入口**：粘贴群聊文本 AI 识别（专用提示词 `go/config/systemPromptWuyin.txt`，
  提取收件人/手机号/地址/省份/商品件数/发送人），或导入各仓日报 xlsx
  （表头：日期/客户编码/客户/电话/物料编码/规格型号/物料名称/库存单位/实发数量）。
- **标签**：按群聊发送人昵称匹配关键词自动打标（合肥仓/广西仓/云南仓…），
  标签与关键词在「标签管理」页维护；匹配不到落「未分配」，预览与详情页均可手动改。
- **结账**：按标签分组的商品×日期×件数账单，单价（元/件）落库回填，
  导出 Excel 每标签一个 sheet，可全量或按单标签导出。
- **独立库**：`settings.yaml` 的 `dbWuyin` 配置独立 MySQL 库（模板见
  `go/config/settings.example.yaml`）；未配置时 `/api/wuyin/*` 端点统一报错，
  一点点链路不受影响。首次使用建库后执行 `go run main.go -dbwuyin` 建表
  （空表会播种默认仓库标签）。

## 目录结构

```
go/
  router/       路由注册
  handler/      HTTP 处理层
  application/  业务逻辑层
  server/       数据服务层（server/wuyin 为吴饮品牌平行链路）
  conf/ core/ global/ flags/ models/ utils/ common/   配置、初始化与公共组件
  config/       运行时配置（settings.yaml、aimap.json、productbelong.json、systemPrompt.txt、systemPromptWuyin.txt）
web/            Vue 3 + Vite 前端（components/Wuyin/ 为吴饮页面）
deploy/         systemd 单元与 nginx 子路径片段模板
scripts/        部署期在服务器上执行的脚本
.github/        CI/CD 工作流
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

## 部署

推送到 `main` 分支即自动发布到生产（`.github/workflows/deploy.yml`）。流水线为
`校验 → 构建 → 部署`三段：交叉编译 Linux 二进制、以子路径基准构建前端、打包后经 SSH
上传，由 `scripts/remote_deploy.sh` 在服务器上完成替换、健康检查与失败回滚。

### 架构

后端由 systemd 单元 `order-web.service`（模板见 `deploy/order_web.service`）托管，监听
本机回环端口，不对外暴露。nginx 通过子路径 `/rec` 反代：前端静态资源走 `alias`，
`/rec/api/`、`/rec/uploads/` 剥掉前缀后转发给后端，SPA 深链回落到 `/rec/index.html`。

前端因此必须构建到 `/rec` 前缀，由 `web/.env.production` 的 `VITE_BASE=/rec/` 与
`VITE_API_BASE=/rec` 控制。**若同一域名的根路径已被别的应用占用**（`/api/`、`/uploads/`
已反代到其他后端），缺少这层前缀会让请求打到别人的服务上。nginx 片段模板见
`deploy/nginx-rec.conf.example`。

SSE 流式端点（`/api/order/recognition/stream`）要求 nginx 侧 `proxy_buffering off`，
否则增量会被攒到结束才吐出，模板中已包含。

### 需要配置的仓库 Secrets

以下值均不得写入仓库，只存在于 GitHub Secrets：

| Secret | 说明 |
| --- | --- |
| `ECS_HOST` | 服务器地址 |
| `ECS_PORT` | SSH 端口 |
| `ECS_USER` | SSH 用户 |
| `ECS_SSH_KEY` | 部署专用私钥（`~/.ssh/github_actions_deploy`） |
| `ECS_KNOWN_HOSTS` | 服务器主机公钥条目，用于固定 known_hosts 防中间人 |
| `APP_HEALTH_URL` | 公网健康检查完整 URL |
| `APP_BASE_URL` | 公网前端入口完整 URL |

部署私钥应为一把**专用**密钥，不要复用个人日常登录密钥；泄露时可单独吊销。

### 服务器上不可被部署覆盖的文件

`scripts/remote_deploy.sh` 只同步二进制、前端产物与静态配置（`aimap.json`、
`systemPrompt.txt`）。以下两个文件属服务器本地资产，脚本永不覆盖：

- `go/config/settings.yaml` —— 含密钥，且端口、CORS 来源等生产取值与开发环境不同；
- `go/config/productbelong.json` —— 会被 `PUT /api/checkout/product-belong` 在运行期回写，
  覆盖即丢失线上数据。

首次部署需人工把这两个文件放到 `/opt/order_web/go/config/` 下。

### 手动运维

```bash
systemctl status order-web.service          # 服务状态
journalctl -u order-web.service -f          # 实时日志
tail -f /opt/order_web/logs/$(date +%F)/order_server.log   # 应用日志

# 回滚到上一版（部署脚本会自动保留 server.prev 与 order_web.prev）
mv /opt/order_web/server.prev /opt/order_web/server && systemctl restart order-web.service
```

