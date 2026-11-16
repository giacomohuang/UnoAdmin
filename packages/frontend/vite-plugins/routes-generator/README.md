# routes-generator

UnoAdmin SPA 的统一路由生成插件。它在 **Vite 构建期**扫描应用与 workspace package 的多个 `src/views` 根目录，把页面汇总成一张路由表，通过 `virtual:router` 虚拟模块暴露给应用；应用只创建一个 Vue Router 实例并消费该模块。

包名为 `@unoadmin/vite-plugins`，子路径导出 `@unoadmin/vite-plugins/routes-generator` 定义在 [`../package.json`](../package.json)；入口是 TypeScript 源码（无独立构建步骤），供 workspace 内的 `dev` / `build` 使用。

## 职责边界

| 插件负责 | 插件不负责 |
| --- | --- |
| 扫描页面文件、生成路径与嵌套结构 | 创建 `createRouter()` 或注册全局守卫 |
| 生成 `name`、`meta`、`props`、`alias`、`redirect` | 登录态、权限判定与菜单渲染 |
| 校验 root 唯一性、路径归属冲突 | 运行时按权限动态加载组件 |
| 输出 `virtual:router`（Layout 静态导入 + 页面动态 import） | 登录页、403/404、根重定向等手工路由 |

`meta.permission` 只是给应用守卫读取的声明，前端权限不是安全边界，后端必须再次校验。

## 工作流程

```text
配置 routeRoots
  → 递归收集 .vue / .tsx / .jsx（跳过 components 目录与保留文件名）
  → 读取页面 <router lang="json"> 与目录 routeMeta.json
  → 计算 parts = basePath 分段 + 页面相对路径分段，解析 param / alias
  → 按路径建树，校验 moduleId 与路径归属，合并 meta
  → 序列化为 virtual:router：import Layout + export default 路由数组
```

对应的 Vite 钩子：

| 钩子 | 行为 |
| --- | --- |
| `resolveId` | 把 `virtual:router` 解析为内部 id `\0virtual:router` |
| `load` | 生成模块源码：`import Layout from "<layoutDir>"` + `export default <路由数组>` |
| `handleHotUpdate` | 任一 root 目录下文件变化时，失效虚拟模块并 `full-reload` |

页面组件使用 `() => import("/绝对/路径/Page.vue")`，生产构建会自动拆成异步 chunk；`Layout` 是同一 SPA 内所有生成路由共享的顶层容器。

## 接入

`@unoadmin/vite-plugins` 应作为应用壳的 devDependency（`workspace:*`）。以下示例摘自 [`../../admin-app/vite.config.ts`](../../admin-app/vite.config.ts)：

```ts
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'

import { routesGeneratorPlugin } from '@unoadmin/vite-plugins/routes-generator'

const workspaceRoot = fileURLToPath(new URL('../../..', import.meta.url))
const adminBaseViews = fileURLToPath(new URL('../admin-base/src/views', import.meta.url))
const demoViews = fileURLToPath(new URL('../domain-demo/src/views', import.meta.url))
const appViews = fileURLToPath(new URL('./src/views', import.meta.url))

export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    routesGeneratorPlugin({
      layoutDir: fileURLToPath(new URL('../admin-base/src/views/Layout.vue', import.meta.url)),
      routeRoots: [
        { moduleId: 'admin-base', directory: adminBaseViews, basePath: '' },
        { moduleId: 'demo', directory: demoViews, basePath: '/demo' },
        { moduleId: 'app', directory: appViews, basePath: '' },
      ],
    }),
  ],
  server: {
    fs: { allow: [workspaceRoot] }, // 跨 package 的绝对路径必须放行
  },
})
```

> `server.fs.allow` 不可省略：生成的动态 import 指向 workspace 中其它 package 的绝对文件路径，默认 Vite 文件访问策略会拦截它们。

### 配置

`routesGeneratorPlugin(options?: Options)` 与 `getGeneratedRouteRecords(options?: Options)` 共用同一份 `Options`：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `layoutDir` | `string` | `/src/views/Layout.vue` | Layout 组件的**导入说明符**，直接写进生成模块的 `import` 语句；可用绝对路径或 Vite 根相对路径 `/src/...`，必须能被 Vite 解析为 Vue 组件 |
| `routeRoots` | `RouteRoot[]` | 单 root：`{ moduleId: 'app', directory: <cwd>/src/views, basePath: '' }` | 省略或传空数组时兼容旧行为，只扫当前工作目录的 `src/views` |

