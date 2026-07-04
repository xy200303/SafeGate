# 系统架构

## 总体设计

SafeGate 是一个可配置的 **IP 风控网关 / 反向代理防火墙**，采用前后端分离但后端同体部署的架构：

- **后端**：单体 Gin 服务，同时承载两个 HTTP Server：
  - **Proxy Server**（`PORT`）：反向代理入口，按 `Host` 头匹配域名映射。
  - **Admin Server**（`ADMIN_PORT`）：管理后台 REST API（`/api/admin/*`），`MODE=all` 时还负责托管前端构建产物。
- **前端**：React + TypeScript + Vite + shadcn/ui 的单页应用，通过 Axios 调用 Admin API。
- **持久化**：PostgreSQL 存储管理员、域名映射、风控规则、风控计数、代理日志。
- **缓存/会话**：Redis 缓存风控计数，并存储 JWT 黑名单。
- **部署**：Docker Compose 编排 PostgreSQL、Redis、后端服务；生产环境可配合 1Panel / Nginx / Caddy 作为统一入口。

## 技术栈

| 层级 | 技术 | 版本（截至当前代码） |
|------|------|---------------------|
| 后端语言 | Go | 1.25+（go.mod 声明 1.25，Docker 构建使用 1.26） |
| Web 框架 | Gin | 1.12 |
| ORM | GORM | 1.31 |
| 数据库 | PostgreSQL | 16 |
| 缓存 | Redis | 7 |
| Redis 客户端 | go-redis | 9 |
| 认证 | golang-jwt/jwt/v5 + bcrypt | - |
| 反向代理 | `net/http/httputil.ReverseProxy` | - |
| JSON 路径 | tidwall/gjson + tidwall/sjson | - |
| 前端框架 | React | 19.2 |
| 前端语言 | TypeScript | 6.0 |
| 构建工具 | Vite | 8.1 |
| 样式 | Tailwind CSS | 3.4 |
| 组件库 | shadcn/ui | - |
| 路由 | React Router DOM | 7 |
| HTTP 客户端 | Axios | 1.18 |
| 部署 | Docker + Docker Compose | - |

## 服务架构

```
┌─────────────────────────────────────────────────────────────┐
│                        外部用户                              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │  Nginx / 1Panel / Caddy │  SSL、域名解析、静态资源
              │   80 / 443             │
              └──────────┬─────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
   admin.example.com  api.example.com  其他绑定域名
          │              │              │
          │              ▼              │
          │    ┌───────────────────┐    │
          │    │  SafeGate Proxy   │    │
          │    │  Server (PORT)    │    │
          │    │  8080 / 18080     │    │
          │    └─────────┬─────────┘    │
          │              │              │
          │              ▼              │
          │    ┌───────────────────┐    │
          │    │  目标站点 / Upstream │   │
          │    └───────────────────┘    │
          │                             │
          ▼                             │
   ┌───────────────────┐                │
   │  SafeGate Admin   │                │
   │  Server           │                │
   │  ADMIN_PORT       │                │
   │  8081 / 18081     │                │
   └───────────────────┘                │
          │                             │
          ▼                             │
   ┌───────────────────┐                │
   │  PostgreSQL       │                │
   │  Redis            │                │
   └───────────────────┘                │
```

## 请求流转

### 管理后台请求

```
浏览器 / HTTP 客户端
        │
        ▼
   Nginx / 1Panel
        │
        ▼
   SafeGate Admin Server（ADMIN_PORT）
        │
        ├─ /api/admin/*  → Gin 路由 → Handler → Service → Repository → PostgreSQL/Redis
        │
        └─ 其他路径       → 静态文件服务（MODE=all 时返回 web/dist 内容）
```

### 代理请求

```
用户请求
    │
    ▼
Nginx / 1Panel（保留 Host、X-Real-IP、X-Forwarded-For）
    │
    ▼
SafeGate Proxy Server（PORT）
    │
    ├─ 按 Host 精确匹配 domains.bind_domain
    │     ├─ 命中 → 继续后续处理
    │     └─ 未命中 → 查找 is_default = true 的默认站点
    │           ├─ 命中 → 继续后续处理（标记为默认站点命中）
    │           └─ 未命中 → 返回 404
    │
    ├─ 提取真实 Client IP
    │
    ├─ 加载该域名下的风控规则并匹配
    │     ├─ 命中阈值 → 返回拦截响应（JSON 或 HTML 防火墙页）
    │     └─ 未命中 → 继续转发
    │
    ├─ 应用请求体 JSON 字段映射（request_transform）
    │
    ├─ 设置转发头（X-Real-IP、X-Forwarded-For 等）
    │
    ├─ ReverseProxy 转发到 target_url
    │
    ├─ 接收上游响应，按 rewrite_mode 改写响应
    │
    ├─ duplicate_ip 规则在上游响应满足成功判定后增加计数
    │
    └─ 异步写入 proxy_logs
```

