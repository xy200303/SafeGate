# Admin API 文档

所有 Admin API 路径前缀为 `/api/admin`。管理后台 React 前端通过 `VITE_API_BASE_URL` 拼接出完整地址，Docker 环境下默认访问 `/api/admin/*`。

## 目录

- [认证机制](#认证机制)
- [通用响应格式](#通用响应格式)
- [认证接口](#认证接口)
- [域名映射接口](#域名映射接口)
- [风控规则接口](#风控规则接口)
- [访问日志接口](#访问日志接口)
- [错误码说明](#错误码说明)

## 认证机制

- 登录成功后返回 `access_token`（JWT）。
- 前端将 token 保存到 `localStorage`，后续请求在 `Authorization: Bearer <token>` 头中携带。
- 登出时调用 `POST /api/admin/logout`，后端将 token 的 `jti` 加入 Redis 黑名单直到 token 过期。
- 所有 `/api/admin/*` 接口（除登录外）都需要 JWT 认证。

## 通用响应格式

```json
{
  "code": 0,
  "message": "ok",
  "data": { }
}
```

- 成功时 `code == 0`。
- 失败时 `code != 0`，`message` 描述错误原因。
- 认证失败会返回 HTTP 401，参数校验失败返回 HTTP 400。

## 认证接口

### POST /api/admin/login

管理员登录。

**请求体：**

```json
{
  "username": "admin",
  "password": "your-password"
}
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "bearer"
  }
}
```

**错误：**

- `400` 请求体解析失败或字段缺失。
- `401` 用户名或密码错误。

### POST /api/admin/logout

登出，将当前 token 加入黑名单。

**请求头：**

```
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "message": "ok"
}
```

### GET /api/admin/me

获取当前登录管理员信息。

**请求头：**

```
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "username": "admin"
  }
}
```

## 域名映射接口

### GET /api/admin/domains

获取全部域名映射列表。

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "bind_domain": "api.example.com",
      "target_url": "https://upstream.example.com",
      "real_ip_headers": "X-Real-IP,X-Forwarded-For,CF-Connecting-IP",
      "forward_ip_header": "X-Forwarded-For",
      "request_transform": [{"src": "mobile", "dst": "phone"}],
      "response_transform": [],
      "rewrite_host": true,
      "rewrite_mode": "full",
      "is_default": false,
      "created_at": "2026-07-01T12:00:00+08:00",
      "updated_at": "2026-07-01T12:00:00+08:00"
    }
  ]
}
```

### POST /api/admin/domains

创建域名映射。

**请求体：**

```json
{
  "bind_domain": "api.example.com",
  "target_url": "https://upstream.example.com",
  "real_ip_headers": "X-Real-IP,X-Forwarded-For,CF-Connecting-IP",
  "forward_ip_header": "X-Forwarded-For",
  "request_transform": [{"src": "mobile", "dst": "phone"}],
  "response_transform": [],
  "rewrite_host": true,
  "rewrite_mode": "full",
  "is_default": false
}
```

字段说明：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `bind_domain` | string | 是 | - | 用户访问的绑定域名，保存时会转小写并去空格 |
| `target_url` | string | 是 | - | 上游目标地址，如 `https://target.example.com` |
| `real_ip_headers` | string | 否 | `X-Real-IP,X-Forwarded-For,CF-Connecting-IP` | 逗号分隔的真实 IP 来源头 |
| `forward_ip_header` | string | 否 | `X-Forwarded-For` | 转发给上游时使用的 IP 头 |
| `request_transform` | JSONB / array | 否 | `[]` | 请求体 JSON 字段映射规则 |
| `response_transform` | JSONB / array | 否 | `[]` | 响应体字段映射（当前预留） |
| `rewrite_host` | boolean | 否 | `true` | 是否将请求 Host 改写为目标域名 |
| `rewrite_mode` | string | 否 | `full` | 响应改写模式：`none` / `headers` / `full` |
| `is_default` | boolean | 否 | `false` | 是否为默认站点 |

**响应：** 返回创建后的 Domain 对象，包装在 `data` 中。

### GET /api/admin/domains/:id

获取单个域名映射详情。

**响应：**

```json
{
  "code": 0,
  "data": { /* Domain 对象 */ }
}
```

### PUT /api/admin/domains/:id

更新域名映射。

**请求体：** 同创建。

**响应：**

```json
{
  "code": 0,
  "message": "ok"
}
```

> 注意：设置 `is_default = true` 时，系统会自动取消其他域名的默认状态，保证全表最多只有一个默认站点。

### DELETE /api/admin/domains/:id

删除域名映射（软删除）。

**响应：**

```json
{
  "code": 0,
  "message": "ok"
}
```

## 风控规则接口

### GET /api/admin/rules?domain_id=

获取某个域名下的全部风控规则。

**查询参数：**

- `domain_id`（必填）：域名映射 ID。

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "domain_id": 1,
      "name": "单 IP 单次注册",
      "path_prefix": "/api/register",
      "methods": "POST",
      "rule_type": "duplicate_ip",
      "identity_fields": "phone",
      "max_attempts": 1,
      "window_seconds": 86400,
      "block_seconds": 0,
      "block_status": 403,
      "block_response": {
        "code": 403,
        "message": "重复注册"
      },
      "enabled": true,
      "created_at": "2026-07-01T12:00:00+08:00",
      "updated_at": "2026-07-01T12:00:00+08:00"
    }
  ]
}
```

### POST /api/admin/rules

创建风控规则。

**请求体：**

```json
{
  "domain_id": 1,
  "name": "IP 注册速率限制",
  "path_prefix": "/api/register",
  "methods": "POST",
  "rule_type": "rate_limit",
  "identity_fields": "",
  "max_attempts": 10,
  "window_seconds": 60,
  "block_seconds": 0,
  "block_status": 429,
  "block_response": {
    "code": 429,
    "message": "请求过于频繁，请稍后再试"
  },
  "enabled": true
}
```

字段说明：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `domain_id` | int64 | 是 | - | 所属域名映射 ID |
| `name` | string | 是 | - | 规则名称 |
| `path_prefix` | string | 是 | - | 匹配的路径前缀，如 `/api/register` |
| `methods` | string | 否 | `ALL` | 逗号分隔的 HTTP 方法，`ALL` 表示全部 |
| `rule_type` | string | 是 | - | `duplicate_ip` 或 `rate_limit` |
| `identity_fields` | string | 否 | - | 逗号分隔的 JSON 字段路径，用于构造身份标识 |
| `max_attempts` | int | 否 | `1` | 最大允许次数 |
| `window_seconds` | int | 否 | `0` | 计数窗口秒数，`0` 表示永久 |
| `block_seconds` | int | 否 | `0` | 拦截后封禁时长（秒），当前预留未生效 |
| `block_status` | int | 否 | `403` | 拦截返回的 HTTP 状态码 |
| `block_response` | JSONB / object | 否 | - | 拦截返回的 JSON 内容 |
| `enabled` | boolean | 否 | `true` | 是否启用 |

**响应：** 返回创建后的 Rule 对象。

### GET /api/admin/rules/:id

获取单个规则详情。

### PUT /api/admin/rules/:id

更新规则。

### DELETE /api/admin/rules/:id

删除规则（软删除）。

## 访问日志接口

### GET /api/admin/logs

分页查询全部代理访问日志。

**查询参数：**

- `page`：页码，默认 `1`。
- `page_size`：每页条数，默认 `20`，范围 `1..100`。

**响应：**

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "bind_domain": "api.example.com",
        "client_ip": "1.2.3.4",
        "method": "POST",
        "path": "/api/register",
        "query_params": {"ref": "web"},
        "request_headers": {"Content-Type": "application/json"},
        "request_body": "{\"phone\":\"13800138000\"}",
        "user_agent": "Mozilla/5.0 ...",
        "target_url": "https://upstream.example.com/api/register",
        "status_code": 200,
        "blocked": false,
        "rule_id": null,
        "rule_name": "",
        "message": "",
        "created_at": "2026-07-01T12:00:00+08:00"
      }
    ],
    "total": 1000,
    "page": 1,
    "page_size": 20
  }
}
```

### GET /api/admin/blocks

分页查询仅被拦截的日志。

**查询参数：** 同 `/api/admin/logs`。

**响应：** 数据结构同 `/api/admin/logs`，但只返回 `blocked = true` 的记录。

### GET /api/admin/logs/stats

获取拦截统计聚合数据，用于首页看板。

**响应：**

```json
{
  "code": 0,
  "data": {
    "total_blocked": 128,
    "today_blocked": 12,
    "unique_ips": 45,
    "active_rules": 3,
    "top_ips": [
      {"client_ip": "1.2.3.4", "count": 32}
    ],
    "top_rules": [
      {"rule_id": 1, "rule_name": "单IP单次注册", "count": 56}
    ],
    "daily_trend": [
      {"date": "2026-07-01", "count": 8}
    ]
  }
}
```

字段说明：

| 字段 | 说明 |
|------|------|
| `total_blocked` | 历史累计拦截次数 |
| `today_blocked` | 今日拦截次数 |
| `unique_ips` | 被拦截的不同 IP 数量 |
| `active_rules` | 实际触发过拦截的规则数量 |
| `top_ips` | TOP 10 被拦截 IP 及次数 |
| `top_rules` | TOP 10 触发规则及次数 |
| `daily_trend` | 最近 7 天每日拦截次数 |

## 错误码说明

| HTTP 状态 | 常见场景 |
|-----------|----------|
| 200 | 正常响应，需结合 `code` 判断业务结果。 |
| 204 | OPTIONS 预检请求。 |
| 400 | 请求参数缺失或格式错误。 |
| 401 | 未提供 token、token 无效或已过期/黑名单。 |
| 404 | 资源不存在。 |
| 500 | 服务器内部错误。 |

> 代理入口返回的状态码由上游或风控规则决定，不遵循上述 Admin API 统一响应格式。