`RouteRoot`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `moduleId` | `string` | 是 | 模块标识，全局唯一；用于路由 `name` 前缀和路径归属校验 |
| `directory` | `string` | 是 | 该模块的 `src/views` 目录，**务必传绝对路径**（`fileURLToPath(new URL(...))`） |
| `basePath` | `string` | 否 | URL 前缀，如 `/demo`；页面路径先按相对 root 的路径生成，再拼接前缀 |

> `directory` 用相对路径时，生成器内部的字符串替换依赖前缀完全一致；以 `./` 开头的相对路径会解析错文件。统一使用绝对路径可避开这个陷阱（单测中形如 `../admin-base/src/views` 的路径同样可用）。

启动与生成阶段会做以下校验，失败时直接抛错并中断构建：

| 错误信息 | 触发条件 |
| --- | --- |
| `Duplicate route moduleId: X` | 两个 root 使用相同 `moduleId` |
| `Route root does not exist: X` | `directory` 不存在 |
| `Route path X is declared by both A and B` | 不同 `moduleId` 声明了同一路径 |
| `Route file X must resolve to a non-empty path` | 解析后路径为空（例如无 `basePath` 的 root 顶层放 `index.vue`） |
| `<file> must contain a JSON object` | `routeMeta.json` 不是 JSON 对象；`<router lang="json">` 内容非法 JSON 同样会抛错 |

## 使用 virtual:router

在应用 [`env.d.ts`](../../admin-app/env.d.ts) 中声明类型：

```ts
declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
```

在 [`router/index.ts`](../../admin-app/src/router/index.ts) 中只保留手工路由，其余交给生成表：

```ts
import { createRouter, createWebHistory } from 'vue-router'
import generatedRoutes from 'virtual:router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/signin', name: 'signin', component: () => import('../views/Signin.vue') },
    ...generatedRoutes,
    { path: '/403', name: 'forbidden', component: () => import('../views/403.vue') },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../views/404.vue') },
  ],
})
```

`Layout.vue`、`Signin.vue`、`403.vue`、`404.vue` 会被扫描器跳过，正是为了让登录、错误页这类手工路由不与自动路由重复。守卫在应用侧安装，见 [`router/guards.ts`](../../admin-app/src/router/guards.ts)：

```ts
// router.beforeEach((to) => { ... }) 内部
const requiredPermission = to.meta.permission
if (typeof requiredPermission !== 'string' || runtime.can(requiredPermission)) return true
return { name: 'forbidden' }
```

## 页面 → 路由映射

### 扫描与排除

递归收集 `.vue`、`.tsx`、`.jsx` 文件，但跳过：

- 任意层级名为 `components` 的目录（大小写不敏感）。
- 文件名为 `Layout.vue`、`Signin.vue`、`403.vue`、`404.vue`（精确匹配，大小写敏感）。
- 页面 `<router lang="json">` 中声明 `"isRouter": false` 的文件。

同一 root 内文件按相对路径字符串排序后依次生成；不同 root 之间按 `routeRoots` 的配置顺序输出。

### 路径规则

相对 root 的路径去掉扩展名后**整体转小写**（不做分词），`index` 作为末段会被去掉；再与 `basePath` 分段拼接。下表前三行为本仓库真实页面，其余为规则示例：

| 页面文件 | root / basePath | 生成 `path` | 说明 |
| --- | --- | --- | --- |
| `admin-base/src/views/Dashboard.vue` | `admin-base` / `''` | `/dashboard` | 一级页面：Layout + `path: ''` 的 index 子路由 |
| `admin-base/src/views/System/Users.vue` | `admin-base` / `''` | `/system/users` | 二级页面：`/system` 容器下的子路由 |
| `domain-demo/src/views/Records.vue` | `demo` / `'/demo'` | `/demo/records` | 前缀由 root 提供，页面文件名不再写 `demo` |
| `System/UserProfile.vue` | 无前缀 | `/system/userprofile` | PascalCase 只转小写连写，同时生成 alias `/system/user-profile` |
| `System/Detail.vue` + `"param": ":id"` | 无前缀 | `/system/detail/:id` | 参数段追加到叶子路径（及 alias）末尾 |
| `System/index.vue` | 无前缀 | `/system` 的 `path: ''` 子路由 | 该分组不再生成 `redirect` |

