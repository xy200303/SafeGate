<div align="center">

# 🛡️ SafeGate

**可配置的 IP 风控网关 / 反向代理防火墙**

为 Nginx / 1Panel / Caddy 身后的站点提供域名映射、真实 IP 透传、接口风控拦截、JSON 请求体转换与访问日志审计。

<p>
  <a href="https://github.com/xy200303/SafeGate/stargazers"><img src="https://img.shields.io/github/stars/xy200303/SafeGate?style=social" alt="GitHub Stars"></a>
  <a href="https://github.com/xy200303/SafeGate/issues"><img src="https://img.shields.io/github/issues/xy200303/SafeGate" alt="GitHub Issues"></a>
</p>

<p>
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/Gin-1.12-008ECF?logo=go&logoColor=white" alt="Gin">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker">
</p>

<p>
  <a href="./docs/README.md">📚 文档中心</a> ·
  <a href="#-快速开始">🚀 快速开始</a> ·
  <a href="#-核心特性">✨ 核心特性</a> ·
  <a href="#-部署架构">🏗️ 部署架构</a> ·
  <a href="#-技术栈">🛠️ 技术栈</a>
</p>

<p>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg" alt="Platform">
</p>

</div>

---

## 🎯 产品定位

把 SafeGate 放在你的统一入口网关之后，它会在请求到达上游业务站点之前完成风控判定、字段转换与日志记录：

```
用户请求
    │
    ▼
Nginx / 1Panel / Caddy（SSL、域名解析）
    │
    ├─ admin.example.com ──▶ SafeGate Admin（管理后台 + API）
    │
    └─ api.example.com ────▶ SafeGate Proxy（风控、转发、审计）
                                │
                                ▼
                         目标站点 / Upstream
```

无论是保护注册接口不被刷单、代理外部站点时自动转换 JSON 字段，还是集中审计所有访问流量，SafeGate 都能以低侵入的方式接入现有架构。

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| 🌐 **域名映射** | 将访问域名绑定到上游目标地址，支持默认站点兜底。 |
| 🔍 **真实 IP 透传** | 自动从 `CF-Connecting-IP`、`X-Forwarded-For`、`X-Real-IP` 等头中提取真实 IP 并转发给上游。 |
| 🛡️ **接口风控引擎** | 支持 `duplicate_ip`（成功后计数）与 `rate_limit`（请求即计数），可按 IP 或 IP+业务字段组合身份。 |
| 📝 **JSON 字段映射** | 使用 `gjson/sjson` 路径语法，在转发前自动重命名、重组请求体字段。 |
| 🔄 **响应智能改写** | `none` / `headers` / `full` 三种模式，处理重定向、Cookie 域与 HTML 正文链接。 |
| 📊 **可视化审计** | 首页统计看板、全量访问日志、被拦截日志详情，支持查看 Query、Headers、Body。 |
| 🖥️ **响应式管理后台** | React + shadcn/ui 构建的现代化控制台，桌面与移动端自适应。 |
| 🐳 **一键部署** | Docker Compose 编排 PostgreSQL、Redis 与后端服务，开箱即用。 |

## 🚀 快速开始

### 环境要求

- Docker Engine >= 24
- Docker Compose >= 2

### 1. 启动服务

```bash
cp .env.example .env
# 编辑 .env，设置 ADMIN_PASSWORD 和 JWT_SECRET
docker compose up -d --build
```

### 2. 获取初始密码

如果未设置 `ADMIN_PASSWORD`，系统会生成随机密码并打印到日志：

```bash
docker compose logs -f backend
```

默认管理员账号：`admin`。

### 3. 访问控制台

| 入口 | 地址 |
|------|------|
| 管理后台 | `http://localhost:18081` |
| 后端 API | `http://127.0.0.1:18081/api/admin` |
| 代理入口 | `http://127.0.0.1:18080` |

## 🏗️ 部署架构

生产环境推荐将 SafeGate 部署在 1Panel / Nginx 身后：

```
用户 → 1Panel Nginx（443 SSL）
        │
        ├─ admin.yourdomain.com → 127.0.0.1:18081
        │
        └─ api.yourdomain.com   → 127.0.0.1:18080
                                    │
                                    ▼
                                目标站点 / Upstream
```

- 设置 `MODE=api`，由 Nginx/1Panel 托管前端或继续使用 `MODE=all`。
- 后端监听 `127.0.0.1`，避免直接暴露在公网。
- 管理后台域名限制来源 IP 或仅内网使用。

详细部署指南见 [docs/deployment.md](./docs/deployment.md)。

## 🛠️ 技术栈

**后端**

- Go 1.25 + Gin 1.12
- GORM + PostgreSQL 16
- Redis 7 + go-redis 9
- JWT（golang-jwt/jwt/v5）+ bcrypt
- `net/http/httputil.ReverseProxy`
- tidwall/gjson + tidwall/sjson

**前端**

- React 19 + TypeScript 6
- Vite 8
- Tailwind CSS 3 + shadcn/ui
- React Router DOM 7
- Axios

**部署**

- Docker + Docker Compose
- 支持 1Panel / Nginx / Caddy 反代

## 📚 文档

- [docs/README.md](./docs/README.md) — 文档索引
- [docs/architecture.md](./docs/architecture.md) — 系统架构
- [docs/api.md](./docs/api.md) — Admin REST API
- [docs/database.md](./docs/database.md) — 数据库设计
- [docs/proxy.md](./docs/proxy.md) — 反向代理与风控
- [docs/frontend.md](./docs/frontend.md) — 前端设计
- [docs/deployment.md](./docs/deployment.md) — 部署指南
- [docs/security.md](./docs/security.md) — 安全设计

## 💻 本地开发

```bash
# 启动完整服务栈
docker compose up -d

# 或单独启动前端开发服务器
cd web
cp .env.example .env.local
npm install
npm run dev
```

后端本地开发说明见 [docs/deployment.md](./docs/deployment.md)。

## 🤝 参与贡献

欢迎通过 Issue 和 Pull Request 参与项目改进。修改代码时，请同步更新相关文档。

## 📄 License

[MIT](./LICENSE)