## 后端目录结构

```
ip_check/
├── cmd/
│   └── server/
│       └── main.go              # 服务入口：加载配置、初始化依赖、启动双 Server
├── internal/
│   ├── config/
│   │   └── config.go            # 环境变量读取与默认值
│   ├── db/
│   │   └── db.go                # PostgreSQL 连接、GORM AutoMigrate
│   ├── handler/
│   │   └── handler.go           # Admin API Handler + Proxy Handler
│   ├── middleware/
│   │   ├── auth.go              # JWT 校验中间件
│   │   └── cors.go              # CORS 中间件
│   ├── models/
│   │   └── models.go            # GORM 模型：User、Domain、Rule、ProxyLog、FirewallAttempt
│   ├── redis/
│   │   └── redis.go             # Redis 客户端初始化
│   ├── repository/
│   │   └── repository.go        # 数据访问层
│   └── service/
│       └── service.go           # 业务逻辑：Auth、Domain、Rule、Proxy、Log、Stats
├── web/                         # React 前端源码
├── go.mod / go.sum
├── docker-compose.yml
├── Dockerfile
└── docs/                        # 本文档集合
```

## 前端目录结构

```
web/
├── public/                      # 静态资源（favicon、图标）
├── src/
│   ├── api/
│   │   ├── admin.ts             # Admin API 封装
│   │   └── client.ts            # Axios 实例、拦截器
│   ├── components/
│   │   ├── layout/
│   │   │   └── AdminLayout.tsx  # 响应式管理后台布局
│   │   └── ui/                  # shadcn/ui 组件
│   ├── hooks/
│   │   └── useAuth.ts           # localStorage token 读写与同步
│   ├── lib/
│   │   └── utils.ts             # cn() 等工具函数
│   ├── pages/
│   │   ├── Login.tsx            # 登录页
│   │   ├── Stats.tsx            # 拦截统计看板
│   │   ├── Domains.tsx          # 域名映射管理
│   │   ├── Rules.tsx            # 接口风控规则管理
│   │   ├── Logs.tsx             # 全量访问日志
│   │   └── BlockedLogs.tsx      # 被拦截日志详情
│   ├── router/
│   │   └── index.tsx            # 路由配置与登录守卫
│   ├── App.tsx
│   ├── index.css                # Tailwind 入口 + CSS 变量主题
│   └── main.tsx
├── index.html
├── package.json
├── tailwind.config.js
├── tsconfig.json
└── vite.config.ts
```

## 模块职责

| 模块 | 职责 |
|------|------|
| `config` | 从环境变量（支持 `.env` 文件）读取配置，提供默认值。 |
| `db` | 建立 PostgreSQL 连接，启动时执行 AutoMigrate。 |
| `models` | 定义数据模型、JSONB 类型以及响应聚合类型。 |
| `repository` | 封装对 PostgreSQL 的 CRUD 和复杂查询（如统计聚合）。 |
| `service` | 实现业务规则：登录认证、JWT 管理、域名/规则 CRUD、真实 IP 提取、风控判定、反向代理构造、日志写入。 |
| `handler` | 绑定 HTTP 路由，处理请求参数绑定、调用 service、返回统一响应格式。 |
| `middleware` | 提供可复用的 Gin 中间件（JWT 认证、CORS）。 |
| `redis` | 初始化 Redis 连接，供 service 缓存风控计数和维护 JWT 黑名单使用。 |

## 统一响应格式

Admin API 使用统一 JSON 响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": { }
}
```

- `code == 0` 表示成功。
- `code != 0` 表示业务错误，具体值和 `message` 由接口定义。
- 认证失败时返回 HTTP 401；参数错误返回 HTTP 400；服务器错误返回 HTTP 500。

## 设计原则

1. **简单优先**：单后端二进制同时承载 API 与代理，降低部署和运维复杂度。
2. **无状态**：后端不保存会话状态，JWT + Redis 黑名单实现登录态和登出。
3. **可扩展**：
   - 风控计数以 PostgreSQL 为准，Redis 作为共享缓存；未来水平扩展时共享 Redis 和 PostgreSQL 即可。
   - 代理层未来可拆分为独立服务，Admin Server 仅保留 API。
4. **云原生**：所有依赖容器化，支持 `docker compose up -d --build` 一键启动。
5. **可观测**：所有代理请求和拦截事件写入 `proxy_logs`，并提供可视化统计。