命名与结构约定：

| 项 | 取值 |
| --- | --- |
| `name` | `moduleId` + `-` + 各路径分段用 `-` 连接，index 子路由再加 `-index`；例如 `demo-demo-records` |
| `component` | 一级容器为 `Layout`；页面为动态 import |
| `props` | 全部为 `true`，路由参数作为 props 传入页面 |
| `children` | 多级路由的每一段都会建成容器记录；一级容器挂 Layout |
| `redirect` | 多级分组在没有 index 子路由时指向该分组下**按相对路径排序后最后处理**的叶子页面；放一个 `index.vue` 可关闭它 |
| `alias` | 仅当未显式指定 `routePath` 且 kebab 形式与规范路径不同时生成，值为 kebab-case 路径 |

`/system` 分组在本仓库生成的真实结构（省略绝对路径）：

```js
{
  path: '/system',
  name: 'admin-base-system',
  meta: { title: 'system', permission: 'system:view' },
  props: true,
  component: Layout,
  redirect: '/system/users',
  children: [
    {
      path: '/system/users',
      name: 'admin-base-system-users',
      meta: { title: '用户管理', permission: 'user:read' },
      props: true,
      component: () => import('<绝对路径>/System/Users.vue'),
    },
    // ...permissions / roles / tenants
  ],
}
```

因 PascalCase 会得到 `/system/userprofile` 这类路径，需要稳定、可读的对外 URL 时，优先使用全小写文件名或显式 `routePath` 固定规范路径；对外链接请统一使用小写形式。

## 页面元信息

生成路由的 `meta` 有两个来源：页面级 `<router lang="json">` 和目录级 `routeMeta.json`。除页面级的三个生成字段外，两者都只写入 `meta`，不参与组件加载。

### 页面级：`<router lang="json">`

页面顶部的自定义块会被解析为 JSON（见 [`System/Users.vue`](../../admin-base/src/views/System/Users.vue)）：

```vue
<router lang="json">
{
  "title": "用户管理",
  "permission": "user:read"
}
</router>
```

| 字段 | 作用 |
| --- | --- |
| `isRouter` | 为 `false` 时整个文件不生成路由 |
| `param` | 追加到多级叶子路径末尾的参数段，通常写 `":id"`；一级页面不生效 |
| `routePath` | 覆盖按文件名推导的局部路径；仍会与 `basePath` 拼接、不做小写化、且**不再生成 kebab alias** |
| 其它字段 | 原样写入该页面路由的 `route.meta`，如 `title`、`permission` |

`routePath` 是**相对 root 的局部片段**。带 `basePath: '/demo'` 的 root 中应写 `"/records"` 得到 `/demo/records`；写成 `"/demo/records"` 会得到 `/demo/demo/records`。

### 目录级：`routeMeta.json`

**作用**：给「某个目录对应的路由容器」提供一组 `meta` 覆盖值，可声明分组级标题、权限码或自定义展示字段（消费方见下文）。它**不生成路由、不改变路径、不决定组件**，也不是让子页面继承的共享配置——只作用于与该目录同名的那一条容器记录（以及该目录下 `index.vue` 生成的 index 子路由）。

```text
admin-base/src/views/
├── System/
│   ├── routeMeta.json          # key: /system
│   ├── Users.vue
│   └── Settings/
│       ├── routeMeta.json      # key: /system/settings
│       └── Profile.vue
├── Dashboard.vue
└── components/                 # 整个目录被扫描器跳过
```

| 文件位置 | 索引 key | 生效的路由记录 |
| --- | --- | --- |
| `System/routeMeta.json` | `/system` | 容器 `/system`，以及 `System/index.vue` 的 index 子路由 |
| `System/Settings/routeMeta.json` | `/system/settings` | 容器 `/system/settings`，以及该目录 `index.vue` 的 index 子路由 |
| `<root>/routeMeta.json` | `/` | 无对应记录，生成器不产生根路径路由 |
| `components/routeMeta.json` | — | 扫描时跳过整个 `components` 目录 |

