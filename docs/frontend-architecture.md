# UnoAdmin 前端架构设计

> 状态：实现前设计稿
>
> 技术基线：Vue 3、TypeScript、Vite、Vue Router、Pinia、pnpm workspace
>
> 适用范围：管理系统前端，以及后续需要复用基础管理能力的其他项目

## 1. 目标与结论

系统采用“应用负责组装，package 负责能力”的 monorepo 架构：

- `packages/frontend/admin-app` 是一个具体可部署的单页应用，只负责启动、环境配置、Router 创建、服务适配和业务 package 注册。
- `@unoadmin/admin-base` 是可复用的基础功能包，包含 Layout、用户、权限、角色、租户等基础页面、路由和页面级交互。
- 每个业务域独立为一个 `@unoadmin/domain-*` package，通过统一模块协议接入管理端。
- `@unoadmin/ui` 只承载跨域 UI 组件和设计令牌，例如文件上传、表格、分页、弹窗、空状态等，不包含用户、租户或业务 API。
- `@unoadmin/core` 和 `@unoadmin/runtime` 提供稳定的类型、服务端口、权限运行时、租户上下文和模块注册能力。
- `routes-generator` 是统一的路由生成入口，扫描应用和各 package 的约定目录，生成 `virtual:router`。
- package 之间通过公开 API 和依赖注入协作，不通过相对路径穿透对方 `src` 目录。

核心判断是：基础功能包应当是“可安装的能力模块”，而不是另一个拥有自己的 `main.ts`、Router 单例和环境变量的应用。只有应用壳拥有运行时组装权；整个部署产物只有一个 SPA 入口。

## 2. 推荐目录结构

```text
UnoAdmin/
├── packages/
│   ├── frontend/
│   │   ├── admin-app/                 # 唯一可部署的 SPA 应用壳
│   │   │   ├── src/
│   │   │   │   ├── main.ts             # 应用启动入口
│   │   │   │   ├── app.ts              # 创建 App、注册模块和插件
│   │   │   │   ├── router/
│   │   │   │   │   ├── index.ts        # 创建唯一 Router 实例
│   │   │   │   │   └── guards.ts       # 应用级登录/权限守卫
│   │   │   │   ├── adapters/           # 后端 API、存储、上传等适配器
│   │   │   │   ├── config/             # 环境变量解析和应用配置
│   │   │   │   └── modules.ts          # base + 业务域模块清单
│   │   ├── core/                       # TypeScript 合同、基础类型和 HTTP 客户端
│   │   ├── runtime/                    # Vue/Pinia/Router 运行时
│   │   ├── ui/                         # 跨业务公共组件和设计令牌
│   │   ├── admin-base/                 # Layout + 用户/权限/角色/租户
│   │   │   └── src/views/              # 基础包的自动路由页面
│   │   ├── domain-xxx/                 # 业务域 package，按实际域命名
│   │   │   └── src/views/              # 业务域的自动路由页面
│   │   ├── domain-yyy/
│   │   └── vite-plugins/               # Vite 构建插件
│
├── docs/
├── package.json
├── pnpm-workspace.yaml
└── pnpm-lock.yaml
```

应用壳使用 `@unoadmin/admin-app`，可复用能力包命名为 `@unoadmin/admin-base`。应用壳与基础能力包位于同一个 `packages/frontend` workspace 下，但只有应用壳负责最终部署。

根工作区只纳入前端 package：

```yaml
packages:
  - packages/frontend/*
```

如果未来需要多个应用，可以增加 `packages/frontend/tenant-app`、`packages/frontend/ops-app` 等，它们都可以复用 `@unoadmin/admin-base`，但不应反向成为基础包的依赖。

每个 package 在自己的 `tsconfig.json` 中维护完整编译配置。这样 package 可以独立复制、发布和复用，不需要依赖 workspace 内的配置包；应用壳的 Vite、Vitest、Playwright 和 TypeScript 配置也继续由 `packages/frontend/admin-app` 自己管理。

## 3. Package 职责和依赖方向

### 3.1 `@unoadmin/core`

纯 TypeScript 包，禁止依赖 Vue、Pinia、Vue Router、具体 UI 库和具体后端客户端。它负责定义跨应用都需要理解的合同，并提供不绑定后端的 HTTP 客户端：

