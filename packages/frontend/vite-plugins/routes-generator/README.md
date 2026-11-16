# Vue Vite Routes Generator

一个 Vue + Vite 自动路由生成插件。它从任意数量的页面根目录收集 `.vue`、`.tsx`、`.jsx` 文件，读取页面中的路由声明，生成统一的 `virtual:router` 虚拟模块。

插件只负责生成 Vue Router 路由记录，不绑定具体 UI、权限系统、菜单协议或后端接口，可用于 SPA、后台管理界面、工具站以及 monorepo 中的多包页面聚合。

> 下文使用 `@your-scope/vite-routes-generator` 作为示例包名。安装和导入时请替换为实际发布的包名；monorepo 中也可以使用对应 package 的子路径。

## 特性

- 支持单页面根目录，也支持从多个 package 或目录聚合页面。
- 基于文件路径生成嵌套路由、路由名称、标题和默认权限码。
- 支持页面级 `<router lang="json">` 配置。
- 支持目录级 `routeMeta.json` 元信息。
- 支持 Gitignore 风格的目录与文件排除规则。
- 支持 Layout 页面和不依赖 Layout 的 standalone 页面。
- 页面组件使用动态 import，适合按路由拆包。
- 提供同步生成函数，便于在单元测试中校验路由契约。

## 环境要求

- Vue 3
- Vue Router
- Vite
- 页面文件使用 `.vue`、`.tsx` 或 `.jsx`

如果项目使用 JSX/TSX，Vite 中还需要配置对应的 Vue JSX 插件。

## 安装

```bash
pnpm add -D @your-scope/vite-routes-generator
```

## 快速开始

```ts
import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { routesGeneratorPlugin } from '@your-scope/vite-routes-generator'

const projectRoot = fileURLToPath(new URL('.', import.meta.url))
const appViews = fileURLToPath(new URL('./src/views', import.meta.url))
const reportsViews = fileURLToPath(new URL('./packages/reports/src/views', import.meta.url))

export default defineConfig({
  plugins: [
    vue(),
    routesGeneratorPlugin({
      layoutDir: '/src/layouts/DefaultLayout.vue',
      exclude: ['**/components', '**/*.spec.vue'],
      routeRoots: [
        {
          namePrefix: 'app',
          directory: appViews,
          basePath: ''
        },
        {
          namePrefix: 'reports',
          directory: reportsViews,
          basePath: '/reports'
        }
      ]
    })
  ],
  server: {
    fs: {
      allow: [projectRoot]
    }
  }
})
```

当 `routeRoots` 指向 Vite 项目根目录之外的 package 时，需要通过 `server.fs.allow` 放行对应工作区；否则开发服务器可能拒绝加载生成的动态 import。

### 目录示例

```text
project/
├── src/
│   ├── layouts/
│   │   └── DefaultLayout.vue
│   └── views/
│       ├── Dashboard.vue
│       └── System/
│           ├── Users.vue
│           └── UserProfile.vue
└── packages/
    └── reports/
        └── src/
            └── views/
                ├── Overview.vue
                └── Records.vue
```

配置上述 root 后，页面会映射为：

| 页面文件                                  | 生成路径                                                     |
| ----------------------------------------- | ------------------------------------------------------------ |
| `src/views/Dashboard.vue`                 | `/dashboard`                                                 |
| `src/views/System/Users.vue`              | `/system/users`                                              |
| `src/views/System/UserProfile.vue`        | `/system/userprofile`，同时生成 `/system/user-profile` alias |
| `packages/reports/src/views/Overview.vue` | `/reports/overview`                                          |
| `packages/reports/src/views/Records.vue`  | `/reports/records`                                           |

## 工作流程

```text
读取 Options
  → 遍历 routeRoots
  → 应用 Gitignore 风格 exclude 规则
  → 读取页面 <router lang="json"> 与目录 routeMeta.json
  → 计算 basePath + 页面相对路径
  → 生成 Layout 路由树或 standalone 顶层路由
  → 校验路径归属和配置合法性
  → 输出 virtual:router
```

Vite 钩子行为：

| 钩子              | 行为                                       |
| ----------------- | ------------------------------------------ |
| `resolveId`       | 将 `virtual:router` 解析为内部模块 id      |
| `load`            | 静态导入 Layout，并导出生成的路由数组      |
| `handleHotUpdate` | 参与扫描的文件变化时失效虚拟模块并触发刷新 |

## API

