# UnoAdmin

UnoAdmin 是一个基于 Vue 3、Vite、TypeScript 和 pnpm workspace 的模块化管理系统前端骨架。

## 快速开始

```sh
pnpm install
pnpm dev
```

应用入口位于 `packages/frontend/admin-app`，默认启动地址为 <http://localhost:5173/>。

## Workspace

- `packages/frontend/admin-app`：唯一 SPA 应用壳，负责 Router、守卫、会话与上下文、环境配置和服务 adapter。
- `packages/frontend/admin-base`：可复用的 Layout 和用户、权限、角色、租户基础页面。
- `packages/frontend/domain-demo`：用于验证 package 自动路由的示例业务域。
- `packages/frontend/core`：纯 TypeScript 类型、应用上下文合同、服务 port 和可注入 HTTP 客户端。
- `packages/frontend/ui`：跨业务公共组件，目前包含文件上传。
- `packages/frontend/vite-plugins/routes-generator`：扫描多个 package 的 `src/views` 并生成 `virtual:router`。
- 每个 package 的 `tsconfig.json`：在项目内部维护自己的 TypeScript 编译配置，避免共享配置包影响独立复用。

## 检查

```sh
pnpm typecheck
pnpm test
pnpm build
pnpm lint
```

详细架构和迁移方案见 [docs/frontend-architecture.md](docs/frontend-architecture.md)。
