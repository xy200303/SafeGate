# SafeGate 技术文档

本项目是一个可配置的反向代理控制台，支持管理员配置域名映射、真实 IP 透传、接口风控拦截与自定义 JSON 数据转发。

## 文档目录

| 文档 | 内容 |
|------|------|
| [architecture.md](./architecture.md) | 总体架构、技术栈、目录结构、请求流转 |
| [api.md](./api.md) | Admin REST API 接口说明 |
| [database.md](./database.md) | PostgreSQL 表结构与 Redis 键设计 |
| [proxy.md](./proxy.md) | 反向代理、真实 IP、风控规则、JSON 字段映射 |
| [frontend.md](./frontend.md) | React 前端设计、页面、响应式布局 |
| [deployment.md](./deployment.md) | Docker Compose 部署与环境变量 |
| [security.md](./security.md) | 认证、JWT、权限、输入校验 |

## 快速开始

```bash
cp .env.example .env
docker compose up -d --build
```

详见 [deployment.md](./deployment.md)。
