# UnoAdmin 管理工作台

这是 UnoAdmin 的 SPA 应用壳，使用 `routes-generator` 自动汇总应用、基础管理包和示例业务包的页面路由。

## 启动

在仓库根目录执行：

```sh
pnpm install
pnpm dev
```

浏览器打开 <http://localhost:5173/>。当前应用使用本地 mock adapter，不需要后端即可查看用户、角色、权限、租户和示例业务页面。

示例地址：

- `/dashboard`：基础管理工作台
- `/system/users`：基础包用户管理
- `/system/roles`：基础包角色管理
- `/system/permissions`：基础包权限管理
- `/system/tenants`：基础包租户管理
- `/demo/overview`：独立 `domain-demo` 业务包
- `/demo/records`：示例业务列表

## 常用检查

```sh
pnpm typecheck
pnpm test
pnpm build
pnpm lint
```

## 端到端测试

首次执行需要安装 Playwright 浏览器：

```sh
pnpm exec playwright install chromium
pnpm --filter @unoadmin/admin-app test:e2e --project=chromium
```

如果本机已有可用的 Chromium，也可以通过 `PLAYWRIGHT_EXECUTABLE_PATH` 指定浏览器路径：

```sh
PLAYWRIGHT_EXECUTABLE_PATH="/path/to/chromium" pnpm --filter @unoadmin/admin-app test:e2e --project=chromium
```

端到端测试覆盖 package 自动路由、SPA 深链接刷新和基础 Layout 渲染。

## SPA 部署

应用使用 `createWebHistory`，静态服务器必须把未知前端路径回退到 `dist/index.html`。Nginx 示例：

```nginx
location / {
  try_files $uri $uri/ /index.html;
}
```

部署在子路径时，同时设置 Vite `base` 和 history fallback 的子路径。
