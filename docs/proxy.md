# 反向代理与风控设计

## 反向代理流程

1. 后端启动两个 HTTP 服务：代理入口（`PORT`）和管理后台 API（`ADMIN_PORT`）。
2. 管理后台请求进入 `ADMIN_PORT`，由 Admin API 处理。
3. 代理入口收到的请求查询 PostgreSQL `domains` 表，按 `Host` 头中的 `bind_domain` 查找目标地址。
4. 未命中返回 `404 Not Found`。
5. 命中后：
   - 提取真实 IP
   - 匹配并执行风控规则
   - 执行请求体 JSON 字段映射
   - 构造 `httputil.ReverseProxy` 转发到目标
   - 记录访问日志

## 真实 IP 识别

### 可配置来源头

`domains.real_ip_headers` 字段为逗号分隔的头部列表，默认：

```
CF-Connecting-IP,X-Forwarded-For,X-Real-IP
```

### 提取逻辑

1. 遍历配置的 IP 头。
2. 对于 `X-Forwarded-For`，按逗号分割，取第一个有效 IP。
3. 其他头直接取第一个有效 IP。
4. 全部未命中则回退到 `r.RemoteAddr` 的 IP 部分。

### 透传给上游

- 自动设置 `X-Real-IP: <real_ip>`。
- 根据 `domains.forward_ip_header`（默认 `X-Forwarded-For`）追加或设置 IP。
- 设置 `X-Forwarded-Host: <bind_domain>`。
- 根据 `domains.rewrite_host` 决定是否将请求 `Host` 头改写为目标域名（默认开启）。

## 默认站点映射

类似 Nginx 的 `default_server`，可在 `domains` 表中指定一个 `is_default = true` 的域名。

匹配顺序：

1. 按请求 `Host` 精确匹配 `bind_domain`。
2. 未命中时，查找 `is_default = true` 的域名。
3. 仍未命中则返回 `404 Not Found`。

当命中默认站点时：

- 转发逻辑、风控规则、请求体字段映射等仍正常生效。
- 向上游传递的 `X-Forwarded-Host` 使用用户实际访问的 Host（如 `1.2.3.4` 或某个未配置域名），而不是 `bind_domain`。
- 响应改写（Location / Set-Cookie / HTML body）也使用用户实际访问的 Host，保证浏览器继续访问代理入口。

> 注意：全表最多只允许一个默认站点。设置新的默认站点时，系统会自动取消其他域名的默认状态。

## 响应改写策略

通过 `domains.rewrite_mode` 控制，支持三种模式：

| 模式 | 行为 | 适用场景 |
|------|------|----------|
| `none` | 只改写 `Host` 头（若开启），不改写响应内容 | 内部 API、微服务、同构站点 |
| `headers` | 改写 `Location` 重定向与 `Set-Cookie` 的 Domain | 需要保持登录态跳转的外部站点 |
| `full` | 在 `headers` 基础上，额外改写 HTML body 中的目标域名链接 | 外部公开站点（默认） |

配置建议：
- 内部服务或前端 SPA 使用相对路径时，选 `none` 以获得最佳性能并避免误改 JS 动态内容。
- 需要完整代理外部站点（如百度、政府查询页）时，选 `full`。

> 注意：即使 `full` 模式也无法处理 JS 动态生成的绝对 URL、WebSocket、硬编码在 JS 中的接口地址。

## 风控规则引擎

### 规则匹配

请求命中规则需同时满足：

1. 规则 `enabled = true`
2. 请求路径以 `path_prefix` 开头
3. 请求方法在 `methods` 列表中，或 `methods = ALL`

### 身份标识生成

```
identity = <client_ip>[|<field>=<value>]...
```

- 若 `identity_fields` 为空，则仅使用 IP。
- 否则解析 JSON 请求体，按字段顺序提取值并拼接。

示例：IP 为 `1.2.3.4`，字段 `phone,email`，请求体 `{phone:"138", email:"a@b"}`，则：

```
identity = 1.2.3.4|phone=138|email=a@b
```

### 计数方式

使用 Redis String：

```
key = attempt:<rule_id>:<identity>
```

- `duplicate_ip`：仅在上游返回 2xx 后 `INCR` 并设置 TTL（`window_seconds > 0` 时）。
- `rate_limit`：每次请求前 `INCR`，命中阈值则拦截。

### 拦截响应

- 返回 `block_status` 状态码。
- 返回 `block_response` JSON（若未配置则使用默认提示）。
- 根据请求 `Accept` 头自动选择响应格式：
  - `Accept: application/json`（或不包含 `text/html`）时返回 JSON，保持 API 兼容。
  - 浏览器直接访问（`Accept: text/html` 或 `*/*`）时，返回炫酷的 HTML 防火墙警告页。

HTML 警告页会从 `block_response` 中读取以下字段自定义文案：

| 字段 | 说明 |
|------|------|
| `title` | 页面标题，默认 "访问已被拦截" |
| `message` | 主要提示文案 |
| `detail` | 详细说明，显示在卡片中 |
| `code` | 页面展示的状态码，默认使用 `block_status` |

示例：

```json
{
  "code": 403,
  "title": "访问受限",
  "message": "当前 IP 已触发注册频率限制",
  "detail": "请 24 小时后再试，或联系管理员。"
}
```
- 记录 `proxy_logs.blocked = true`。

## 自定义数据解析/转发

### 请求体字段映射

`domains.request_transform` 格式：

```json
[
  {"src": "mobile", "dst": "phone"},
  {"src": "captcha", "dst": "verify.code"}
]
```

### 处理流程

1. 读取请求体（限制 10MB）。
2. 校验 Content-Type 为 JSON 且 JSON 合法。
3. 使用 `gjson.GetBytes` 提取 `src` 字段原始 JSON 片段。
4. 使用 `sjson.SetRawBytes` 写入 `dst` 路径。
5. 重新设置 `Content-Length` 并转发。

### 响应体映射

`domains.response_transform` 字段已预留，未来可在 `ReverseProxy.ModifyResponse` 中实现相同逻辑。
