# SafeGate 部署文档

## 环境要求

- Docker Engine >= 24
- Docker Compose >= 2

## 快速启动

1. 复制环境变量模板：

```bash
cp .env.example .env
```

2. 按需修改 `.env` 中的密码与密钥。

3. 构建并启动：

```bash
docker compose up -d --build
```

4. 查看后端日志获取初始管理员密码（若未设置 `ADMIN_PASSWORD`）：

```bash
docker compose logs -f backend
```

5. 本地访问：

- 管理后台：`http://localhost:18081`（后端 `MODE=all` 直接挂载 `web/dist`）
- 后端 API：`http://127.0.0.1:18081/api/admin`
- 代理测试：`curl http://127.0.0.1:18080/post`

> 本地 Docker 默认使用 `MODE=all`，前端静态页面由后端 admin 端口直接提供，无需单独 frontend 容器。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `HOST` | 后端监听地址，留空表示监听所有网卡 | - |
| `PORT` | 代理入口监听端口 | `8080` |
| `ADMIN_PORT` | 管理后台 API 监听端口 | `8081` |
| `MODE` | 启动模式：`api` 仅 API / `all` 额外挂载前端静态页面 | `all` |
| `WEB_DIST` | 前端静态资源目录，`MODE=all` 时生效 | `web/dist` |
| `ADMIN_USERNAME` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码（未设置则随机生成） | - |
| `DATABASE_URL` | PostgreSQL DSN | `postgres://safegate:safegate@postgres:5432/safegate?sslmode=disable` |
| `REDIS_ADDR` | Redis 地址 | `redis:6379` |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `JWT_SECRET` | JWT 签名密钥 | 随机生成 |
| `JWT_EXPIRE_HOURS` | JWT 有效期（小时） | `24` |
| `CORS_ORIGIN` | 前端跨域来源 | `*` |

## 服务说明

### postgres

- 镜像：`postgres:16-alpine`
- 持久化卷：`pg_data`

### redis

- 镜像：`redis:7.4.1-alpine`
- 持久化卷：`redis_data`

### backend

- 多阶段构建：先构建 React 前端产物，再编译 Go 二进制，最终复制到 alpine 镜像。
- 自动执行 GORM 迁移。
- 启动两个 HTTP 服务：
  - `PORT`：代理入口
  - `ADMIN_PORT`：管理后台 API
- `MODE=all` 时，在 `ADMIN_PORT` 上挂载 `web/dist` 静态页面，直接访问 admin 端口即可打开管理后台。

## 生产环境：使用 1Panel / 自有 Nginx 作为入口

生产环境建议设置 `MODE=api`，让 1Panel（或你自己维护的 Nginx/Caddy）作为统一入口，反向代理到 SafeGate 后端。前端静态页面可由 1Panel 直接托管，也可以继续使用 `MODE=all` 让后端自带。

架构：

```
用户 → 1Panel Nginx（80/443、SSL、域名解析）
        ↓
    SafeGate backend
      ├── 管理后台 API：127.0.0.1:18081
      └── 代理入口：127.0.0.1:18080
        ↓
    目标站点 / Upstream
```

### 1Panel 配置示例

1. 管理后台域名反代到 admin 端口：
   - 域名：`admin.yourdomain.com`
   - 代理地址：`http://127.0.0.1:18081`
   - 开启“保留 Host 头”/“传递真实 IP”

2. 为每个需要代理的绑定域名创建反代网站，指向代理端口：
   - 域名：`api.yourdomain.com`
   - 代理地址：`http://127.0.0.1:18080`
   - 同样保留 Host 头和真实 IP

3. 后端环境变量：
   - `HOST=127.0.0.1`
   - `PORT=18080`
   - `ADMIN_PORT=18081`

### Nginx 配置示例

```nginx
upstream safegate_admin {
    server 127.0.0.1:18081;
}

upstream safegate_proxy {
    server 127.0.0.1:18080;
}

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

### 安全建议

- 管理后台域名建议限制访问来源 IP，或只在内网使用。
- 后端监听端口建议只绑定 `127.0.0.1`，避免直接暴露在公网。
- 通过 1Panel 申请 SSL 证书，强制 HTTPS。

## 域名解析

- 将管理后台域名指向宿主机 IP。
- 将各绑定域名（需要代理的域名）指向宿主机 IP。

## 升级

```bash
docker compose pull
docker compose up -d --build
```