key 由「相对该 root 的目录路径」转小写得到，因此 `System/routeMeta.json` 能匹配已小写化的 `/system`。文件必须是 JSON 对象，数组或 `null` 会抛 `<file> must contain a JSON object`；`.json` 不是页面文件，本身不会生成路由；改动会触发 `full-reload`（见「热更新与测试」）。

| 写入字段 | 行为 |
| --- | --- |
| `title` | 覆盖容器默认的 `parts.join('.')` |
| `permission` | 覆盖容器默认权限码，应用守卫读取 `meta.permission` |
| 任意自定义字段 | 原样写入 `meta`，如 `icon`、`group`、`hidden`、`keepAlive`；生成器不做字段白名单 |
| `isRouter` / `param` / `routePath` | **只当作普通 meta 字段**。这三个生成字段仅在页面 `<router>` 块中被解构，写在目录配置里不会屏蔽路由或改变路径 |

**应用侧消费**：`meta` 只是声明，插件不渲染任何 UI。本仓库目前只有 [`router/guards.ts`](../../admin-app/src/router/guards.ts) 消费 `meta.permission`；Layout 的菜单来自模块的 `navigation` 声明（`runtime.getNavigation()`），`title` 与自定义字段需要应用自行读取（面包屑、页签、文档标题等）。子页面要自定义访问控制，只能在页面 `<router>` 块中声明 `permission`；不声明时使用默认 `parts.join(':') + ':view'`。

**生效示例**（以下均取自本插件的实际输出）：

`System/routeMeta.json` 内容：

```json
{ "title": "系统管理", "permission": "system:read", "icon": "setting" }
```

`System/Users.vue` 的页面块：

```vue
<router lang="json">
{ "title": "用户管理" }
</router>
```

| 生成的记录 | `meta` |
| --- | --- |
| `/system`（容器） | `{ title: '系统管理', permission: 'system:read', icon: 'setting' }` |
| `/system/users`（子页面） | `{ title: '用户管理', permission: 'system:users:view' }` |

容器拿到了目录配置；子页面**没有继承** `icon` 和 `permission`，只保留页面自己的 `title` 与默认 `permission`。当目录配置和页面配置落在同一条记录上时，叠加关系才体现出来：`Dashboard.vue`（`<router>` 写 `title: '页面标题'`）配合 `Dashboard/routeMeta.json`（`title: '目录标题'`、`icon: 'home'`），容器 `meta` 为目录标题，而 `/dashboard` 的 index 子路由为 `{ title: '页面标题', permission: 'dashboard:view', icon: 'home' }`——页面覆盖 `title`，目录的 `icon` 仍保留。

### `meta` 合并规则

同一条路由记录上的覆盖顺序（后者覆盖前者）：

```text
默认值  →  目录 routeMeta.json  →  页面 <router lang="json"> 的其它字段
```

| 默认项 | 计算方式 | 示例（`/system/users`） |
| --- | --- | --- |
| `title` | `parts.join('.')` | `system.users` |
| `permission` | `parts.join(':') + ':view'` | `system:users:view` |

注意：目录配置与页面配置通常作用于**不同记录**（容器 vs 叶子），此时不存在覆盖关系；仅写 `title` 的页面配置不会移除默认 `permission`。想确认某个目录配置是否生效，用 `getGeneratedRouteRecords()` 得到 `routes` 后打印 `routes.find(r => r.path === '/system')?.meta` 即可（用法见「热更新与测试」）。

## 与前端导航菜单结合

生成器只产出路由表，**不生成菜单**。菜单属于模块协议：模块在 `navigation` 中声明条目，应用创建 runtime 时注册模块，`runtime.getNavigation()` 按当前权限过滤后交给 Layout 渲染。路由与菜单各自声明，靠 **path 对齐**。

```text
模块 navigation 声明
  → createAdminRuntime({ modules })
  → runtime.getNavigation()：can(permission) 过滤，丢弃空分组
  → Layout.vue 侧边栏（v-for key=id；isActive 用 route.path 匹配）
路由侧：virtual:router → createRouter → 守卫读 route.meta.permission
```

不让生成器直接生成菜单，是因为路由粒度不等于导航粒度：详情页、编辑页有路由但不该进菜单，`isRouter: false` 的页面则连路由都不生成；分组、排序、图标等属于业务语义，不在文件路径里；导航还需要与模块权限声明一起评审，避免新增页面绕过菜单和权限评审。

