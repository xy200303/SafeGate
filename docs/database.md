# 数据库设计

使用 PostgreSQL，GORM 自动迁移。

## 数据表

### users

管理员账号。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 自增主键 |
| username | VARCHAR(64) UNIQUE | 用户名 |
| password_hash | VARCHAR(255) | bcrypt 哈希 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

### domains

域名映射表。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 自增主键 |
| bind_domain | VARCHAR(255) UNIQUE | 用户访问的绑定域名 |
| target_url | VARCHAR(512) | 目标地址，如 `https://target.example.com` |
| real_ip_headers | VARCHAR(512) | 真实 IP 来源头，逗号分隔 |
| forward_ip_header | VARCHAR(128) | 转发给上游的 IP 头 |
| request_transform | JSONB | 请求体字段映射规则 |
| response_transform | JSONB | 响应体字段映射规则（预留） |
| rewrite_host | BOOLEAN DEFAULT true | 是否将请求 Host 改写为目标域名 |
| rewrite_mode | VARCHAR(32) DEFAULT 'full' | 响应改写模式：`none` / `headers` / `full` |
| is_default | BOOLEAN DEFAULT false | 是否为默认站点（类似 Nginx default_server） |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

### rules

接口风控规则。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 自增主键 |
| domain_id | BIGINT FK | 所属域名 |
| name | VARCHAR(128) | 规则名称 |
| path_prefix | VARCHAR(255) | 路径前缀，如 `/api/register` |
| methods | VARCHAR(128) | HTTP 方法，逗号分隔，`ALL` 表示全部 |
| rule_type | VARCHAR(32) | `duplicate_ip` / `rate_limit` |
| identity_fields | VARCHAR(512) | 身份字段，逗号分隔，如 `phone,email` |
| max_attempts | INT | 最大允许次数 |
| window_seconds | INT | 计数窗口（秒），0 表示永久 |
| block_seconds | INT | 拦截后封禁时长（秒），0 表示仅本次 |
| block_status | INT | 拦截返回的 HTTP 状态码 |
| block_response | JSONB | 拦截返回的响应体 |
| enabled | BOOLEAN | 是否启用 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

### proxy_logs

代理访问日志。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 自增主键 |
| bind_domain | VARCHAR(255) | 绑定域名 |
| client_ip | VARCHAR(64) | 客户端真实 IP |
| method | VARCHAR(16) | HTTP 方法 |
| path | VARCHAR(2048) | 请求路径 |
| target_url | VARCHAR(512) | 目标地址 |
| status_code | INT | 上游返回状态码 |
| blocked | BOOLEAN | 是否被风控拦截 |
| rule_id | BIGINT NULL | 命中规则 ID |
| message | VARCHAR(512) | 说明 |
| created_at | TIMESTAMPTZ | 创建时间 |

## Redis 键设计

| 键 | 类型 | 说明 |
|----|------|------|
| `attempt:<rule_id>:<identity>` | String | 风控计数，带 TTL |
| `jwt:blacklist:<jti>` | String | JWT 黑名单，TTL 为 token 剩余有效期 |

## 关系图

```
users
  │
domains 1───* rules
  │
  └───────* proxy_logs
```