```ts
import { getGeneratedRouteRecords, routesGeneratorPlugin } from '@your-scope/vite-routes-generator'
```

### `routesGeneratorPlugin(options?)`

返回 Vite 插件实例。插件生成：

```ts
import Layout from '<layoutDir>'

export default [
  // RouteRecordRaw[]
]
```

### `getGeneratedRouteRecords(options?)`

同步生成路由记录，不依赖 Vite 运行时。适合用于：

- 路由快照测试
- 菜单指向校验
- 权限码一致性检查
- 构建前的自定义静态检查

## 配置项

```ts
interface Options {
  layoutDir?: string
  routeRoots?: RouteRoot[]
  exclude?: readonly string[]
}
```

| 字段         | 类型                | 默认值                         | 说明                                                           |
| ------------ | ------------------- | ------------------------------ | -------------------------------------------------------------- |
| `layoutDir`  | `string`            | `/src/views/Layout.vue`        | Layout 组件的导入说明符。可以使用 Vite root 相对路径或绝对路径 |
| `routeRoots` | `RouteRoot[]`       | 当前工作目录下的 `./src/views` | 页面根目录列表。为空时退化为单 root 扫描                       |
| `exclude`    | `readonly string[]` | `[]`                           | 应用于所有 root 的 Gitignore 风格排除规则                      |

### `RouteRoot`

```ts
interface RouteRoot {
  namePrefix?: string
  directory: string
  basePath?: string
  exclude?: readonly string[]
}
```

| 字段        | 必填 | 说明                                                  |
| ----------- | ---- | ----------------------------------------------------- |
| `namePrefix` | 否 | 可选的路由名称前缀；省略时直接使用路径生成路由名 |
| `directory` | 是   | 页面根目录。建议使用绝对路径                          |
| `basePath`  | 否   | URL 前缀，例如 `/reports`                             |
| `exclude`   | 否   | 仅作用于当前 root 的排除规则，追加在全局 `exclude` 后 |

`namePrefix` 不是 URL 前缀，也不会自动出现在浏览器地址中。它只影响路由名称。例如：

```ts
{ namePrefix: 'app', basePath: '' }
```

`Reports/Overview.vue` 会生成：

```ts
{
  path: '/reports/overview',
  name: 'app-reports-overview',
}
```

对于根目录下的 `Overview.vue`，如果同时配置 `basePath`，前缀也会进入路由名称：

```ts
{ namePrefix: 'app', basePath: '/workspace' }
// /workspace/overview
// name: app-workspace-overview
```

省略 `namePrefix` 时，名称只由路径生成：

```ts
{ directory: appViews, basePath: '' }
// /system/users
// name: system-users
```

## 排除规则

插件没有内置的文件名或目录排除项。未配置 `exclude` 时，root 下所有页面文件都会参与扫描。