### 菜单条目结构

`NavigationItem` 定义在 [`core/src/composition.ts`](../../core/src/composition.ts)：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | `string` | 是 | 列表渲染的 key；顶层项在所有模块合并后应唯一 |
| `label` | `string` | 是 | 菜单文案，与 `route.meta.title` 相互独立 |
| `to` | `string` | 否 | 目标路径，应等于生成路由的 `path`（含 `basePath`）；纯分组项不写 |
| `permission` | `string` | 否 | 控制菜单可见性；`can()` 支持 `*` 通配；与路由 `meta.permission` 是两套声明 |
| `children` | `NavigationItem[]` | 否 | 子菜单；自身权限不通过时整组移除，子项全被移除且无 `to` 时分组也移除 |

本仓库的声明见 [`admin-base/src/index.ts`](../../admin-base/src/index.ts) 和 [`domain-demo/src/index.ts`](../../domain-demo/src/index.ts)：

```ts
navigation: [
  { id: 'dashboard', label: '工作台', to: '/dashboard' },
  {
    id: 'system',
    label: '系统管理',
    children: [
      { id: 'users', label: '用户管理', to: '/system/users', permission: 'user:read' },
      // ...
    ],
  },
]
```

### 菜单与生成路由的对照

| 菜单项 | `to` | 页面文件 | 生成的 `path` |
| --- | --- | --- | --- |
| 工作台 | `/dashboard` | `admin-base/Dashboard.vue` | `/dashboard` |
| 用户管理 | `/system/users` | `admin-base/System/Users.vue` | `/system/users` |
| 业务概览 | `/demo/overview` | `domain-demo/Overview.vue` | `/demo/overview` |
| 项目记录 | `/demo/records` | `domain-demo/Records.vue` | `/demo/records` |

`to` 写**生成后的 URL**：`basePath` 由 root 提供，不能省掉 `/demo`；也不要写 `System/Users.vue` 这类文件路径或 kebab 别名形式。

### 对齐规则与常见错位

| 规则 | 说明 |
| --- | --- |
| 路径对齐 | `to` 与生成 `path` 必须完全一致；不一致时点击匹配不到路由，会落到全局 404 |
| 分组不是路由 | 有 `children`、无 `to` 的项只是菜单分组；若该目录下还有页面，生成器会额外生成 `/system` 容器并 `redirect`，两者不是同一概念 |
| 顺序独立 | 菜单顺序 = 模块注册顺序 + 声明顺序；生成器的文件扫描顺序不影响菜单 |
| 新增页面要补菜单 | 只加 `.vue` 会有路由无入口；只加菜单不建页面会 404，建议用下方单测兜住 |
| 两套权限 | 菜单 `permission` 控制可见性，路由 `meta.permission` 控制可访问性，两处应使用同一个权限码 |
| 高亮按 path | [`Layout.vue`](../../admin-base/src/views/Layout.vue) 的 `isActive` 用 `route.path` 与 `to` 做等值/前缀比较；通过 alias 访问时不命中 canonical `to`，菜单不会高亮 |
| 文案两处维护 | 菜单文案是 `label`，不读取 `meta.title`；改标题不会同步菜单 |

菜单隐藏从来不是访问控制：路由守卫只读 `to.meta.permission`。例如新增 `System/Audit.vue` 未写 `<router>` 时，路由权限是默认的 `system:audit:view`，而菜单若声明 `audit:read`：

- 用户只有 `audit:read`：菜单可见，点击被守卫拦到 403；
- 用户只有 `system:audit:view`：菜单不可见，但直接输入 URL 可以进入。

因此两处应引用同一个权限码（通常来自模块的 `permissions` 声明）；以路由 `meta.permission` 为准生成或校验菜单更稳妥。

### 一致性校验

`getGeneratedRouteRecords()` 也可以在单测里校验「菜单指向的路由都存在」：

