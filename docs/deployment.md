# SafeGate 部署文档

## 目录

- [环境要求](#环境要求)
- [快速启动](#快速启动)
- [环境变量](#环境变量)
- [服务说明](#服务说明)
- [生产环境部署](#生产环境部署)
- [域名解析](#域名解析)
- [升级与维护](#升级与维护)
- [常见问题](#常见问题)

## 环境要求

- Docker Engine >= 24
- Docker Compose >= 2

如使用 1Panel 等面板，请确保已安装 Docker 插件并开放所需端口。

## 快速启动

### 1. 复制环境变量模板

```bash
cp .env.example .env
```

### 2. 修改 `.env`

至少修改以下两项：

```env
ADMIN_PASSWORD=your-strong-admin-password
JWT_SECRET=your-random-jwt-secret-at-least-32-chars
```

### 3. 构建并启动

```bash
docker compose up -d --build
```

### 4. 查看初始密码

如果未设置 `ADMIN_PASSWORD`，首次启动会生成随机密码并打印到日志：

```bash
docker compose logs -f backend
```

默认管理员账号为 `admin`。

### 5. 本地访问

| 入口 | 地址 | 说明 |
|------|------|------|
| 管理后台 | `http://localhost:18081` | `MODE=all` 时后端直接挂载 `web/dist` |
| 后端 API | `http://127.0.0.1:18081/api/admin` | Admin REST API |
| 代理测试 | `curl http://127.0.0.1:18080/post` | 反向代理入口 |

> 本地 Docker 默认使用 `MODE=all`，前端静态页面由后端 admin 端口直接提供，无需单独 frontend 容器。

## 环境变量

### 后端环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `HOST` | 后端监听地址，留空表示监听所有网卡 | - |
| `PORT` | 代理入口监听端口 | `8080` |
| `ADMIN_PORT` | 管理后台 API 监听端口 | `8081` |
| `MODE` | 启动模式：`api` 仅 API+代理；`all` 额外挂载前端静态页面 | `all` |
| `WEB_DIST` | 前端静态资源目录，`MODE=all` 时生效 | `web/dist` |
| `ADMIN_USERNAME` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码（未设置则随机生成） | - |
| `DATABASE_URL` | PostgreSQL DSN | `postgres://safegate:safegate@postgres:5432/safegate?sslmode=disable` |
| `REDIS_ADDR` | Redis 地址 | `redis:6379` |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `JWT_SECRET` | JWT 签名密钥 | 随机生成 |
| `JWT_EXPIRE_HOURS` | JWT 有效期（小时） | `24` |
| `CORS_ORIGIN` | 前端跨域来源 | `*` |

### 前端环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VITE_API_BASE_URL` | 前端 API 基础路径 | `/api` |

### 构建环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `GOPROXY` | Docker 构建阶段执行 `go mod download` 使用的 Go 模块代理，国内环境建议使用默认值；海外或私有代理环境可改为 `https://proxy.golang.org,direct` 或内部代理地址 | `https://goproxy.cn,direct` |

### 安全建议

- 生产环境必须设置 `ADMIN_PASSWORD` 和 `JWT_SECRET`，不要使用 `.env.example` 中的默认值。
- `JWT_SECRET` 建议使用长度不低于 32 个字符的随机字符串。
- 后端监听地址建议设置为 `127.0.0.1`，通过 Nginx/1Panel 反向代理暴露。
- 管理后台域名建议限制访问来源 IP 或仅内网使用。

## 服务说明

### postgres

- 镜像：`postgres:16-alpine`
- 容器名：`safegate_postgres`
- 持久化卷：`pg_data`
- 环境变量：`POSTGRES_USER=safegate`、`POSTGRES_PASSWORD=safegate`、`POSTGRES_DB=safegate`
- 健康检查：`pg_isready -U safegate -d safegate`

### redis

- 镜像：`redis:7.4.1-alpine`
- 容器名：`safegate_redis`
- 持久化卷：`redis_data`
- 健康检查：`redis-cli ping`

### backend

- 镜像：`safegate/backend:latest`（本地构建）
- 容器名：`safegate_backend`
- 多阶段构建：
  1. `web-builder`：Node 22 环境构建 React 前端产物到 `web/dist`。
  2. `builder`：Go 1.26 环境编译 Go 二进制。
  3. 运行时：Alpine 3.21 最小镜像，仅包含 `ca-certificates`、编译后的二进制和 `web/dist`。
- 自动执行 GORM 迁移。
- 启动两个 HTTP 服务：
  - `PORT`：代理入口。
  - `ADMIN_PORT`：管理后台 API。
- `MODE=all` 时，在 `ADMIN_PORT` 上挂载 `web/dist` 静态页面。
- 依赖 `postgres` 和 `redis` 健康检查通过后启动。

## 生产环境部署

生产环境推荐设置 `MODE=api`，让 1Panel（或 Nginx/Caddy）作为统一入口，反向代理到 SafeGate 后端。前端静态页面可由 1Panel 直接托管，也可以继续使用 `MODE=all` 让后端自带。

### 推荐架构

```
用户 → 1Panel Nginx（443 SSL、域名解析）
        │
        ├─ admin.yourdomain.com → 127.0.0.1:18081（管理后台）
        │
        └─ api.yourdomain.com   → 127.0.0.1:18080（代理入口）
                                    ↓
                                目标站点 / Upstream
```

### 后端环境变量示例

```env
HOST=127.0.0.1
PORT=18080
ADMIN_PORT=18081
MODE=api
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<强密码>
JWT_SECRET=<随机密钥>
DATABASE_URL=postgres://safegate:safegate@postgres:5432/safegate?sslmode=disable
REDIS_ADDR=redis:6379
```

### 1Panel 配置示例

#### 管理后台站点

- 域名：`admin.yourdomain.com`
- 代理地址：`http://127.0.0.1:18081`
- 开启“保留 Host 头”/“传递真实 IP”
- 申请 SSL 证书并强制 HTTPS

#### 代理入口站点

为每个需要代理的绑定域名创建反代网站：

- 域名：`api.yourdomain.com`
- 代理地址：`http://127.0.0.1:18080`
- 同样保留 Host 头和真实 IP
- 申请 SSL 证书并强制 HTTPS

### Nginx 配置示例

```nginx
upstream safegate_admin {
    server 127.0.0.1:18081;
}

upstream safegate_proxy {
    server 127.0.0.1:18080;
}

# 管理后台
server {
    listen 80;
    server_name admin.yourdomain.com;

    location / {
        proxy_pass http://safegate_admin;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# 代理入口
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://safegate_proxy;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

> 生产环境请通过 Certbot 或 1Panel 申请证书后启用 `listen 443 ssl`。

### 安全加固建议

- 管理后台域名限制访问来源 IP 段，或仅在内网可达。
- 后端监听端口只绑定 `127.0.0.1`，避免直接暴露在公网。
- 通过 Nginx/1Panel 申请 SSL 证书，强制 HTTPS。
- 定期修改管理员密码。
- 为 PostgreSQL 和 Redis 设置强密码（当前 docker-compose 中为默认密码，生产请修改）。
- 建议为容器配置日志大小限制，避免日志无限增长。

### 可选：优化 docker-compose

生产环境可考虑在 `docker-compose.yml` 中为服务增加重启策略和日志限制：

```yaml
services:
  backend:
    restart: unless-stopped
    logging:
      driver: json-file
      options:
        max-size: "50m"
        max-file: "3"
  postgres:
    restart: unless-stopped
    logging:
      driver: json-file
      options:
        max-size: "50m"
        max-file: "3"
  redis:
    restart: unless-stopped
    logging:
      driver: json-file
      options:
        max-size: "20m"
        max-file: "3"
```

> 修改 `docker-compose.yml` 后请注意不要覆盖仓库中的默认配置，可通过 `docker-compose.override.yml` 扩展。

## 域名解析

- 管理后台域名指向宿主机公网 IP。
- 各绑定域名（需要代理的域名）指向宿主机公网 IP。
- 如果测试环境使用本地 hosts，可在 `C:\Windows\System32\drivers\etc\hosts` 或 `/etc/hosts` 中添加：

```
127.0.0.1 admin.local.test
127.0.0.1 api.local.test
```

## 升级与维护

### 升级版本

```bash
docker compose pull
docker compose up -d --build
```

### 查看日志

```bash
# 后端日志
docker compose logs -f backend

# 全部服务日志
docker compose logs -f

# 查看最近 100 行
docker compose logs --tail=100 backend
```

### 备份数据

```bash
# 备份 PostgreSQL
docker exec safegate_postgres pg_dump -U safegate safegate > safegate_$(date +%F).sql

# 备份 Redis（若开启持久化）
docker cp safegate_redis:/data/dump.rdb ./dump.rdb
```

### 重置管理员密码

如果忘记了管理员密码，可以临时修改 `.env` 中的 `ADMIN_PASSWORD` 并清空 `users` 表让系统重新创建：

```bash
# 进入 PostgreSQL 容器
docker exec -it safegate_postgres psql -U safegate -d safegate

# 清空用户表（会重新生成管理员）
TRUNCATE TABLE users;
\q

# 重启后端
docker compose restart backend
```

> 注意：此操作会删除现有管理员账号，生产环境请谨慎。

### 停止服务

```bash
docker compose down
```

如需同时删除数据卷（会丢失所有数据）：

```bash
docker compose down -v
```

## 常见问题

### 启动时报数据库连接失败

检查 `DATABASE_URL` 中的主机名是否正确：

- Docker Compose 环境使用 `postgres`。
- 本地开发使用 `127.0.0.1:5434`（docker-compose 中映射的宿主机端口）。

### 前端请求 401

- 检查 `localStorage` 中是否有 `token`。
- 检查 `JWT_SECRET` 是否被修改（修改后之前的 token 会失效）。
- 检查后端时间是否准确。

### 代理请求返回 404

- 检查 `domains` 表中是否有匹配的 `bind_domain`。
- 检查 Nginx/1Panel 是否正确传递了 `Host` 头。
- 如果使用了默认站点，检查是否已设置 `is_default = true`。

### 真实 IP 不正确

- 检查 `real_ip_headers` 配置顺序是否正确。
- 检查 Nginx/1Panel 是否正确传递了 `X-Forwarded-For` 或 `CF-Connecting-IP`。
- 如果 SafeGate 直接暴露在公网（无反向代理），真实 IP 会取自 `RemoteAddr`。
