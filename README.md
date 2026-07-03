# SafeGate

一个可配置的 **IP 风控网关 / 反向代理防火墙**，支持管理员配置域名映射、真实 IP 透传、接口风控拦截与自定义 JSON 数据转发。

## 定位

把它放在你的 Nginx（如 1Panel）后面作为上游服务：

```
用户 → Nginx（80/443、SSL、域名解析）
        ↓
    SafeGate（风控、转发、转换）
        ↓
    目标站点 / Upstream
```

## 技术栈

- **后端**：Go + Gin + GORM + PostgreSQL + Redis
- **前端**：React + TypeScript + Vite + Tailwind CSS + shadcn/ui
- **部署**：Docker Compose（本地开发） / 1Panel Nginx 反代（生产）

## 快速开始（本地 Docker）

```bash
cp .env.example .env
# 编辑 .env，设置 ADMIN_PASSWORD 和 JWT_SECRET
docker compose up -d --build
```

- 管理后台：`http://localhost:18081`（后端 `MODE=all` 直接挂载 `web/dist`）
- 后端 API：`http://127.0.0.1:18081/api/admin`
- 代理测试：`curl http://127.0.0.1:18080/post`

首次启动时若未设置 `ADMIN_PASSWORD`，请在后端日志中查看随机生成的管理员密码：

```bash
docker compose logs -f backend
```

## 功能特性

- 管理员登录与 JWT 认证
- 域名映射：绑定域名 → 目标域名
- 真实 IP 识别与透传（`X-Real-IP`、`X-Forwarded-For`、`CF-Connecting-IP`）
- 接口风控：重复 IP 拦截、速率限制
- 自定义请求体/响应体 JSON 字段映射
- 访问日志与移动端响应式管理后台

## 文档

详细设计文档见 [docs](./docs)。

## 开发

### 后端

```bash
# 仅启动 API + 代理（配合 1Panel 等独立前端）
ADMIN_PASSWORD=admin JWT_SECRET=secret \
DATABASE_URL="postgres://safegate:safegate@127.0.0.1:5434/safegate?sslmode=disable" \
REDIS_ADDR="127.0.0.1:6379" \
PORT=18080 ADMIN_PORT=18081 MODE=api go run ./cmd/server

# 后端自带管理页面（单文件部署 / 本地 Docker）
ADMIN_PASSWORD=admin JWT_SECRET=secret \
DATABASE_URL="postgres://safegate:safegate@127.0.0.1:5434/safegate?sslmode=disable" \
REDIS_ADDR="127.0.0.1:6379" \
PORT=18080 ADMIN_PORT=18081 MODE=all go run ./cmd/server
```

### 前端开发

```bash
cd web
cp .env.example .env.local
npm install
npm run dev
```

开发服务器默认会把 `/api` 代理到 `http://127.0.0.1:18081`。

如需修改 API 地址，编辑 `web/.env.local`：

```env
# 前后端同域（推荐）
VITE_API_BASE_URL=/api

# 跨域访问本地后端
VITE_API_BASE_URL=http://127.0.0.1:18081/api
```

## License

MIT