- `User`、`Role`、`Permission`、`Tenant` 等领域基础模型。
- `PermissionCode`、`PermissionSet`、`TenantId` 等标识类型。
- `AuthPort`、`UserService`、`RoleService`、`PermissionService`、`TenantService`、`FileStoragePort` 等服务端口。
- `AdminModule`、`NavigationItem`、`PermissionDefinition` 等模块协议。
- `HttpClient`、`HttpClientOptions` 和 `createHttpClient` 等网络基础设施。
- 统一错误类型、分页查询参数、分页结果、排序和过滤模型。
- 当前租户、平台作用域、租户作用域的上下文合同。

`createHttpClient` 只使用标准 Fetch API，并通过 options 注入 API 地址、认证凭证和租户上下文。这样基础功能包可以接入 REST、GraphQL、BFF 或测试内存服务，而不必重写页面。

### 3.2 `@unoadmin/runtime`

Vue 运行时包，依赖 `core`，可以依赖 Vue、Pinia、Vue Router。它负责：

- 创建和持有当前登录会话。
- 创建权限查询器，例如 `can('user:read')`、`require('tenant:manage')`。
- 持有当前租户、可选租户和租户切换状态。
- 创建模块注册表和应用上下文。
- 创建路由守卫需要的运行时接口。
- 把存储、认证、租户上下文等外部能力作为 adapter 注入。

它不包含用户管理页面，也不包含任何具体业务域页面。

### 3.3 `@unoadmin/ui`

跨业务可复用的 UI package，依赖 Vue，必要时依赖一个统一的底层 UI 组件库。它负责：

- Button、Form、Table、Pagination、Dialog、Drawer、Dropdown 等基础封装。
- `FileUpload`、`ImageUpload`、`FileList`、上传进度、失败重试等文件能力。
- Loading、Empty、ErrorState、PermissionGate 等通用展示组件。
- 表单校验展示、响应式布局、颜色和间距等设计令牌。
- 无业务含义的 composable，例如 `usePagination`、`useDialog`。

公共组件只能通过 props、slots、events 和明确的 port 协作。比如 `FileUpload` 不应直接调用某个业务域的 OSS API，而应接收 `FileStoragePort` 或 `uploader` 属性。

### 3.4 `@unoadmin/admin-base`

可发布的基础管理能力包，依赖 `core`、`runtime`、`ui`，可以使用 `core` 的 HTTP 抽象，但不能依赖任何 `domain-*` package。内容包括：

- Layout、侧边导航、顶部栏、面包屑、错误页和权限兜底页面。
- 用户列表、用户详情/编辑、用户状态和用户角色配置。
- 权限列表、权限树和权限详情。
- 角色列表、角色授权和角色成员管理。
- 租户列表、租户详情、租户切换和租户作用域展示。
- 基础模块的路由、导航项、权限定义和服务 port。

基础页面只依赖 `BaseServices`，不绑定后端 URL、DTO 命名和认证实现。应用通过 `createAdminBaseModule({ services, branding, features })` 配置它。

### 3.5 `@unoadmin/domain-*`

每个业务域一个 package，例如 `@unoadmin/domain-orders`、`@unoadmin/domain-assets`。每个业务包负责自己的：

- 页面、局部组件、API adapter 和页面状态。
- 路由、导航、权限定义和审计事件声明。
- 该域的 DTO 到 UI model 的转换。
- 该域的测试和 mock 数据。

业务域可以依赖 `core`、`runtime`、`ui`，并使用 `core` 提供的 HTTP 抽象，不能依赖另一个业务域的内部实现。跨域共用的内容应下沉到 `core` 或 `ui`，而不是复制代码或建立业务域之间的环依赖。

### 3.6 `packages/frontend/admin-app`

应用壳是唯一的组合根，负责：

- 创建 Vue App、Pinia 和 Router。
- 解析运行时配置。
- 实现 `AuthPort`、`UserService`、`TenantService` 等后端适配器。
- 注册 `admin-base` 和业务域模块。
- 配置全局错误处理、埋点、国际化和主题。
- 生成最终的生产构建。

应用壳可以依赖所有前端 package；任何可复用 package 都不能依赖 `packages/frontend/admin-app`。

## 4. 依赖图

```text
                         ┌────────────────────┐
                         │ packages/frontend │
                         │     admin-app      │
                         │  composition root  │
                         └─────────┬──────────┘
                                   │ 注册/组装
              ┌────────────────────┼────────────────────┐
              │                    │                    │
              ▼                    ▼                    ▼
     admin-base module     domain-* modules       app adapters
              │                    │                    │
              ├──────────┬─────────┤                    │
              ▼          ▼                              ▼
          runtime       ui                         后端/API/存储
              │          │
              └──────────┐
                         │
                         ▼
                        core
```

