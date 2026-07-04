<div align="center">

# 🛡️ SafeGate

**可配置的 IP 风控网关 / 反向代理防火墙**

为 Nginx / 1Panel / Caddy 身后的站点提供域名映射、真实 IP 透传、接口风控拦截、JSON 请求体转换与访问日志审计。

<p>
  <a href="https://github.com/xy200303/SafeGate/stargazers"><img src="https://img.shields.io/github/stars/xy200303/SafeGate?style=social" alt="GitHub Stars"></a>
  <a href="https://github.com/xy200303/SafeGate/issues"><img src="https://img.shields.io/github/issues/xy200303/SafeGate" alt="GitHub Issues"></a>
</p>

<p>
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/Gin-1.12-008ECF?logo=go&logoColor=white" alt="Gin">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker">
</p>

<p>
  <a href="./docs/README.md">📚 文档中心</a> ·
  <a href="#-快速开始">🚀 快速开始</a> ·
  <a href="#-防火墙规则配置">🧱 防火墙规则配置</a> ·
  <a href="#-核心特性">✨ 核心特性</a> ·
  <a href="#-部署架构">🏗️ 部署架构</a> ·
  <a href="#-技术栈">🛠️ 技术栈</a>
</p>

<p>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg" alt="Platform">
</p>

</div>

---

## 🎯 产品定位

把 SafeGate 放在你的统一入口网关之后，它会在请求到达上游业务站点之前完成风控判定、字段转换与日志记录：

```
用户请求
    │
    ▼
Nginx / 1Panel / Caddy（SSL、域名解析）
    │
    ├─ admin.example.com ──▶ SafeGate Admin（管理后台 + API）
    │
    └─ api.example.com ────▶ SafeGate Proxy（风控、转发、审计）
                                │
                                ▼
                         目标站点 / Upstream
```

无论是保护注册接口不被刷单、代理外部站点时自动转换 JSON 字段，还是集中审计所有访问流量，SafeGate 都能以低侵入的方式接入现有架构。

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| 🌐 **域名映射** | 将访问域名绑定到上游目标地址，支持默认站点兜底。 |
| 🔍 **真实 IP 透传** | 自动从 `CF-Connecting-IP`、`X-Forwarded-For`、`X-Real-IP` 等头中提取真实 IP 并转发给上游。 |
| 🛡️ **接口风控引擎** | 支持 `duplicate_ip`（成功后计数）与 `rate_limit`（请求即计数），可按 IP 或 IP+业务字段组合身份。 |
| 📝 **JSON 字段映射** | 使用 `gjson/sjson` 路径语法，在转发前自动重命名、重组请求体字段。 |
| 🔄 **响应智能改写** | `none` / `headers` / `full` 三种模式，处理重定向、Cookie 域与 HTML 正文链接。 |
| 📊 **可视化审计** | 首页统计看板、全量访问日志、被拦截日志详情，支持查看 Query、Headers、Body。 |
| 🖥️ **响应式管理后台** | React + shadcn/ui 构建的现代化控制台，桌面与移动端自适应。 |
| 🐳 **一键部署** | Docker Compose 编排 PostgreSQL、Redis 与后端服务，开箱即用。 |

## 🚀 快速开始

### 环境要求

- Docker Engine >= 24
- Docker Compose >= 2

### 1. 启动服务

```bash
cp .env.example .env
# 编辑 .env，设置 ADMIN_PASSWORD 和 JWT_SECRET
docker compose up -d --build
```

### 2. 获取初始密码

如果未设置 `ADMIN_PASSWORD`，系统会生成随机密码并打印到日志：

```bash
docker compose logs -f backend
```

默认管理员账号：`admin`。

### 3. 访问控制台

| 入口 | 地址 |
|------|------|
| 管理后台 | `http://localhost:18081` |
| 后端 API | `http://127.0.0.1:18081/api/admin` |
| 代理入口 | `http://127.0.0.1:18080` |

## 🧱 防火墙规则配置

SafeGate 的“防火墙规则”指管理后台里的 **接口风控规则**。它不是操作系统层面的 `iptables` / 云防火墙规则，而是在请求进入上游业务站点之前，按域名、路径、HTTP 方法、真实 IP 和业务字段做拦截判断。

### 规则如何生效

一次请求进入代理入口后，大致按下面顺序处理：

```
访问域名匹配 → 提取真实 IP → 读取请求体 → 匹配防火墙规则
    ├─ 超过限制：直接返回 block_status / block_response，并写入拦截日志
    └─ 未超过限制：继续转发到上游站点
```

一条规则只有同时满足以下条件才会参与判断：

1. 规则已启用：`enabled = true`。
2. 请求路径以 `path_prefix` 开头，例如 `/api/register` 会匹配 `/api/register/send`。
3. 如果配置了 `query_match`，请求 URL 必须包含对应 Query 参数，例如 `e=index.post_register`。
4. 请求方法命中 `methods`，或 `methods = ALL`。
5. 该规则属于当前访问域名对应的 `domain_id`。

