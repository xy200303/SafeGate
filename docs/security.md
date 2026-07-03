# 安全设计

## 管理员认证

- 密码使用 bcrypt 哈希存储，成本因子默认。
- 首次启动时若未设置 `ADMIN_PASSWORD`，后端生成 16 位随机密码并打印到日志。
- 登录成功后返回 JWT Access Token。

## JWT

- 使用 `HS256` 签名，密钥来自 `JWT_SECRET`。
- Token 包含 `username`、`jti`（唯一 ID）、`exp`。
- 登出时将 `jti` 写入 Redis 黑名单，TTL 与 token 剩余有效期一致。
- 中间件校验签名、过期时间与黑名单。

## 会话安全

- 前端将 JWT 保存在 `localStorage`。
- Axios 请求拦截器自动附加 `Authorization: Bearer <token>`。
- 生产环境建议启用 HTTPS 与 `Secure` 相关配置。

## 接口权限

- `/api/admin/*` 全部需要 JWT 认证。
- 代理路由不需要认证（对外提供服务）。

## 输入校验

- 使用 Gin 的 `binding` 标签对请求体进行必填、类型、URL 格式校验。
- 域名与目标地址统一转小写、去前后空格。

## 风控拦截

- 风控规则仅影响匹配的路径与方法，避免误伤。
- Redis 计数 key 包含规则 ID 与身份标识，防止跨规则污染。

## 日志

- 所有代理请求与拦截事件写入 `proxy_logs`。
- 不记录敏感请求体内容。