```ts
import type { NavigationItem } from '@unoadmin/core'
import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'

const routes = getGeneratedRouteRecords({ routeRoots: [/* 与 vite.config.ts 中的三个 root 保持一致，不能留空 */] })
const paths = new Set<string>()
const collect = (records: typeof routes) => records.forEach((record) => {
  if (record.path) paths.add(record.path)
  if (record.children) collect(record.children)
})
collect(routes)

const flatten = (items: NavigationItem[]): NavigationItem[] =>
  items.flatMap((item) => [item, ...(item.children ? flatten(item.children) : [])])

for (const item of flatten(modules.flatMap((module) => module.navigation))) {
  if (item.to) expect(paths.has(item.to)).toBe(true)
}
```

`modules` 是应用组合根导出的模块清单（见 [`admin-app/src/modules.ts`](../../admin-app/src/modules.ts)）。同一循环里还可比对 `path` 对应的 `meta.permission` 与菜单 `permission`，防止权限码错位。

### 当前实现的边界

- Layout 只渲染 `label` 与 `to`，没有 icon、badge、排序权重字段；需要时同时扩展 `NavigationItem` 与 Layout。
- 菜单不读 `route.meta`：`title`、`icon` 等目录元信息不会自动出现在菜单里。
- `children` 子项漏写 `to` 时，Layout 会把它渲染成指向 `/404` 的链接（`child.to || '/404'`）；这只是兜底，不应作为正常配置。
- 面包屑和 `document.title` 尚未实现，应用侧可读取 `useRoute().meta.title` 自行实现。
- 后端下发的远端菜单只能过滤/排序已声明条目，不能凭返回值加载未知组件（见[架构文档 §6.1](../../../../docs/frontend-architecture.md)）。

## 热更新与测试

- `routeRoots` 目录下**任意**文件变化（含 `components/` 内文件、`routeMeta.json`）都会使 `virtual:router` 失效并触发整页刷新。
- 新建目录若开发服务未感知到文件系统事件，重启 Vite 即可；生产构建每次都会重新扫描。
- `getGeneratedRouteRecords(options)` 是同步生成函数，不依赖 Vite 运行时，适合在单测中锁定路径契约（见 [`App.spec.ts`](../../admin-app/src/__tests__/App.spec.ts)）：

```ts
import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'

const routes = getGeneratedRouteRecords({
  routeRoots: [
    { moduleId: 'admin-base', directory: adminBaseViews, basePath: '' },
    { moduleId: 'demo', directory: demoViews, basePath: '/demo' },
  ],
})
```

建议在 CI 中执行（脚本来自根 [`package.json`](../../../../package.json) 与各 package）：

```bash
pnpm --filter @unoadmin/vite-plugins type-check   # 插件自身类型检查
pnpm typecheck                                   # 应用类型检查（含 virtual:router 声明）
pnpm test                                        # 单测：生成路由契约 + 守卫
pnpm build                                       # 生产构建，验证跨 package 动态 import
pnpm --filter @unoadmin/admin-app test:e2e --project=chromium  # 深链接直开与刷新
```

端到端测试需先执行 `pnpm exec playwright install chromium`。history 模式要求静态服务器把未知非静态请求回退到同一个 `index.html`（Nginx `try_files $uri $uri/ /index.html;`），详见 [`admin-app/README.md`](../../admin-app/README.md)。

## 已知限制

| 限制 | 规避方式 |
| --- | --- |
| 带 `basePath` 的 root 内，`routeMeta.json` 的 key 不含前缀，而查找使用拼接后的完整路径，因此**不会命中**（详见「目录级 `routeMeta.json`」） | 把分组元信息写进各页面的 `<router lang="json">`；或改用不带 `basePath` 的 root（会改变 URL 结构） |
| `routeMeta.json` 只作用于同名容器，不下发到子页面 | 子页面各自声明 `title` / `permission` |
| `param` 只对多级叶子路由生效 | 需要动态段的页面放在子目录下 |
| canonical 路径是小写连写，PascalCase 只额外提供 kebab alias | 用全小写文件名或显式 `routePath` 固定 URL |
| 插件只在构建期扫描，不处理运行时按后端返回的路径加载组件 | 后端只返回权限码/菜单标识，页面由已注册模块匹配 |
| 导出的是 TS 源码而非构建产物，消费者需能编译 TypeScript | 目前仅用于 workspace 内应用；对外发布需先补构建与 `exports` 产物 |

## 相关文档

- [前端架构设计 §10 路由生成插件](../../../../docs/frontend-architecture.md)
- [admin-app README](../../admin-app/README.md)
