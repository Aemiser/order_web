# OrderMaster Pro — 前端

订单识别与结账系统的 Vue 3 + Vite + TypeScript 前端。项目整体说明（后端、配置、部署）见仓库根目录的 [README.md](../README.md)。

## 开发

```bash
npm install
npm run dev      # http://localhost:3000，已代理 /api 与 /uploads 到后端 :8080
```

需先启动后端（仓库根目录执行 `go run main.go`）。

## 构建

```bash
npm run build    # 产物输出到 dist/
```

## 结构

- `src/App.vue`：应用外壳与页面切换
- `src/components/`：业务组件（订单录入 / 预览表格 / 订单详情、结账、设置）
- `src/api/`：后端接口封装；`request.ts` 为统一请求层，baseURL 可用环境变量 `VITE_API_BASE` 覆盖（默认留空走同域 / 开发代理）
- `src/index.css`：全局样式与 Tailwind 配置