强制规则：

1. 依赖只能从上游能力流向下游组合，禁止循环依赖。
2. `domain-a` 不得导入 `domain-b` 的页面、store 或 API；跨域合同放进 `core`。
3. 页面不得直接访问 `fetch`、`axios`、`localStorage` 或具体环境变量。
4. package 之间只使用 package name 导入，例如 `@unoadmin/ui`，禁止 `../../ui/src/*`。
5. `admin-base` 不得为了复用而反向依赖任何 `domain-*`。

可以在 CI 中使用 `dependency-cruiser`、`madge` 或 ESLint import boundary 规则检查这些约束。

## 5. 统一模块协议

每个可插拔功能包都导出一个模块工厂。路由页面由 `routes-generator` 从 package 的 `src/views` 约定目录自动生成；导航和权限仍必须与模块一起声明，避免新增页面后忘记注册菜单或权限。

示意接口：

```ts
export interface AdminModule {
  /** 稳定的模块标识，用于注册、审计和测试。 */
  id: string

  /** 路由和权限的稳定命名空间，必须和生成器配置一致。 */
  routeNamespace: string

  /** 侧边栏和面包屑使用的导航声明。 */
  navigation: NavigationItem[]

  /** 后端权限码和前端展示元数据。 */
  permissions: PermissionDefinition[]

  /** 可选的模块初始化逻辑。 */
  install?: (context: AdminModuleContext) => void | (() => void)
}
```

实际实现时，可以把 Vue Router 相关类型放在 `runtime`，让 `core` 保持纯 TypeScript；上面的协议是行为要求，不要求所有接口必须位于同一个文件。

模块协议不直接暴露 `routes: RouteRecordRaw[]`。路由文件由生成器统一扫描、校验和生成，模块工厂只提供运行时所需的导航、权限、服务和扩展点。这样既保留文件约定式开发体验，也避免每个 package 手动维护一套路由数组。

应用组装示意：

```ts
const modules = [
  createAdminBaseModule({
    services: baseServices,
    branding: appConfig.branding,
  }),
  createOrdersModule({ services: orderServices }),
  createAssetsModule({ services: assetServices }),
]

const runtime = createAdminRuntime({
  auth: authAdapter,
  storage: storageAdapter,
  modules,
})
```

模块工厂应返回稳定的公开对象。不要让应用通过深层路径导入 `admin-base/src/views/Users.vue`；如果某项目确实需要替换一个基础页面，应通过 `features`、slot、组件覆盖或明确的扩展点完成。

## 6. Layout、路由和权限设计

### 6.1 Router 的所有权

Router 实例只在应用壳创建一次。package 不创建 `createRouter()`，不直接挂载全局守卫，也不使用隐式单例。

推荐流程：

1. Vite 启动时由 `routes-generator` 扫描应用和已启用 package 的页面目录。
2. 生成器通过 `virtual:router` 暴露带有懒加载组件和路由元信息的路由表。
3. 应用创建 runtime 和所有模块。
4. 应用使用 `virtual:router` 创建唯一 Router，并安装基础守卫。
5. runtime 根据当前登录态、租户上下文和权限集决定是否允许访问。
6. 导航菜单由已注册模块的 navigation 声明结合权限集过滤得到。

未授权页面不应依赖后端返回任意组件路径来动态 import。后端只返回权限码或菜单标识，前端只从已注册的安全模块中匹配页面。

### 6.2 Layout 的职责

Layout 只处理应用级布局和导航状态：

- 当前路由、页面标题、面包屑和导航展开状态。
- 当前用户、当前租户和退出登录。
- 全局通知、错误提示和内容区域。
- 响应式侧栏和移动端导航。

Layout 不直接实现用户 CRUD、角色授权或业务域请求；这些功能属于子模块页面。

### 6.3 权限模型

权限码建议使用稳定、可读的资源动作格式，例如：

```text
user:read
user:write
role:read
role:grant
tenant:read
tenant:manage
orders:read
orders:export
```

页面访问、菜单显示和按钮显示都复用同一个权限查询器：

```ts
can('user:write')
require('role:grant')
```

