# UnoAdmin 前端架构

> 技术基线：Vue 3、TypeScript、Vite、Vue Router、Pinia、UnoCSS + UnoUI、pnpm workspace
>
> 领域模型见 [multi-tenant-model.md](multi-tenant-model.md)。

## 1. 结论

采用「应用负责组装，package 负责能力」的 monorepo 结构：

- `admin-app` 是唯一可部署的 SPA，只负责启动、环境配置、路由创建、服务适配与模块注册。
- `admin-base` 是可复用的基础管理模块，包含 Layout、导航、工作台与租户/主体/组织/账户/角色/权限页面。
- `core` 只放 TypeScript 合同：实体、服务端口、会话契约与注入令牌，不依赖 Vue 与具体 UI 库。
- `routes-generator` 扫描各 package 的 `src/views`，生成 `virtual:router`。
- 所有 UI 统一使用 **UnoUI**（`@mcistudio/unoui-vue`），样式由 UnoCSS 原子类与语义色承载。

## 2. 目录结构

```text
packages/
├── backend/admin-service/            # Go + Gin + MongoDB + Redis
└── frontend/
    ├── admin-app/                    # 唯一可部署 SPA
    │   ├── src/main.ts               # 组装 UnoUI、路由、会话与服务
    │   ├── src/app-session.ts        # 租户 / 经营主体会话状态
    │   ├── src/adapters/             # REST、令牌、上下文头
    │   ├── src/router/               # 生成路由 + 守卫
    │   ├── src/views/                # 登录、403、404
    │   ├── uno.config.ts             # presetUnoUI + 内容扫描
    │   └── vite.config.ts            # UnoCSS、路由生成、依赖预构建
    ├── admin-base/                   # 基础管理模块
    │   ├── src/components/           # PageHeader、NavMenu、ContextSwitcher、树组件
    │   ├── src/composables/          # useAdminBaseContext / Services
    │   └── src/views/                # Layout、Dashboard、System/*
    ├── core/                         # 合同与类型
    └── vite-plugins/routes-generator/
```

## 3. 模块协议

模块通过 `AdminModule` 声明导航与权限：

```ts
export interface NavigationItem {
  id: string
  title: string
  path?: string     // 二级页面路由
  icon?: string     // 一级图标（Iconify class）
  permission?: string
  children?: NavigationItem[]
}
```

- 一级项通常只有 `icon`，承接二级菜单；二级项带 `path` 直接导航。
- `useAdminBaseContext().getNavigation()` 按 `session.can(permission)` 递归过滤，空分组自动隐藏。
- 路由页面的 `<router lang="json">` 中声明 `title` 与 `permission`，与后端权限码一致。

## 4. UI 规范

### 4.1 布局

```text
┌──────────────────────────────────────────────────────────┐
│ 顶部栏 56px：菜单开关 + 品牌 + 面包屑 + 租户/主体切换 + 账户 │
├──────┬───────────┬───────────────────────────────────────┤
│ 100px│ 180px     │ 内容区（bg-secondary）                 │
│ 图标 │ 二级菜单   │ PageHeader + 表格 / 树 / 卡片          │
└──────┴───────────┴───────────────────────────────────────┘
```

- 布局来自 UnoMap 管理端：一级图标栏 100px、二级菜单 180px、内容最小宽度 800px。
- 窄屏（< 900px）侧边栏作为抽屉浮层，选择菜单后自动收起。
- 租户与经营主体切换集中在顶部 `ContextSwitcher`，切换后整页刷新，避免跨上下文脏数据。

### 4.2 组件约定

| 场景 | 组件 |
| --- | --- |
| 页面标题 / 状态 / 操作 | `PageHeader` 插槽 |
| 列表 | `Table` + `Skeleton` + `Pagination` |
| 层级浏览 | `MillerColumns`（权限目录，右侧 `info-panel` 展示详情；`sortable` 启用 Drag & Drop 排序） |
| 新建 / 编辑 | `Drawer` + `Form` + `FormItem` |
| 选择 | `Select`（多选 + `filterable` + `collapse-tags`） |
| 状态 | `Tag`、`Switch` |
| 反馈 | `message`、`Alert`、`Popconfirm` |

约定：

- 组件使用子路径导入，例如 `import { Table } from '@mcistudio/unoui-vue/table'`。
- 图标使用 Iconify 类名，例如 `i-lucide:plus`，不额外引入图标组件。
- 颜色使用语义类：`bg-primary`、`text-secondary`、`border-light`、`bg-brand-50`。
- 表单校验使用 `Form` 的 `rules` 与 `validate()`；业务错误用 `Alert` 呈现，不用浏览器弹窗。

### 4.3 权限呈现三处一致

菜单、路由、按钮使用同一批权限码：

```ts
session.can('account:create')    // 按钮显隐
meta.permission = 'account:view' // 路由守卫
item.permission = 'account:view' // 菜单过滤
```

## 5. 会话与上下文

```ts
export interface AdminSessionState {
  ready: boolean
  account: Account | null
  permissions: string[]
  tenants: Tenant[]
  currentTenant: Tenant | null
  legalEntities: LegalEntity[]
  currentLegalEntity: LegalEntity | null
  dataScope: DataScope
}
```

- `createAdminSession()` 位于 `admin-app`，通过 `AuthPort` 适配器登录、恢复与切换。
- 切换租户或主体时，适配器先换取新令牌，再调用 `auth/restore` 重建会话，最后整页刷新。
- `canAccessEntity(id)` 统一判断主体访问权限：平台管理员与租户级范围放行，其余比对授权列表。

## 6. 服务适配

`core` 定义 `BaseServices` 端口，`admin-app` 提供 REST 实现：

| 端口 | 方法 |
| --- | --- |
| `accounts` | `list` / `create` / `update` / `updateStatus` |
| `roles` | `list` / `create` / `update` |
| `permissions` | `list` / `create` / `update` / `remove` / `reorder` |
| `tenants` | `list` / `create` / `update` |
| `legalEntities` | `list` / `create` / `update` |
| `orgs` | `tree` / `create` / `update` |
| `stats` | `get` |

HTTP 客户端在每次请求注入 `Authorization`、`X-Tenant-Id`、`X-Legal-Entity-Id`；401 时并发请求共享一次刷新。

## 7. 路由生成

`routes-generator` 扫描 `admin-base/src/views` 与 `admin-app/src/views`：

- 文件路径即路由：`System/Accounts.vue` → `/system/accounts`；
- 顶层页面使用 `Layout`，`<router lang="json">` 中 `layout: false` 的页面独立渲染（登录、403、404）；
- 目录可通过 `routeMeta.json` 批量声明元数据；
- 排除规则：`**/components`、`Layout.vue`。

## 8. 检查

```sh
pnpm typecheck
pnpm test
pnpm build
pnpm lint

cd packages/backend/admin-service && go test ./...
```

## 9. 新增业务域

1. 新建 `packages/frontend/domain-xxx`，在 `package.json` 中声明 `@unoadmin/core` 依赖。
2. 在 `src/views` 放页面，在 `src/index.ts` 导出 `AdminModule`（导航 + 权限）。
3. 在 `admin-app/src/modules.ts` 注册模块，在 `vite.config.ts` 的 `routeRoots` 中追加目录。
4. 若需要新服务，先在 `core` 增加端口，再在 `admin-app/src/adapters` 实现。
