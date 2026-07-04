# SafeGate 技术文档

本文档集合面向开发者、运维人员和希望深入理解 SafeGate 的读者，覆盖架构设计、接口规范、数据库模型、反向代理与风控逻辑、前端实现、部署方式以及安全机制。

## 文档目录

| 文档 | 目标读者 | 内容 |
|------|----------|------|
| [architecture.md](./architecture.md) | 开发者、架构师 | 总体架构、技术栈、目录结构、请求流转、设计原则 |
| [api.md](./api.md) | 前端开发者、第三方集成者 | Admin REST API 接口说明、认证方式、请求/响应示例 |
| [database.md](./database.md) | 后端开发者、DBA | PostgreSQL 表结构、字段说明、索引、Redis 键设计 |
| [proxy.md](./proxy.md) | 后端开发者、运维 | 反向代理流程、真实 IP 提取、默认站点、响应改写、风控规则引擎、JSON 字段映射 |
| [frontend.md](./frontend.md) | 前端开发者 | React 前端技术栈、目录结构、页面说明、路由与状态管理 |
| [deployment.md](./deployment.md) | 运维、部署人员 | Docker Compose 部署、环境变量、1Panel/Nginx 反代、升级与维护 |
| [security.md](./security.md) | 安全人员、开发者 | 认证、JWT、权限、输入校验、风控隔离、敏感数据处理 |

## 快速开始

```bash
cp .env.example .env
# 编辑 .env，修改 ADMIN_PASSWORD 和 JWT_SECRET
docker compose up -d --build
```

启动后：

- 管理后台：`http://localhost:18081`
- 后端 API：`http://127.0.0.1:18081/api/admin`
- 代理入口：`http://127.0.0.1:18080`

详细的部署说明与生产环境建议见 [deployment.md](./deployment.md)。

## 项目概览

SafeGate 是一个单体 Go 服务，同时运行两个 HTTP Server：

1. **Proxy Server**（`PORT`，默认 `8080`）：对外提供反向代理能力，按请求 `Host` 头匹配域名映射。
2. **Admin Server**（`ADMIN_PORT`，默认 `8081`）：提供管理后台 REST API，并在 `MODE=all` 时托管 React 前端静态页面。

数据持久化使用 PostgreSQL；风控计数持久化在 PostgreSQL 中，Redis 用于运行时计数缓存和 JWT 黑名单。前端使用 React 19 + TypeScript + Vite + Tailwind CSS + shadcn/ui 构建。

## 术语表

| 术语 | 说明 |
|------|------|
| 绑定域名（Bind Domain） | 用户实际访问 SafeGate 时使用的域名，如 `api.example.com`。 |
| 目标地址（Target URL） | SafeGate 把请求转发到的上游地址，如 `https://upstream.example.com`。 |
| 真实 IP | 经过 `X-Forwarded-For`、`CF-Connecting-IP` 等头解析后的原始客户端 IP。 |
| 身份标识（Identity） | 风控计数使用的 key 组成部分，通常是 `IP` 或 `IP|field1=value1|field2=value2`。 |
| 默认站点 | `domains.is_default = true` 的映射，当请求 `Host` 未精确匹配时作为兜底。 |
| 响应改写模式 | `none` / `headers` / `full`，控制代理如何改写上游返回的内容。 |

## 贡献与维护

- 代码修改后，请同步更新本文档集合中对应的内容。
- 新增 API、页面或配置项时，请至少更新 `api.md`、`frontend.md`、`database.md` 或 `deployment.md`。