前端权限只负责导航和交互体验，不能作为安全边界。API Gateway 或后端服务必须再次验证用户、角色、租户和资源范围。

### 6.4 租户上下文

租户上下文应是 runtime 的一等状态，而不是散落在各页面的 `ref`：

- 登录后初始化可用租户和默认租户。
- 切换租户时清理或失效化与旧租户相关的查询缓存。
- HTTP 客户端从 runtime 读取当前租户并注入请求上下文。
- 所有租户级 API 明确要求 `tenantId` 或使用统一上下文，不允许页面自行拼接 Header。
- 平台级接口和租户级接口在服务 port 层明确区分。
- 租户切换失败时保留旧上下文，并给出可恢复的错误状态。

这样可以避免“界面切换了租户，但列表仍然显示上一个租户数据”的状态泄漏。

## 7. 公共组件和文件上传

### 7.1 公共组件的分层

公共组件按复用范围分三层：

1. `@unoadmin/ui`：无业务语义，例如文件上传、表格、筛选栏。
2. `@unoadmin/admin-base`：管理系统语义，例如权限树、租户选择器、用户角色选择器。
3. `domain-*` 内部组件：只服务一个业务域，例如订单状态时间线。

当一个组件只在两个业务域中出现时，不要立即放进公共包。先确认它的交互、数据模型和错误处理确实稳定，再提升层级。

### 7.2 文件上传 port

`FileUpload` 只处理文件选择、校验、进度、取消、失败和重试。上传地址、鉴权、对象存储、业务关联和最终文件元数据由 `FileStoragePort` 注入：

```ts
export interface FileStoragePort {
  upload(input: UploadInput, signal?: AbortSignal): Promise<UploadResult>
  remove?(file: StoredFile): Promise<void>
  getPreviewUrl?(file: StoredFile): Promise<string>
}
```

建议上传流程为：

```text
选择文件
  -> 前端类型/大小校验
  -> 获取上传凭证或上传会话
  -> 上传并报告进度
  -> 服务端确认并返回文件元数据
  -> 业务域保存 fileId，而不是保存临时 URL
```

需要明确区分：临时上传凭证、持久化文件 ID、预览 URL 和下载 URL。公共组件不得把对象存储密钥、永久 URL 或业务表字段写死在组件内部。

## 8. 状态和数据访问约定

- Pinia store 只保存跨页面共享且具有生命周期意义的状态，例如会话、权限、租户、Layout 偏好。
- 页面筛选条件、分页和弹窗开关默认保持在页面或 composable 内，不要把所有状态都提升到全局 store。
- 服务端列表、详情和缓存优先使用一个查询层；如果暂时不引入 TanStack Query，也应统一封装 loading、error、refresh 和失效化逻辑。
- API DTO 只能在 adapter 或 service 层出现，页面使用稳定的 UI model。
- 列表刷新、删除、切换租户等操作必须明确刷新哪些查询，不能依赖全局事件总线碰运气刷新。
- 业务数据的最终权限、租户隔离和字段脱敏由后端保证，前端只消费结果并展示状态。

## 9. Vite、发布和依赖管理

### 9.1 Vite 的使用边界

- `packages/frontend/*-app` 使用普通 Vite application build，产出最终静态资源。
- 可复用 package 使用 Vite library mode 或等价的库构建配置，产出 ESM、类型声明和样式入口。
- package 内部开发可以通过 workspace symlink 使用源码，但发布消费者只能通过 `exports` 访问公开入口。
- Vue、Vue Router、Pinia 以及选定的 UI 库应在可复用 package 中声明为 `peerDependencies`，避免项目内出现多份运行时。
- 不把应用的 `vite.config.ts`、`import.meta.env` 或路径 alias 复制进可复用 package。
- `packages/frontend/admin-app` 只配置一个 `index.html` 和一个 Vite application entry，不配置 MPA 多入口，也不引入 SSR。
- `routes-generator` 在开发和生产构建阶段都生成同一个 `virtual:router`，页面组件通过动态 import 拆分为异步 chunk。

package 的 `package.json` 应提供类似的公开入口：

```json
{
  "name": "@unoadmin/admin-base",
  "type": "module",
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./styles.css": "./dist/styles.css"
  },
  "peerDependencies": {
    "pinia": ">=3",
    "vue": ">=3.5",
    "vue-router": ">=4"
  }
}
```

版本策略建议如下：