### 配置项通俗解释

| 配置项 | 示例 | 通俗解释 |
|--------|------|----------|
| `domain_id` | `1` | 这条规则挂在哪个域名映射下面。不同域名的规则互不影响。通常从“域名映射”页点“规则”进入时会自动带上。 |
| `name` | `注册防重复` | 给管理员看的规则名称。建议写清楚用途，方便在拦截日志里快速判断是哪条规则触发。 |
| `path_prefix` | `/api/register` | 要保护的接口路径前缀。只要请求路径以它开头就算命中；如果只想保护注册接口，不要填成 `/api` 这种过宽路径。 |
| `query_match` | `e=index.post_register` | Query 参数匹配条件。为空时不限制 Query；填写后必须全部匹配才会触发规则。多个条件用 `&` 连接，例如 `e=index.post_register&type=1`。 |
| `methods` | `POST` / `POST,PUT` / `ALL` | 限制哪些 HTTP 方法会被检查。注册、提交表单通常填 `POST`；所有方法都要检查时填 `ALL`。 |
| `rule_type` | `duplicate_ip` | 规则类型。`duplicate_ip` 适合“成功后只能再来几次”的场景；`rate_limit` 适合“单位时间内最多请求几次”的限流场景。 |
| `identity_fields` | `phone,email` | 身份字段。为空时只按真实 IP 统计；填写后会把真实 IP 和请求体字段一起作为身份，例如 `IP + 手机号`，减少多个用户共用出口 IP 时的误伤。支持 JSON、普通 POST 表单和 multipart 表单。 |
| `success_statuses` | `2xx` / `2xx,302` | `duplicate_ip` 的成功计数状态码。默认只把 `2xx` 当成功；如果上游注册成功后返回 302，可填 `2xx,302`。 |
| `success_location_match` | `key=register_success` | 可选。`duplicate_ip` 的成功重定向 Query 匹配条件；填写后，`Location` 里的 Query 必须匹配才计数。 |
| `failure_location_match` | `key=username_repeat_register` | 可选。`duplicate_ip` 的失败重定向 Query 匹配条件；匹配到时不计入成功次数。适合上游用 302 跳转页面表示失败的站点。 |
| `max_attempts` | `1` / `10` | 允许次数。当前计数达到这个值后，下一次命中规则的请求会被拦截。 |
| `window_seconds` | `60` / `0` | 统计窗口，单位秒。`60` 表示 60 秒后计数过期；`0` 表示不过期，适合“同一身份只能成功注册一次”。 |
| `block_seconds` | `0` | 预留字段，当前版本尚未实现额外封禁时长逻辑。实际是否拦截主要由 `max_attempts` 和 `window_seconds` 决定。 |
| `block_status` | `403` / `429` | 被拦截时返回给客户端的 HTTP 状态码。重复注册常用 `403`，请求太频繁常用 `429`。 |
| `block_response` | `{"code":403,"message":"重复注册"}` | 被拦截时返回的 JSON 内容。浏览器访问时也会用里面的 `title`、`message`、`detail` 渲染防火墙拦截页。 |
| `enabled` | `true` | 是否启用规则。调试或临时放行时可以关掉，不用删除规则。 |

管理后台的“路径 / Query 匹配”输入框可以直接填写 `/index.php?e=index.post_register`，保存时会自动拆成 `path_prefix = /index.php` 和 `query_match = e=index.post_register`。

### 两种规则类型怎么选

| 规则类型 | 什么时候计数 | 适合场景 | 举例 |
|----------|--------------|----------|------|
| `duplicate_ip` | 请求转发到上游后，根据 `success_statuses`、`success_location_match`、`failure_location_match` 判断成功后再计数。 | 防止同一 IP 或同一业务身份重复完成某个动作。 | 同一 IP 只能成功注册 1 次；同一 IP + 手机号只能领取 1 次奖励。 |
| `rate_limit` | 请求命中规则后、转发到上游前就计数。 | 防止短时间刷接口，不关心上游是否处理成功。 | 同一 IP 每 60 秒最多请求注册接口 10 次。 |

简单理解：想限制“成功次数”用 `duplicate_ip`，想限制“访问频率”用 `rate_limit`。

### 身份字段如何工作

`identity_fields` 用逗号分隔，字段名从请求体里读取。JSON 请求体支持 `gjson` 路径语法，普通 POST 表单和 multipart 表单支持按字段名读取，也兼容 `user.phone` 对应 `user[phone]` 这类常见嵌套表单写法。假设真实 IP 是 `1.2.3.4`，配置为：

```text
identity_fields = phone,email
```

请求体为：

```json
{"phone":"13800138000","email":"a@example.com"}
```

最终统计身份会变成：

```text
1.2.3.4|phone=13800138000|email=a@example.com
```

这样同一个出口 IP 下，不同手机号或邮箱可以分开计数。字段不存在时会按空值拼接，所以请尽量填写上游请求中稳定存在的字段。

POST 表单也可以使用同样的字段配置：