规则支持 [Gitignore 模式语法](https://git-scm.com/docs/gitignore#_pattern_format)：

```ts
routesGeneratorPlugin({
  exclude: ['**/components', '**/*.spec.vue', 'legacy/**', '!legacy/Keep.vue']
})
```

语义说明：

- 规则相对于各自 `RouteRoot.directory`。
- 支持 `*`、`?`、`**`、字符范围、行首 `/` 锚定和行尾 `/` 目录匹配。
- 默认大小写不敏感。
- 后匹配的规则覆盖先匹配的规则。
- `!` 可以重新包含文件，但 Gitignore 对父目录的忽略规则仍然生效。
- 插件不会自动读取项目中的 `.gitignore` 文件。
- `directory`、页面文件和 `routeMeta.json` 使用同一套规则。

## 页面路由配置

页面可以通过 `<router lang="json">` 声明生成选项：

```vue
<router lang="json">
{
  "title": "用户列表",
  "permission": "users:read"
}
</router>
```

支持以下保留字段：

| 字段        | 类型      | 默认值   | 作用                                                    |
| ----------- | --------- | -------- | ------------------------------------------------------- |
| `isRouter`  | `boolean` | `true`   | 为 `false` 时不生成该页面的路由                         |
| `layout`    | `boolean` | `true`   | 为 `false` 时生成不依赖 Layout 的顶层路由               |
| `routePath` | `string`  | 文件路径 | 覆盖根据文件推导的局部路径，仍会拼接 root 的 `basePath` |
| `param`     | `string`  | 无       | 在路径末尾追加参数段，例如 `":id"`                      |

除以上字段外，其余内容都会写入顶层路由记录的 `meta`。

### 显式路径

`routePath` 是相对于当前 root 的局部路径：

```vue
<router lang="json">
{
  "routePath": "/account/profile"
}
</router>
```

在 `basePath: '/admin'` 的 root 中，该配置会生成 `/admin/account/profile`。

显式指定 `routePath` 时，插件不会额外生成 kebab-case alias。

### 动态参数

```vue
<router lang="json">
{
  "param": ":id"
}
</router>
```

对于 `Users/Detail.vue`，会生成 `/users/detail/:id`。路由记录统一设置 `props: true`，参数会作为 props 传入页面组件。

## Layout 与 standalone 页面

### 默认 Layout 模式

页面未设置 `layout` 或设置为 `true` 时，首段路由使用 Layout：

```ts
{
  path: '/dashboard',
  name: 'app-dashboard',
  component: Layout,
  children: [
    {
      path: '',
      name: 'app-dashboard-index',
      component: () => import('/absolute/path/Dashboard.vue'),
    },
  ],
}
```

嵌套页面会在各路径层级建立容器，例如：

```text
System/Users.vue
→ /system Layout 容器
→ /system/users 页面路由
```

### standalone 模式

不需要 Layout 的页面声明：

```vue
<router lang="json">
{
  "layout": false,
  "routePath": "/standalone/preview"
}
</router>
```

生成结果直接是一条顶层页面路由：

```ts
{
  path: '/standalone/preview',
  name: 'app-standalone-preview',
  component: () => import('/absolute/path/Preview.vue'),
}
```

standalone 页面不会创建 Layout 父级，也不会隐式创建中间路径容器。

### 根路径 `Index.vue`

根目录的 `Index.vue` 没有可推导路径，因此必须显式声明：

```vue
<router lang="json">
{
  "layout": false,
  "routePath": "/"
}
</router>
```

生成 `/` 后，应用中不能再保留同一路径的手工路由或重定向，否则 Vue Router 会出现路径竞争。插件只校验自己生成的路由，无法检查应用手工路由表。

### 路径冲突

同一路径不能同时由 Layout 页面和 standalone 页面占用。例如：

```text
Reports/Index.vue   layout: false  → /reports
Reports/Users.vue   layout: true   → /reports Layout 容器
```

这种配置会在生成阶段直接报错。可以：

- 给 standalone 页面设置其他 `routePath`。
- 将整个目录改为 standalone 路由。
- 将 Layout 页面移动到其他路径。

## 路径与名称规则

### 路径

- 去掉文件扩展名。
- `index` 作为最后一段时会被移除。
- 页面相对路径整体转小写。
- 拼接 root 的 `basePath`。
- `index.vue` 生成的页面路由使用 `path: ''`。

示例：

| 页面               | 生成路径                              |
| ------------------ | ------------------------------------- |
| `Dashboard.vue`    | `/dashboard`                          |
| `System/Users.vue` | `/system/users`                       |
| `System/Index.vue` | `/system` 的 index 页面               |
| `UserProfile.vue`  | `/userprofile`，alias `/user-profile` |

### 名称

路由名默认格式：

```text
[<namePrefix>-]<路径分段用 - 连接>
```

例如：

| 路由                    | `name`                |
| ----------------------- | --------------------- |
| `/system`               | `app-system`          |
| `/system/users`         | `app-system-users`    |
| 一级页面的 index 子路由 | `app-dashboard-index` |
| 根 standalone 页面      | `app-index`           |

不配置 `namePrefix` 时，对应名称分别是 `system`、`system-users`、`dashboard-index` 和 `index`。

顶层 `name` 是 Vue Router 的路由标识。`meta.name` 没有框架级语义，不会覆盖顶层 `name`。

## 元信息

### 默认 `meta`

每个页面路由至少会生成：

```ts
{
  title: 'system.users',
  permission: 'system:users:view',
}
```

- `title` 默认为路径分段用 `.` 连接。
- `permission` 默认为路径分段用 `:` 连接，并追加 `:view`。
- 根 standalone 页面使用 `index` 作为默认名称空间。

这些值只作为元信息声明，不会自动注册权限、修改页面标题或创建菜单。

### 页面级元信息

页面 `<router>` 中除保留字段外的内容会覆盖默认 `meta`：

```vue
<router lang="json">
{
  "title": "用户列表",
  "permission": "users:read",
  "icon": "users"
}
</router>
```

### 目录级 `routeMeta.json`

目录中的 `routeMeta.json` 为该目录对应的 Layout 容器提供元信息：

```text
views/
├── System/
│   ├── routeMeta.json
│   ├── Users.vue
│   └── Roles.vue
```

```json
{
  "title": "系统管理",
  "permission": "system:view",
  "icon": "settings"
}
```

生成的 `/system` 容器会使用这些字段。

注意：

- `routeMeta.json` 不生成路由。
- 目录元信息默认不会继承到所有子页面。
- 页面级字段只覆盖同一条路由记录上的值。
- `isRouter`、`layout`、`param`、`routePath` 写在目录配置中时只是普通元数据，不会改变路由生成行为。

### 合并顺序

同一条记录上的覆盖顺序为：

```text
默认值 → 目录 routeMeta.json → 页面配置
```

## 使用 `virtual:router`

在环境类型声明中注册虚拟模块：

```ts
declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
```

创建 Router：

```ts
import { createRouter, createWebHistory } from 'vue-router'
import generatedRoutes from 'virtual:router'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/signin',
      name: 'signin',
      component: () => import('./views/Signin.vue')
    },
    ...generatedRoutes,
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('./views/NotFound.vue')
    }
  ]
})
```

插件不创建 Router，也不安装导航守卫。应用可以按自身需要消费 `meta`：

```ts
router.beforeEach((to) => {
  const permission = to.meta.permission
  if (typeof permission !== 'string' || canAccess(permission)) return true
  return { name: 'forbidden' }
})
```

前端权限判断不是安全边界，服务端仍需独立鉴权。

## 热更新与测试

开发环境下：

- 未排除页面文件发生变化时，`virtual:router` 会失效并刷新页面。
- 被排除文件不会触发路由刷新。
- 新增目录若未被文件监听器感知，可重启 Vite。

使用同步 API 编写路由契约测试：

```ts
import { expect, it } from 'vitest'
import { getGeneratedRouteRecords } from '@your-scope/vite-routes-generator'

it('contains the reports page', () => {
  const routes = getGeneratedRouteRecords({
    routeRoots: [
      {
        namePrefix: 'reports',
        directory: '/absolute/path/to/reports/views',
        basePath: '/reports'
      }
    ]
  })

  const reportsRoute = routes.find((route) => route.path === '/reports')
  expect(reportsRoute?.children?.some((route) => route.path === '/reports/overview')).toBe(true)
})
```

## 常见错误

| 错误                                            | 原因                                            |
| ----------------------------------------------- | ----------------------------------------------- |
| `Route root does not exist: X`                  | `directory` 不存在或路径错误                    |
| `Route path X is declared by both A and B`      | 多个 root 或 Layout/standalone 模式占用了同一路径 |
| `Route file X must resolve to a non-empty path` | 页面无法推导出路径                              |
| `field "layout" must be a boolean`              | 页面 `layout` 配置不是布尔值                    |
| `must contain a JSON object`                    | `routeMeta.json` 或页面 JSON 配置非法           |

## 已知限制

- 带 `basePath` 的 root 中，`routeMeta.json` 的索引 key 不包含前缀，因此当前无法匹配对应容器。
- 根目录的 `routeMeta.json` 没有对应路由容器，不会产生根路径元信息。
- `routeMeta.json` 不会向所有子页面继承。
- 普通一级 Layout 页面对 `param` 不生效；需要动态一级路径时可使用嵌套页面或 standalone 页面。
- 只支持一个全局 `layoutDir`，尚不支持按页面选择不同 Layout 组件。
- 即使所有页面都使用 `layout: false`，虚拟模块仍会静态导入 `layoutDir`，因此该导入路径必须有效。
- canonical 路径会整体转小写，PascalCase 只额外提供 kebab-case alias。
- 路由名称由可选 `namePrefix` 与路径共同生成，目前不能在页面配置中单独覆盖。
- 插件只在 Vite 构建阶段扫描页面，不负责运行时远程路由或后端下发组件。
- 插件不生成菜单、面包屑、页面标题、权限守卫或登录流程。

## 设计边界

插件解决的问题是“文件如何变成 Vue Router 路由记录”。

它刻意不处理：

- 应用如何创建 Router。
- 用户是否登录。
- 用户是否拥有访问权限。
- 侧边栏、菜单和页签如何生成。
- 后端接口如何组织。
- 页面采用何种 UI 框架或视觉规范。

这些能力应由应用或独立模块实现，通过 `path`、`name` 和 `meta` 与生成路由对接。