- `core`、`runtime`、`ui`、`admin-base` 使用独立 semver 版本，但在 monorepo 内通过 `workspace:` 管理。
- 公开的 port、模块协议、权限码和路由 name 视为 API；破坏性变化必须升级 major。
- 同一个仓库内的业务 package 可以独立发布和升级，应用通过锁文件固定版本。
- 如果基础 package 只服务公司内部项目，也建议按照可发布包设计，先用私有 registry，避免未来提取时重做边界。

### 9.2 SPA 运行和部署

应用使用 history 路由，Router 创建方式固定为：

```ts
createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: generatedRoutes,
})
```

SPA 的要求不只是 Vite 配置，还包括静态服务器配置：

- 任意非静态资源 URL，例如 `/users`、`/orders/list`，首次访问和刷新都必须回退到同一个 `index.html`。
- Nginx 等服务器应配置 `try_files $uri $uri/ /index.html`；部署在子路径时，回退目标和 `base` 必须使用同一个子路径。
- 未知前端路径由 Vue Router 的 404 路由展示，而不是由静态服务器直接返回 HTML 404。
- 构建结果是一个可部署的 `dist/`，其中只有一个应用入口，业务页面通过异步 chunk 加载。
- 端到端测试必须覆盖直接打开深层 URL、刷新深层 URL、未登录跳转、无权限跳转和前端 404。

只有在部署环境无法配置 history 回退时才考虑 `createWebHashHistory`。默认不使用 hash 路由，因为它会影响 URL 规范、服务端日志和外部链接体验。

## 10. 路由生成插件的处理建议

`routes-generator` 采用“多路由根目录汇总、单一虚拟模块输出”的方案，成为应用内所有页面路由的唯一生成入口：

- `packages/frontend/admin-app/src/views` 放应用专属页面。
- `@unoadmin/admin-base/src/views` 放 Layout 之外的用户、权限、角色、租户页面。
- 每个 `domain-*` package 在自己的 `src/views` 下放业务页面。
- `components` 目录、Layout、登录页、错误页和显式 `isRouter: false` 页面不生成普通业务路由。
- 页面 `<router lang="json">` 和目录 `routeMeta.json` 继续负责 `title`、`permission`、`routePath`、`param` 等元信息。
- 每个路由根目录必须有明确的模块 ID、路由前缀和 Layout 归属，生成器要在构建时检查路径和 route name 冲突。

目标配置形态如下，应用不再手写业务 `RouteRecordRaw[]`：

```ts
routesGeneratorPlugin({
  layoutDir: resolvePackageView('@unoadmin/admin-base', 'Layout.vue'),
  routeRoots: [
    { moduleId: 'admin-base', packageName: '@unoadmin/admin-base', directory: 'src/views', basePath: '' },
    { moduleId: 'orders', packageName: '@unoadmin/domain-orders', directory: 'src/views', basePath: '/orders' },
    { moduleId: 'assets', packageName: '@unoadmin/domain-assets', directory: 'src/views', basePath: '/assets' },
    { moduleId: 'app', packageName: '@unoadmin/admin-app', directory: 'src/views', basePath: '' },
  ],
})
```

实现时应扩展现有插件的 `layoutDir` 配置，使其支持 `routeRoots`；默认行为仍可保留为扫描当前应用的 `src/views`，保证迁移过程兼容。插件最终应输出：

```ts
declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
```

在 monorepo 开发模式下，`routeRoots` 可以指向 workspace package 的源码目录。发布到 registry 的 package 则应在构建时生成并发布自己的 route manifest/route entry，由同一个插件消费；消费者不需要复制基础包页面，也不需要手动导入单个页面组件。两种来源都必须经过同一套路径、权限、命名和冲突校验。

应用侧只保留登录页、全局 404/403、回调页等特殊手工路由。需要避免生成时，应在页面中声明 `isRouter: false`，防止同一个 URL 同时出现在自动路由和手工路由中。

## 11. 测试和质量门禁

每个 package 都应在自己的边界内测试，应用再做一次组合验证：

- `core`：类型合同、权限码、分页模型、租户上下文的纯函数测试。
- `runtime`：登录态恢复、权限守卫、租户切换、缓存失效和模块注册测试。
- `ui`：组件 props/events/slots、键盘操作、错误和 loading 状态测试。
- `admin-base`：用户、权限、角色、租户的页面行为测试，以及无权限和空数据状态。
- `domain-*`：业务域页面、服务 adapter 和模块声明测试。
- `packages/frontend/admin-app`：路由组合、登录跳转、真实 API mock、关键端到端流程。

