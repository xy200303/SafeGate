# 前端设计

## 目录

- [技术栈](#技术栈)
- [目录结构](#目录结构)
- [构建与开发](#构建与开发)
- [路由设计](#路由设计)
- [页面说明](#页面说明)
- [布局与导航](#布局与导航)
- [状态管理](#状态管理)
- [API 交互](#api-交互)
- [响应式设计](#响应式设计)
- [主题与样式](#主题与样式)

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| React | 19.2 | UI 框架 |
| TypeScript | 6.0 | 类型安全 |
| Vite | 8.1 | 构建工具与开发服务器 |
| React Router DOM | 7.18 | 单页路由 |
| Tailwind CSS | 3.4 | 原子化 CSS |
| shadcn/ui | - | 基于 Radix 的 UI 组件 |
| Axios | 1.18 | HTTP 客户端 |
| Lucide React | 1.23 | 图标 |
| Oxlint | 1.71 | 代码检查 |

## 目录结构

```
web/
├── public/                      # 静态资源
│   ├── favicon.svg
│   └── icons.svg
├── src/
│   ├── api/
│   │   ├── admin.ts             # 所有 Admin API 请求函数与 TypeScript 类型
│   │   └── client.ts            # Axios 实例、拦截器、统一响应包装
│   ├── components/
│   │   ├── layout/
│   │   │   └── AdminLayout.tsx  # 响应式后台布局（侧边栏 / 移动端 Sheet）
│   │   └── ui/                  # shadcn/ui 组件
│   │       ├── badge.tsx
│   │       ├── button.tsx
│   │       ├── card.tsx
│   │       ├── checkbox.tsx
│   │       ├── dialog.tsx
│   │       ├── input.tsx
│   │       ├── label.tsx
│   │       ├── select.tsx
│   │       ├── separator.tsx
│   │       ├── sheet.tsx
│   │       └── table.tsx
│   ├── hooks/
│   │   └── useAuth.ts           # localStorage token 读写与跨标签页同步
│   ├── lib/
│   │   └── utils.ts             # cn() 工具函数
│   ├── pages/
│   │   ├── Login.tsx            # 登录页
│   │   ├── Stats.tsx            # 首页拦截统计看板
│   │   ├── Domains.tsx          # 域名映射管理
│   │   ├── Rules.tsx            # 接口风控规则管理
│   │   ├── Logs.tsx             # 全量访问日志
│   │   ├── BlockedLogs.tsx      # 被拦截日志详情页
│   │   └── FirewallBlacklist.tsx # 风控名单管理页
│   ├── router/
│   │   └── index.tsx            # BrowserRouter 配置与登录守卫
│   ├── App.tsx
│   ├── index.css                # Tailwind 指令 + CSS 变量主题
│   └── main.tsx
├── components.json              # shadcn/ui 配置
├── index.html
├── package.json
├── tailwind.config.js
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── vite.config.ts
└── .oxlintrc.json
```

## 构建与开发

### 脚本

| 脚本 | 命令 | 说明 |
|------|------|------|
| `dev` | `vite` | 启动开发服务器 |
| `build` | `tsc -b && vite build` | TypeScript 检查并构建生产产物 |
| `lint` | `oxlint` | 运行 Oxlint 代码检查 |
| `preview` | `vite preview` | 预览生产构建 |

### 开发服务器代理

`vite.config.ts` 中配置：

```ts
server: {
  proxy: {
    '/api': {
      target: 'http://127.0.0.1:18081',
      changeOrigin: true,
    },
  },
}
```

因此开发时前端请求 `/api/admin/*` 会被代理到本地后端 `18081` 端口。

### 环境变量

通过 `web/.env.local` 配置：

```env
# 前后端同域（推荐生产配置）
VITE_API_BASE_URL=/api

# 本地跨域开发
VITE_API_BASE_URL=http://127.0.0.1:18081/api
```

> 只有以 `VITE_` 开头的环境变量会被暴露到前端代码中。

## 路由设计

`src/router/index.tsx` 使用 `createBrowserRouter`。

| 路由 | 组件 | 访问控制 |
|------|------|----------|
| `/` | 重定向 → `/admin/stats` | 公开 |
| `/login` | `LoginPage` | 公开 |
| `/admin` | `ProtectedRoute` + `AdminLayout` | 需登录 |
| `/admin/stats` | `StatsPage` | 需登录 |
| `/admin/domains` | `DomainsPage` | 需登录 |
| `/admin/rules?domain_id=` | `RulesPage` | 需登录 |
| `/admin/logs` | `LogsPage` | 需登录 |
| `/admin/blocks` | `BlockedLogsPage` | 需登录 |
| `/admin/firewall-blacklist` | `FirewallBlacklistPage` | 需登录 |

`ProtectedRoute` 通过检查 `localStorage` 中是否存在 `token` 来判断登录状态，未登录时重定向到 `/login`。

## 页面说明

### 登录页（/login）

- 居中渐变登录卡片，展示 SafeGate 品牌。
- 表单字段：用户名、密码。
- 登录成功后保存 JWT 到 `localStorage`，跳转 `/admin/stats`。
- 请求失败时在表单下方展示错误信息。

### 首页统计（/admin/stats）

- 四个汇总卡片：总拦截次数、今日拦截、被拦截 IP 数、活跃拦截规则。
- TOP 10 被拦截 IP 表格，带进度条可视化。
- TOP 触发规则表格。
- 近 7 天拦截趋势：纯 CSS 实现的柱状图，显示日期与数量。
- 数据来源：`GET /api/admin/logs/stats`。

### 域名映射页（/admin/domains）

- 表格展示所有域名映射。
- 列：ID、绑定域名、目标地址、转发头、改写模式、是否默认、操作。
- 每行操作：
  - **规则**：跳转到 `/admin/rules?domain_id={id}`。
  - **编辑**：打开编辑 Dialog。
  - **删除**：确认后删除。
- 新增/编辑 Dialog 字段：
  - 绑定域名
  - 目标地址
  - 真实 IP 头（逗号分隔）
  - 转发 IP 头
  - 改写 Host 头（checkbox）
  - 设为默认站点（checkbox）
  - 响应改写模式（`none` / `headers` / `full`）
  - 请求体字段映射（JSON 数组字符串）
  - 响应体字段映射（JSON 数组字符串）
- 保存时会把 JSON 字符串解析为对象后提交。

### 接口风控页（/admin/rules?domain_id=）

- 从 URL 查询参数读取 `domain_id`。
- 展示该域名下的所有规则。
- 列：ID、名称、路径、方法、类型、次数、启用、操作。
- 每行操作：编辑、删除。
- 新增/编辑 Dialog 内置快捷模板：
  - 单 IP 仅一次成功
  - IP + 手机号唯一
  - IP + 手机号 + 姓名 + 收款账户
  - IP 注册速率限制
- 表单字段：
  - 规则名称
  - 路径前缀
  - HTTP 方法
  - 规则类型（`duplicate_ip` / `rate_limit`）
  - 身份字段（逗号分隔）
  - 最大次数
  - 窗口（秒，`0` 表示永久）
  - 拦截状态码
  - 拦截响应体（JSON 字符串）
  - 启用（checkbox）
- `block_response` 保存时从 JSON 字符串解析为对象。

### 访问日志页（/admin/logs）

- 表格展示全量代理日志。
- 列：时间、域名、IP、方法、路径、目标、状态、是否拦截。
- 分页组件，默认每页 20 条。
- 数据来源：`GET /api/admin/logs`。

### 拦截日志页（/admin/blocks）

- 表格展示仅被拦截的日志。
- 列：时间、域名、IP、方法、路径、状态、触发规则、操作。
- 每行可点击展开详情 Dialog，展示：
  - 客户端 IP、方法、状态码、触发时间
  - 请求路径
  - 触发规则名称与 ID
  - 拦截原因 / message
  - User-Agent
  - 查询参数（格式化 JSON）
  - 请求头（格式化 JSON）
  - 请求体原文
- 数据来源：`GET /api/admin/blocks`。

### 风控名单页（/admin/firewall-blacklist）

- 表格展示当前风控名单。
- 列：规则 ID、身份标识、计数、过期时间、操作。
- 支持刷新、删除单条、全部清空。
- 数据来源：`GET /api/admin/firewall/blacklist`。
- 删除或清空时会同时处理 PostgreSQL 持久化记录和 Redis 缓存。

## 布局与导航

### AdminLayout

- **桌面端（≥1024px）**：左侧固定侧边栏，主内容区自适应；侧边栏可折叠为窄模式。
- **移动端（<1024px）**：顶部导航栏带汉堡按钮，点击后从左侧滑出 Sheet 抽屉式侧边栏。
- 导航项：
  - 首页 → `/admin/stats`
  - 域名映射 → `/admin/domains`
  - 访问日志 → `/admin/logs`
  - 拦截日志 → `/admin/blocks`
  - 风控名单 → `/admin/firewall-blacklist`
- 底部提供退出登录按钮。
- 当前路由高亮显示。

### 导航图标

使用 Lucide React 图标，各导航项对应图标：

- 首页：LayoutDashboard
- 域名映射：Globe
- 访问日志：FileText
- 拦截日志：ShieldAlert
- 风控名单：ListX

## 状态管理

- 不使用 Redux / Zustand 等全局状态库。
- 页面级状态使用 React `useState` + `useEffect`。
- 认证状态通过 `useAuth` Hook 读取 `localStorage.token`。
- `localStorage` 的 `storage` 事件用于跨标签页同步登录态。

## API 交互

### Axios 实例

`src/api/client.ts` 创建 Axios 实例：

- `baseURL` 取自 `import.meta.env.VITE_API_BASE_URL || "/api"`。
- 请求拦截器自动附加 `Authorization: Bearer <localStorage.token>`。
- 响应拦截器捕获 `401`，清除 token 并跳转 `/login`。
- 统一包装响应类型 `{ code: number; message?: string; data?: T }`。

### API 函数

`src/api/admin.ts` 封装所有后端接口：

| 函数 | 方法 | 路径 |
|------|------|------|
| `login` | POST | `/admin/login` |
| `logout` | POST | `/admin/logout` |
| `me` | GET | `/admin/me` |
| `listDomains` | GET | `/admin/domains` |
| `createDomain` | POST | `/admin/domains` |
| `updateDomain` | PUT | `/admin/domains/:id` |
| `deleteDomain` | DELETE | `/admin/domains/:id` |
| `listRules` | GET | `/admin/rules?domain_id=` |
| `createRule` | POST | `/admin/rules` |
| `updateRule` | PUT | `/admin/rules/:id` |
| `deleteRule` | DELETE | `/admin/rules/:id` |
| `listLogs` | GET | `/admin/logs` |
| `getBlockedStats` | GET | `/admin/logs/stats` |
| `listBlockedLogs` | GET | `/admin/blocks` |
| `listFirewallBlacklist` | GET | `/admin/firewall/blacklist` |
| `deleteFirewallBlacklistEntry` | DELETE | `/admin/firewall/blacklist?key=` |
| `clearFirewallBlacklist` | POST | `/admin/firewall/blacklist/clear` |

## 响应式设计

断点基于 Tailwind 默认配置：

| 断点 | 宽度 | 布局行为 |
|------|------|----------|
| 默认 | < 1024px | 顶部导航栏 + Sheet 抽屉 |
| `lg` | ≥ 1024px | 左侧固定侧边栏 |

页面表格在移动端允许横向滚动，避免列过多导致布局错乱。

## 主题与样式

- `index.css` 定义了一套以 slate 为主的浅色 CSS 变量主题。
- Tailwind 配置把 CSS 变量映射到 `primary`、`secondary`、`destructive`、`muted`、`accent`、`card` 等颜色。
- 配置了 `darkMode: ["class"]`，但当前未实现深色模式切换开关。
- 登录页使用渐变色背景卡片，营造品牌感。
- 统计页卡片、表格、图表统一使用卡片式布局与柔和阴影。

## 与后端的协作约定

- 前端所有 API 请求都通过 `/api` 前缀，生产环境由 Nginx/1Panel 统一反代到 Admin Server。
- 后端 `MODE=all` 时，Admin Server 会托管前端静态资源，浏览器访问 admin 端口即可直接使用。
- 表单中的 JSON 字符串字段（如 `request_transform`、`block_response`）由前端负责在提交前解析为对象；后端负责在保存到数据库前校验格式。
