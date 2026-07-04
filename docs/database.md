# 数据库设计

SafeGate 使用 PostgreSQL 作为主数据库，GORM 在启动时自动执行 `AutoMigrate` 创建或更新表结构。风控计数持久化保存在 PostgreSQL，Redis 仅缓存运行时计数并保存 JWT 黑名单。

## 目录

- [PostgreSQL 数据表](#postgresql-数据表)
- [Redis 键设计](#redis-键设计)
- [ER 关系图](#er-关系图)
- [实现说明与注意事项](#实现说明与注意事项)

## PostgreSQL 数据表

### users

管理员账号表。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 自增主键 |
| `username` | VARCHAR(64) | UNIQUE, NOT NULL | 用户名 |
| `password_hash` | VARCHAR(255) | NOT NULL | bcrypt 哈希后的密码 |
| `created_at` | TIMESTAMPTZ | - | 创建时间 |
| `updated_at` | TIMESTAMPTZ | - | 更新时间 |
| `deleted_at` | TIMESTAMPTZ | INDEX | GORM 软删除字段 |

启动时如果 `users` 表为空，会自动创建一个管理员账号：

- 用户名来自 `ADMIN_USERNAME`（默认 `admin`）。
- 密码来自 `ADMIN_PASSWORD`；若未设置，则生成 16 位随机密码并打印到日志。

### domains

域名映射表，描述访问域名与上游目标地址的对应关系。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 自增主键 |
| `bind_domain` | VARCHAR(255) | UNIQUE, NOT NULL | 用户访问的绑定域名，保存时转小写并去前后空格 |
| `target_url` | VARCHAR(512) | NOT NULL | 上游目标地址，如 `https://target.example.com` |
| `real_ip_headers` | VARCHAR(512) | DEFAULT `'X-Real-IP,X-Forwarded-For,CF-Connecting-IP'` | 逗号分隔的真实 IP 来源头 |
| `forward_ip_header` | VARCHAR(128) | DEFAULT `'X-Forwarded-For'` | 转发给上游时使用的 IP 头 |
| `request_transform` | JSONB | DEFAULT `'[]'` | 请求体 JSON 字段映射规则 |
| `response_transform` | JSONB | DEFAULT `'[]'` | 响应体字段映射规则（当前预留，未生效） |
| `rewrite_host` | BOOLEAN | DEFAULT `true` | 是否将请求 Host 改写为目标域名 |
| `rewrite_mode` | VARCHAR(32) | DEFAULT `'full'` | 响应改写模式：`none` / `headers` / `full` |
| `is_default` | BOOLEAN | DEFAULT `false`, INDEX | 是否为默认站点 |
| `created_at` | TIMESTAMPTZ | - | 创建时间 |
| `updated_at` | TIMESTAMPTZ | - | 更新时间 |
| `deleted_at` | TIMESTAMPTZ | INDEX | GORM 软删除字段 |

**索引说明：**

- `bind_domain` 唯一索引：保证域名不重复。
- `is_default` 普通索引：加速默认站点查询。

**默认站点约束：** 业务层保证全表最多只有一个 `is_default = true` 的记录；设置新的默认站点时，会自动取消其他域名的默认状态。

### rules

接口风控规则表。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 自增主键 |
| `domain_id` | BIGINT | NOT NULL, INDEX, FOREIGN KEY → domains.id | 所属域名映射 |
| `name` | VARCHAR(128) | NOT NULL | 规则名称 |
| `path_prefix` | VARCHAR(255) | NOT NULL | 路径前缀，如 `/api/register` |
| `query_match` | VARCHAR(512) | - | Query 参数匹配条件，如 `e=index.post_register`；为空表示不限制 Query |
| `methods` | VARCHAR(128) | DEFAULT `'ALL'` | 逗号分隔的 HTTP 方法，保存时转大写 |
| `rule_type` | VARCHAR(32) | NOT NULL | `duplicate_ip` 或 `rate_limit` |
| `identity_fields` | VARCHAR(512) | - | 逗号分隔的身份字段路径，如 `phone,email`；支持 JSON 路径和 POST 表单字段 |
| `success_statuses` | VARCHAR(128) | DEFAULT `'2xx'` | `duplicate_ip` 成功计数状态码，支持 `2xx`、`302`、`200-299`、`2xx,302` |
| `success_location_match` | VARCHAR(512) | - | 成功重定向 `Location` 的 Query 匹配条件 |
| `failure_location_match` | VARCHAR(512) | - | 失败重定向 `Location` 的 Query 匹配条件，匹配后不计数 |
| `max_attempts` | INT | DEFAULT `1` | 最大允许次数 |
| `window_seconds` | INT | DEFAULT `0` | 计数窗口（秒），`0` 表示永久 |
| `block_seconds` | INT | DEFAULT `0` | 拦截后封禁时长（秒），当前预留，未生效 |
| `block_status` | INT | DEFAULT `403` | 拦截返回的 HTTP 状态码 |
| `block_response` | JSONB | - | 拦截返回的响应体 |
| `enabled` | BOOLEAN | DEFAULT `true` | 是否启用 |
| `created_at` | TIMESTAMPTZ | - | 创建时间 |
| `updated_at` | TIMESTAMPTZ | - | 更新时间 |
| `deleted_at` | TIMESTAMPTZ | INDEX | GORM 软删除字段 |

**索引说明：**

- `domain_id` 索引：加速按域名加载规则。

### proxy_logs

代理访问日志表，记录每一次进入 Proxy Server 的请求。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 自增主键 |
| `bind_domain` | VARCHAR(255) | NOT NULL, INDEX | 命中的绑定域名 |
| `client_ip` | VARCHAR(64) | NOT NULL, INDEX | 客户端真实 IP |
| `method` | VARCHAR(16) | NOT NULL | HTTP 方法 |
| `path` | VARCHAR(2048) | NOT NULL | 请求路径 |
| `query_params` | JSONB | - | 查询参数，仅在被拦截时记录 |
| `request_headers` | JSONB | - | 请求头，仅在被拦截时记录 |
| `request_body` | TEXT | - | 请求体原文 |
| `user_agent` | VARCHAR(512) | - | User-Agent |
| `target_url` | VARCHAR(512) | NOT NULL | 转发到的目标地址 |
| `status_code` | INT | - | 上游返回状态码；拦截时可能为 `block_status` |
| `blocked` | BOOLEAN | DEFAULT `false`, INDEX | 是否被风控拦截 |
| `rule_id` | BIGINT | - | 命中规则 ID |
| `rule_name` | VARCHAR(128) | - | 命中规则名称 |
| `message` | VARCHAR(512) | - | 说明信息 |
| `created_at` | TIMESTAMPTZ | - | 创建时间 |

**索引说明：**

- `bind_domain` 索引：用于按域名查询日志。
- `client_ip` 索引：用于按 IP 聚合统计。
- `blocked` 索引：用于快速筛选拦截日志。

**数据保留建议：** 该表增长较快，生产环境建议定期归档或按时间分区，并避免长期保留完整请求体。

### firewall_attempts

风控计数表，作为重复提交和限流判断的持久化来源。Redis 重启或缓存丢失后，SafeGate 会从该表读取计数并回填缓存。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 自增主键 |
| `rule_id` | BIGINT | NOT NULL, UNIQUE INDEX | 对应风控规则 ID |
| `identity` | TEXT | NOT NULL, UNIQUE INDEX | 真实 IP 与身份字段拼接后的标识 |
| `count` | BIGINT | NOT NULL, DEFAULT `0` | 当前计数 |
| `expires_at` | TIMESTAMPTZ | INDEX | 计数过期时间；为空表示不过期 |
| `last_seen_at` | TIMESTAMPTZ | NOT NULL | 最近一次计数时间 |
| `created_at` | TIMESTAMPTZ | - | 创建时间 |
| `updated_at` | TIMESTAMPTZ | - | 更新时间 |

**索引说明：**

- `(rule_id, identity)` 唯一索引：保证同一规则下同一身份只有一条计数记录。
- `expires_at` 普通索引：加速过期记录清理。

## Redis 键设计

| 键 | 类型 | 说明 |
|----|------|------|
| `attempt:<rule_id>:<identity>` | String | 风控计数缓存。实际持久化来源是 PostgreSQL `firewall_attempts`，Redis miss 时会从数据库回填；若 `window_seconds > 0` 则设置 TTL。 |
| `jwt:blacklist:<jti>` | String | JWT 黑名单标记，TTL 为 token 剩余有效期。 |

### 计数键示例

假设规则 ID 为 `1`，真实 IP 为 `1.2.3.4`，身份字段为 `phone`，请求体中 `phone=13800138000`：

```
attempt:1:1.2.3.4|phone=13800138000
```

如果 `identity_fields` 为空，则仅使用 IP：

```
attempt:1:1.2.3.4
```

## ER 关系图

```
users
  │
domains 1───* rules
  │
  └───────* proxy_logs

rules 1───* firewall_attempts
```

- 一个 `domain` 可拥有多条 `rule`。
- 一个 `domain` 可对应多条 `proxy_log`。
- `rule` 与 `proxy_log` 通过 `rule_id` / `rule_name` 关联，但 `proxy_log` 不建立外键约束，允许规则删除后日志仍可查询。
- `firewall_attempts` 通过 `rule_id` 关联规则，用于持久化保存每条规则下各身份的计数。

## 实现说明与注意事项

1. **JSONB 字段**：`request_transform`、`response_transform`、`block_response`、`query_params`、`request_headers` 均使用 PostgreSQL `jsonb` 类型。GORM 模型中通过自定义 `JSONB` 类型（包装 `json.RawMessage`）实现存取。

2. **软删除**：所有表都包含 GORM 的 `deleted_at` 字段，删除操作默认为逻辑删除。如需物理清理请直接操作数据库或使用自定义脚本。

3. **预留字段**：
   - `domains.response_transform`：当前已在模型中定义并支持通过 API 读写，但代理流程尚未对上游响应体应用该映射。
   - `rules.block_seconds`：当前已在模型中定义并支持通过 API 读写，但规则引擎尚未实现拦截后的额外封禁时长逻辑。

4. **日志敏感数据**：`proxy_logs.request_body` 会记录所有代理请求的请求体原文，生产环境中如果上游传输敏感信息（如密码、身份证号），建议增加脱敏处理或缩短保留周期。

5. **自动迁移**：启动时 GORM `AutoMigrate` 会根据模型自动创建/更新表结构，无需手动维护迁移脚本。但生产环境重大变更仍建议人工 review。
