# 反向代理与风控设计

## 目录

- [反向代理流程](#反向代理流程)
- [域名匹配与默认站点](#域名匹配与默认站点)
- [真实 IP 识别](#真实-ip-识别)
- [响应改写策略](#响应改写策略)
- [风控规则引擎](#风控规则引擎)
- [自定义数据解析与转发](#自定义数据解析与转发)
- [访问日志](#访问日志)

## 反向代理流程

SafeGate 后端启动两个独立的 Gin HTTP Server：

1. **Admin Server**（`ADMIN_PORT`）：处理 `/api/admin/*` 管理接口，并在 `MODE=all` 时托管前端静态页面。
2. **Proxy Server**（`PORT`）：使用 `NoRoute` 捕获所有请求，统一进入反向代理处理函数。

Proxy Server 处理一次请求的完整流程如下：

```
1. 读取请求 Host 头（去除端口，转小写）
2. 按 bind_domain 精确匹配域名映射
   ├─ 命中 → 继续
   └─ 未命中 → 查找 is_default = true 的默认站点
        ├─ 命中 → 继续（标记 default_hit）
        └─ 未命中 → 返回 404 错误页并记录日志
3. 解析 target_url
4. 提取真实 Client IP
5. 加载该域名下的所有风控规则
6. 读取请求体（限制 10MB）
7. 规则匹配与判定
   ├─ 命中阈值 → 返回拦截响应并记录日志
   └─ 未命中 → 继续
8. 应用请求体 JSON 字段映射（request_transform）
9. 构造 ReverseProxy
10. 设置转发头（X-Real-IP、X-Forwarded-For、X-Forwarded-Host 等）
11. 转发到上游目标
12. 接收响应并按 rewrite_mode 改写
13. 对于 duplicate_ip 规则，若上游返回 2xx 则增加计数
14. 异步写入 proxy_logs
```

## 域名匹配与默认站点

### 匹配顺序

1. 将请求 `Host` 头去除端口并转小写。
2. 在 `domains` 表中精确匹配 `bind_domain`。
3. 未命中时，查找 `is_default = true` 的域名。
4. 仍未命中时返回 `404 Not Found`，并渲染内置的 HTML 错误页面。

### 默认站点行为

默认站点类似 Nginx 的 `default_server`，用于兜底未显式配置的域名或直接用 IP 访问的场景。

当命中默认站点时：

- 转发逻辑、风控规则、请求体字段映射等完全正常生效。
- 向上游传递的 `X-Forwarded-Host` 使用用户实际访问的 Host（可能是 IP 或某个未配置域名），而不是 `bind_domain`。
- 响应改写（`Location`、`Set-Cookie`、HTML body）也使用用户实际访问的 Host，保证浏览器继续访问代理入口。

> 注意：全表最多只允许一个默认站点。创建或更新时设置 `is_default = true`，系统会自动取消其他域名的默认状态。

## 真实 IP 识别

### 可配置来源头

`domains.real_ip_headers` 字段为逗号分隔的头部列表，默认值为：

```
X-Real-IP,X-Forwarded-For,CF-Connecting-IP
```

管理员可按实际部署环境调整顺序和内容，例如前置 CDN 时把 `CF-Connecting-IP` 放在最前面。

### 提取逻辑

1. 按 `real_ip_headers` 配置的顺序遍历每个头部。
2. 对于 `X-Forwarded-For`，按逗号分割，取第一个非空、格式有效的 IP。
3. 对于其他头部，直接取第一个有效 IP。
4. 全部未命中时，回退到 `r.RemoteAddr` 的 IP 部分。

### 透传给上游

确定真实 IP 后，SafeGate 会向下游请求中设置/追加以下头部：

| 头部 | 说明 |
|------|------|
| `X-Real-IP` | 设置为真实 IP。 |
| `X-Forwarded-For`（或 `forward_ip_header` 配置的头部） | 追加真实 IP。 |
| `X-Forwarded-Host` | 设置为绑定域名，默认站点命中时使用用户实际 Host。 |

### Host 头改写

通过 `domains.rewrite_host` 控制：

- `true`（默认）：将请求 `Host` 头改写为 `target_url` 的 host，保证上游按目标域名处理。
- `false`：保留原始 `Host` 头。

## 响应改写策略

通过 `domains.rewrite_mode` 控制，支持三种模式：

| 模式 | 行为 | 适用场景 |
|------|------|----------|
| `none` | 只改写 `Host` 头（若开启 `rewrite_host`），不改写响应内容。 | 内部 API、微服务、前端 SPA 使用相对路径。 |
| `headers` | 改写 `Location` 重定向头，并去除 `Set-Cookie` 中的 `Domain=` 属性。 | 需要保持登录态和跳转的外部站点。 |
| `full` | 在 `headers` 基础上，额外改写 HTML body 中的目标域名绝对链接。 | 外部公开站点完整代理（默认）。 |

### 配置建议

- **内部服务 / REST API / SPA**：使用 `none`，避免误改 JS 动态内容，性能最好。
- **需要代理外部站点并保持可点击跳转**：使用 `full`。
- **需要处理登录态但不希望大规模改写 HTML**：使用 `headers`。

> 注意：即使 `full` 模式也无法处理 JS 动态生成的绝对 URL、WebSocket 连接、硬编码在 JS 中的接口地址。

## 风控规则引擎

### 规则匹配条件

请求命中某条规则需同时满足：

1. 规则 `enabled = true`。
2. 请求路径以 `path_prefix` 开头。
3. 请求方法在 `methods` 列表中，或 `methods = ALL`。

### 身份标识生成

```
identity = <client_ip>[|<field>=<value>]...
```

- 若 `identity_fields` 为空，则仅使用真实 IP。
- 否则解析 JSON 请求体，按字段顺序提取值并拼接。

示例：真实 IP 为 `1.2.3.4`，`identity_fields = "phone,email"`，请求体为：

```json
{"phone": "13800138000", "email": "a@example.com"}
```

则身份标识为：

```
1.2.3.4|phone=13800138000|email=a@example.com
```

### 计数方式

使用 Redis String：

```
key = attempt:<rule_id>:<identity>
```

#### duplicate_ip

- **触发时机**：仅在上游返回 2xx 成功后执行 `INCR`。
- **典型用途**：限制同一 IP（或 IP+字段）成功注册、成功领取奖励等操作次数。
- **TTL**：若 `window_seconds > 0`，计数后设置 TTL；`0` 表示永久。

#### rate_limit

- **触发时机**：每次请求命中规则后、转发前执行 `INCR`。
- **典型用途**：通用接口限流，防止刷接口。
- **TTL**：同 `duplicate_ip`。

### 拦截判定

在 `INCR` 之前（`rate_limit`）或上游 2xx 之后（`duplicate_ip`）检查当前计数：

- 若 `count > max_attempts`，则触发拦截。
- 返回 `block_status` 状态码和 `block_response` JSON。
- 记录 `proxy_logs.blocked = true`。

### 拦截响应

#### JSON 响应

当请求 `Accept` 头为 `application/json` 或不包含 `text/html` 时，返回 JSON：

```json
{
  "code": 403,
  "message": "当前 IP 已触发注册频率限制"
}
```

若 `block_response` 为空，后端会自动填充默认值：

```json
{
  "code": <block_status>,
  "message": "blocked",
  "rule_name": "规则名称",
  "rule_type": "duplicate_ip"
}
```

#### HTML 防火墙警告页

当浏览器直接访问（`Accept` 包含 `text/html` 或为 `*/*`）时，返回内置的炫酷 HTML 防火墙警告页。

页面文案可从 `block_response` 中读取以下字段自定义：

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

### 规则模板建议

| 场景 | rule_type | identity_fields | max_attempts | window_seconds |
|------|-----------|-----------------|--------------|----------------|
| 单 IP 仅一次成功注册 | `duplicate_ip` | - | 1 | 0 |
| IP + 手机号唯一注册 | `duplicate_ip` | `phone` | 1 | 0 |
| IP 注册速率限制 | `rate_limit` | - | 10 | 60 |
| IP + 手机号 + 姓名 + 收款账户去重 | `duplicate_ip` | `phone,name,bank_account` | 1 | 0 |

## 自定义数据解析与转发

### 请求体字段映射

`domains.request_transform` 格式为 JSON 数组：

```json
[
  {"src": "mobile", "dst": "phone"},
  {"src": "captcha", "dst": "verify.code"},
  {"src": "user.addr.city", "dst": "city"}
]
```

- `src`：请求体中的源字段路径（gjson 路径语法）。
- `dst`：要写入的目标字段路径（sjson 路径语法）。

### 处理流程

1. 读取请求体，大小限制为 10MB。
2. 校验 `Content-Type` 为 JSON 且请求体为合法 JSON。
3. 使用 `gjson.GetBytes` 提取 `src` 字段的原始 JSON 片段。
4. 使用 `sjson.SetRawBytes` 写入 `dst` 路径。
5. 重新设置 `Content-Length`，并转发修改后的请求体。

### 示例

原始请求体：

```json
{"mobile": "13800138000", "captcha": "123456"}
```

映射规则：

```json
[
  {"src": "mobile", "dst": "phone"},
  {"src": "captcha", "dst": "verify.code"}
]
```

转发后的请求体：

```json
{"phone": "13800138000", "verify": {"code": "123456"}}
```

### 响应体映射

`domains.response_transform` 字段已在模型和 API 中预留，未来可在 `ReverseProxy.ModifyResponse` 中实现与请求体类似的 JSON 字段映射逻辑。当前版本尚未生效。

## 访问日志

每一次进入 Proxy Server 的请求都会被记录到 `proxy_logs` 表。

### 记录字段

| 字段 | 是否始终记录 | 说明 |
|------|--------------|------|
| `bind_domain` | 是 | 命中的绑定域名 |
| `client_ip` | 是 | 真实客户端 IP |
| `method` | 是 | HTTP 方法 |
| `path` | 是 | 请求路径 |
| `target_url` | 是 | 上游目标地址 |
| `status_code` | 是 | 返回状态码 |
| `blocked` | 是 | 是否被拦截 |
| `rule_id` | 拦截时 | 命中规则 ID |
| `rule_name` | 拦截时 | 命中规则名称 |
| `message` | 拦截时 | 拦截说明 |
| `user_agent` | 是 | User-Agent |
| `request_body` | 是 | 请求体原文 |
| `query_params` | 仅拦截时 | 查询参数（JSONB） |
| `request_headers` | 仅拦截时 | 请求头（JSONB） |

### 异步写入

日志写入在独立的 goroutine 中执行，并设置 2 秒超时，避免影响代理响应延迟。如果写入超时或失败，请求本身不会因此失败，但日志会丢失。

### 日志查询

管理后台提供：

- **访问日志页**：查看全量日志，支持分页。
- **拦截日志页**：只查看 `blocked = true` 的日志，支持展开查看请求详情（Query、Headers、Body）。
- **首页统计**：聚合展示拦截趋势、TOP IP、TOP 规则等。
