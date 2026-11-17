# UnoAdmin 管理工作台

这是 UnoAdmin 的 SPA 应用壳，使用 `routes-generator` 自动汇总应用和基础管理包的页面路由。

## 启动

在仓库根目录执行：

```sh
pnpm install
pnpm dev
```

浏览器打开 <http://localhost:5173/>。应用已移除 mock adapter，所有登录、用户、角色、权限和租户数据都来自 Go 后端微服务。启动前端前请先启动后端：

```sh
cd packages/backend/admin-service
go run ./cmd/seed     # 首次写入种子数据（默认账号 lin.wei / admin123）
go run ./cmd/server   # 默认 HTTP :8080、gRPC :9090，需要本地 MongoDB 与 Redis
```

登录页默认填入种子账号；后端地址默认 `http://localhost:8080/api/v1/`，可在 `admin-app/.env` 中设置 `VITE_API_BASE_URL` 覆盖。HTTP adapter 基于 `ky`，访问令牌与刷新令牌存在浏览器 localStorage，访问令牌过期时会自动用刷新令牌换取新令牌后重试。

示例地址：

- `/dashboard`：基础管理工作台
- `/system/tenants`：租户
- `/system/entities`：经营主体
- `/system/organizations`：组织架构
- `/system/accounts`：账户
- `/system/roles`：角色
- `/system/permissions`：权限

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

端到端测试会通过登录页调用真实后端，因此需先启动后端、数据库与开发服务器；用例覆盖登录进入工作台、权限过滤后的二级导航、深链刷新保持会话、无编辑权限时不渲染新建入口，以及退出登录后的路由守卫拦截。

## SPA 部署

应用使用 `createWebHistory`，静态服务器必须把未知前端路径回退到 `dist/index.html`。Nginx 示例：

```nginx
location / {
  try_files $uri $uri/ /index.html;
}
```

部署在子路径时，同时设置 Vite `base` 和 history fallback 的子路径。