```http
Content-Type: application/x-www-form-urlencoded

phone=13800138000&email=a@example.com
```

如果表单字段使用括号表示嵌套：

```http
user[phone]=13800138000&user[email]=a@example.com
```

身份字段填写：

```text
user.phone,user.email
```

### 拦截响应配置

API 请求被拦截时会返回 `block_status` 和 `block_response`。例如：

```json
{
  "code": 429,
  "message": "请求过于频繁，请稍后再试"
}
```

浏览器访问被拦截时，SafeGate 会渲染内置防火墙警告页。可以在 `block_response` 里增加这些字段来自定义页面文案：

```json
{
  "code": 403,
  "title": "访问受限",
  "message": "当前 IP 已触发注册频率限制",
  "detail": "请稍后再试，或联系管理员。"
}
```

### 常用配置模板

| 场景 | `path_prefix` | `methods` | `rule_type` | `identity_fields` | `max_attempts` | `window_seconds` | `block_status` |
|------|---------------|-----------|-------------|-------------------|----------------|------------------|----------------|
| 单 IP 只能成功注册一次 | `/api/register` | `POST` | `duplicate_ip` | 留空 | `1` | `0` | `403` |
| 同一 IP + 手机号只能成功提交一次 | `/api/register` | `POST` | `duplicate_ip` | `phone` | `1` | `0` | `403` |
| 注册接口 1 分钟最多请求 10 次 | `/api/register` | `POST` | `rate_limit` | 留空 | `10` | `60` | `429` |
| 同一 IP + 手机号 + 姓名 + 收款账户防重复 | `/api/submit` | `POST` | `duplicate_ip` | `phone,name,bank_account` | `1` | `0` | `403` |
| 只保护 `/index.php?e=index.post_register` 注册表单 | `/index.php` | `POST` | `duplicate_ip` | `mobile,accountname,bankaccount` | `1` | `0` | `403` |

其中最后一条还需要额外设置：

```text
query_match = e=index.post_register
success_statuses = 2xx,302
failure_location_match = key=username_repeat_register
```

### 配置建议

- 路径前缀尽量精确，只保护需要风控的接口，避免误伤普通页面或静态资源。
- 前置 CDN / Nginx / 1Panel 时，先确认域名映射里的真实 IP 头配置正确，否则所有用户可能都会被识别成同一个代理 IP。
- `window_seconds = 0` 表示计数长期保留，更适合“一次性动作”；限流类规则建议设置明确窗口，例如 `60`、`300`。
- 上线新规则前可以先用较宽松的 `max_attempts` 观察拦截日志，再逐步收紧。
- 更完整的规则引擎细节见 [docs/proxy.md](./docs/proxy.md#风控规则引擎)。

## 🏗️ 部署架构

生产环境推荐将 SafeGate 部署在 1Panel / Nginx 身后：

```
用户 → 1Panel Nginx（443 SSL）
        │
        ├─ admin.yourdomain.com → 127.0.0.1:18081
        │
        └─ api.yourdomain.com   → 127.0.0.1:18080
                                    │
                                    ▼
                                目标站点 / Upstream
```

- 设置 `MODE=api`，由 Nginx/1Panel 托管前端或继续使用 `MODE=all`。
- 后端监听 `127.0.0.1`，避免直接暴露在公网。
- 管理后台域名限制来源 IP 或仅内网使用。

详细部署指南见 [docs/deployment.md](./docs/deployment.md)。

## 🛠️ 技术栈

**后端**

- Go 1.25 + Gin 1.12
- GORM + PostgreSQL 16
- Redis 7 + go-redis 9
- JWT（golang-jwt/jwt/v5）+ bcrypt
- `net/http/httputil.ReverseProxy`
- tidwall/gjson + tidwall/sjson

**前端**

- React 19 + TypeScript 6
- Vite 8
- Tailwind CSS 3 + shadcn/ui
- React Router DOM 7
- Axios

**部署**

- Docker + Docker Compose
- 支持 1Panel / Nginx / Caddy 反代

## 📚 文档

- [docs/README.md](./docs/README.md) — 文档索引
- [docs/architecture.md](./docs/architecture.md) — 系统架构
- [docs/api.md](./docs/api.md) — Admin REST API
- [docs/database.md](./docs/database.md) — 数据库设计
- [docs/proxy.md](./docs/proxy.md) — 反向代理与风控
- [docs/frontend.md](./docs/frontend.md) — 前端设计
- [docs/deployment.md](./docs/deployment.md) — 部署指南
- [docs/security.md](./docs/security.md) — 安全设计

## 💻 本地开发

```bash
# 启动完整服务栈
docker compose up -d

# 或单独启动前端开发服务器
cd web
cp .env.example .env.local
npm install
npm run dev
```

后端本地开发说明见 [docs/deployment.md](./docs/deployment.md)。

## 🤝 参与贡献

欢迎通过 Issue 和 Pull Request 参与项目改进。修改代码时，请同步更新相关文档。

## 📄 License

[MIT](./LICENSE)
