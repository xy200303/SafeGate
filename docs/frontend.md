# 前端设计

## 技术栈

- React 18 + TypeScript
- Vite（构建工具）
- Tailwind CSS（原子样式）
- shadcn/ui（组件库）
- React Router（路由）
- Axios（HTTP 客户端）

## 目录结构

```
web/
├── public/
├── src/
│   ├── api/                # Axios 实例与 API 请求
│   ├── components/         # 公共组件与 shadcn/ui 组件
│   ├── components/ui/      # shadcn/ui 组件
│   ├── layouts/            # AdminLayout（响应式侧边栏）
│   ├── pages/              # 页面：Login, Domains, Rules, Logs
│   ├── router/             # 路由配置
│   ├── hooks/              # 自定义 hooks
│   ├── lib/                # 工具函数
│   └── App.tsx
├── index.html
├── package.json
├── tailwind.config.js
├── tsconfig.json
└── vite.config.ts
```

## 页面说明

### 登录页（/login）

- 表单：用户名、密码。
- 登录成功后保存 JWT 到 `localStorage`，跳转 `/admin/domains`。

### 管理后台布局（AdminLayout）

- **桌面端**：左侧固定侧边栏，顶部可选面包屑，主内容区自适应。
- **移动端**：顶部导航栏带汉堡按钮，点击后从左侧滑出 Sheet 抽屉式侧边栏。
- 侧边栏菜单：域名映射、接口风控、访问日志、退出登录。

### 域名映射页（/admin/domains）

- 表格展示所有映射。
- 顶部“新增”按钮打开 Dialog 表单。
- 每行支持编辑、删除、进入规则管理。
- 表单字段：绑定域名、目标地址、真实 IP 头、转发 IP 头、请求体映射、响应体映射。

### 接口风控页（/admin/rules?domain_id=）

- 展示某域名下的所有规则。
- 支持新增/编辑/删除。
- 表单字段：名称、路径前缀、方法、规则类型、身份字段、最大次数、时间窗口、拦截状态码、拦截响应体、启用。

### 访问日志页（/admin/logs）

- 表格展示代理日志。
- 分页组件。
- 字段：时间、绑定域名、IP、方法、路径、目标、状态码、是否拦截、信息。

## 状态管理

- 无全局状态库，使用 React `useState` + `useEffect`。
- JWT 保存在 `localStorage`，Axios 请求拦截器自动附加。
- 路由守卫：未登录访问 `/admin/*` 时重定向到 `/login`。

## 响应式断点

- 桌面：`lg:`（≥1024px）显示固定侧边栏。
- 移动端：`<lg` 使用 Sheet 抽屉。

## 与后端交互

- API Base URL 通过环境变量 `VITE_API_BASE_URL` 配置。
- 生产环境中，前端 Nginx 反代 `/api` 到后端服务。
