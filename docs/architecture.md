# 系统架构

## 总体设计

本项目是一个可配置的反向代理控制台，采用**前后端分离**架构：

- **后端**：单体式 Gin 服务，同时承载：
  - 管理后台 REST API（`/api/admin/*`）
  - 实际的反向代理流量入口（按 `Host` 匹配域名映射）
- **前端**：React + TypeScript + Vite + shadcn/ui 的 SPA，通过 Axios 调用后端 API。
- **持久化**：PostgreSQL 存储配置与日志。
- **缓存/计数**：Redis 存储 JWT 黑名单、风控接口计数。
- **部署**：Docker Compose 统一编排后端、前端、PostgreSQL、Redis。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Gin (Go) |
| ORM/数据库 | GORM + PostgreSQL |
| 缓存 | Redis（go-redis） |
| 认证 | JWT（golang-jwt/jwt/v5） + bcrypt |
| 反向代理 | `net/http/httputil.ReverseProxy` |
| JSON 映射 | tidwall/gjson + tidwall/sjson |
| 前端框架 | React 18 + TypeScript |
| 前端构建 | Vite |
| 前端样式 | Tailwind CSS |
| 前端组件 | shadcn/ui |
| 路由 | React Router |
| HTTP 客户端 | Axios |
| 部署 | Docker + Docker Compose |

## 请求流转

```
用户请求
   │
   ▼
Nginx / 1Panel（80/443）
   │
   ├─ admin.yourdomain.com ──▶ SafeGate Admin Server（监听 ADMIN_PORT）
   │                              └── 静态前端 / Admin API
   │
   └─ 其他绑定域名 ───────────▶ SafeGate Proxy Server（监听 PORT）
                                   │
                                   ├─ 查找 domains 表
                                   │     │
                                   │     ├─ 命中：风控检查 → JSON 字段映射 → 反向代理到 target
                                   │     └─ 未命中：404
```

## 目录结构

```
ip_check/
├── cmd/server/main.go          # 后端入口
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── db/                     # 数据库连接与迁移
│   ├── models/                 # GORM 模型
│   ├── repository/             # 数据访问层
│   ├── service/                # 业务逻辑
│   ├── handler/                # HTTP handler
│   └── middleware/             # Gin 中间件
├── web/                        # React 前端
├── docker-compose.yml
├── Dockerfile                  # 后端 Dockerfile
└── docs/                       # 设计文档
```

## 设计原则

1. **简单优先**：单后端同时承载 API 与代理，降低部署复杂度。
2. **可扩展**：未来可将代理层拆分为独立服务，Redis 计数天然适合分布式。
3. **无状态**：后端不保存会话状态，JWT + Redis 黑名单实现登出。
4. **云原生**：所有依赖容器化，支持 `docker compose up` 一键启动。
