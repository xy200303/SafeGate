# Admin API 文档

所有 Admin API 路径前缀为 `/api/admin`。

## 认证

- 登录成功后返回 `access_token`（JWT）。
- 前端将 token 保存到 `localStorage`，后续请求在 `Authorization: Bearer <token>` 头中携带。
- 登出时调用 `POST /api/admin/logout`，后端将 token 加入 Redis 黑名单直到过期。

## 接口列表

### 认证

| 方法 | 路径 | 说明 | 请求体 | 响应 |
|------|------|------|--------|------|
| POST | `/api/admin/login` | 管理员登录 | `{username, password}` | `{access_token, token_type}` |
| POST | `/api/admin/logout` | 登出（加入黑名单） | - | `{message}` |
| GET  | `/api/admin/me` | 获取当前管理员信息 | - | `{username}` |

### 域名映射

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| GET  | `/api/admin/domains` | 列表 | - |
| POST | `/api/admin/domains` | 创建 | `{bind_domain, target_url, real_ip_headers, forward_ip_header, request_transform, response_transform}` |
| GET  | `/api/admin/domains/:id` | 详情 | - |
| PUT  | `/api/admin/domains/:id` | 更新 | 同创建 |
| DELETE | `/api/admin/domains/:id` | 删除 | - |

### 风控规则

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| GET  | `/api/admin/rules?domain_id=` | 某域名下的规则列表 | - |
| POST | `/api/admin/rules` | 创建 | `{domain_id, name, path_prefix, methods, rule_type, identity_fields, max_attempts, window_seconds, block_status, block_response, enabled}` |
| GET  | `/api/admin/rules/:id` | 详情 | - |
| PUT  | `/api/admin/rules/:id` | 更新 | 同创建 |
| DELETE | `/api/admin/rules/:id` | 删除 | - |

### 访问日志

| 方法 | 路径 | 说明 | 查询参数 |
|------|------|------|----------|
| GET  | `/api/admin/logs` | 分页日志列表 | `page`, `page_size` |
| GET  | `/api/admin/logs/stats` | 拦截统计（按 IP / 规则 / 日期聚合） | - |

`/api/admin/logs/stats` 响应示例：

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

## 通用响应格式

```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```

错误时 `code != 0`，`message` 为错误说明。