最低 CI 门禁建议：

```bash
pnpm install --frozen-lockfile
pnpm lint
pnpm typecheck
pnpm test:unit
pnpm build
pnpm test:e2e
```

还应加入以下结构检查：

- 依赖图无环。
- package 只能从公开 exports 导入。
- 所有模块的 route、navigation、permission 声明完整。
- 生产构建中 Vue 等 peer dependency 没有被重复打包。
- `virtual:router` 在开发和生产构建中都能包含基础包、业务包和应用专属页面。
- 关键的 390px/移动端视口、权限拒绝、租户切换、上传失败和 SPA 深链接刷新流程有浏览器验证。

## 12. 分阶段落地计划

### 阶段一：建立边界

1. 将应用壳放入 `packages/frontend/admin-app`，包名使用 `@unoadmin/admin-app`。
2. 更新 workspace、TypeScript project references、lint、format 和测试脚本。
3. 创建 `core`、`runtime`、`ui` 三个最小 package，先定义 port、HTTP 客户端和模块合同。
4. 统一 Vue、Router、Pinia 和 UI 组件库版本，避免每个 package 各自选择。

### 阶段二：提取基础能力

1. 实现 `@unoadmin/admin-base` 的 Layout 和基础模块。
2. 将用户、权限、角色、租户页面迁移到基础包。
3. 删除基础页面对应用环境变量、Router 单例和具体后端客户端的直接依赖。
4. 为基础包增加一个最小 demo app，验证它能脱离主应用运行。

### 阶段三：接入业务域

1. 每次只迁移一个业务域，创建对应 `domain-*` package。
2. 为每个域补齐模块 ID、路由、导航、权限和 service adapter。
3. 从应用组合根注册模块，检查业务包之间没有反向依赖。
4. 将跨域重复组件下沉到 `ui`，但保留明确的提升依据。

### 阶段四：发布和复用验证

1. 为 `ui`、`runtime`、`admin-base` 配置库构建、声明文件和 exports。
2. 使用私有 registry 或 changesets 管理版本和变更日志。
3. 新建第二个最小消费者应用，验证基础包可以复用并支持不同品牌、API 地址和权限模型。
4. 通过组合测试、构建产物检查和浏览器流程验证后，再移除旧的应用内重复实现。

## 13. 首版验收标准

- 新建一个应用时，只需实现 adapter、配置 Layout/品牌信息并注册模块，就能获得用户、权限、角色、租户基础能力。
- `@unoadmin/admin-base` 不依赖任何 `domain-*`，并且没有硬编码 API URL、Token 存储方式或租户 Header。
- 新增业务域时，不修改 `admin-base` 内部源码，只新增一个 `domain-*` package 和应用注册项。
- 新增路由页面时，只需放入约定的 `src/views` 并补充页面/目录元信息，重启开发服务后即可进入 `virtual:router`。
- 文件上传组件可以切换本地 mock、REST 上传和对象存储上传，而不修改组件源码。
- 未授权用户不会看到对应导航，也无法通过直接 URL 进入页面；后端仍执行最终授权。
- 切换租户后，旧租户列表和详情缓存不会继续显示在新租户上下文中。
- package 可以独立 typecheck、test 和 build，应用可以执行一次完整组合构建。
- 所有 package 的外部使用方式都通过 `index.ts` 和 `package.json.exports` 暴露，内部目录结构可以独立演进。
- 应用只有一个 SPA 入口，直接访问和刷新任意合法深层 URL 都能恢复对应页面。

## 14. 需要在实现前确认的产品合同

架构可以先按上述边界落地，但以下内容需要和后端/API 契约一起定稿：

- 权限码是资源动作模型，还是后端菜单树模型；建议优先采用资源动作模型。
- 平台管理员是否跨租户，租户切换是否影响全部请求和缓存。
- 用户、角色、权限和租户的唯一 ID 类型，以及删除/禁用语义。
- 文件上传是直传对象存储、服务端中转，还是两者兼容。
- 基础包需要支持哪些认证方式，以及 Token 刷新由谁负责。
- 不同消费者项目是否允许覆盖基础路由、主题、文案和某些基础页面。

这些属于公开合同。应先写进 `core` 的类型和测试，再开始大规模页面迁移。
