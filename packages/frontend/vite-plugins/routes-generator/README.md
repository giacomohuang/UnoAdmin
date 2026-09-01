# routes-generator

`routes-generator` 是 UnoAdmin SPA 的统一路由生成插件。它扫描应用和 workspace package 的多个 `src/views` 根目录，将页面生成到一个 `virtual:router` 虚拟模块中。应用只创建一个 Vue Router 实例并消费该模块。

## 接入

```ts
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import { routesGeneratorPlugin } from '@unoadmin/vite-plugins/routes-generator'

export default defineConfig({
  plugins: [
    routesGeneratorPlugin({
      layoutDir: fileURLToPath(new URL('../../packages/frontend/admin-base/src/views/Layout.vue', import.meta.url)),
      routeRoots: [
        { moduleId: 'admin-base', directory: fileURLToPath(new URL('../../packages/frontend/admin-base/src/views', import.meta.url)) },
        { moduleId: 'demo', directory: fileURLToPath(new URL('../../packages/frontend/domain-demo/src/views', import.meta.url)), basePath: '/demo' },
        { moduleId: 'app', directory: fileURLToPath(new URL('./src/views', import.meta.url)) },
      ],
    }),
  ],
})
```

### 配置

- `layoutDir`：布局组件的绝对路径，必须是 Vite 可以解析的导入路径。
- `routeRoots`：路由根目录列表。每项包含唯一的 `moduleId`、绝对 `directory` 和可选的 `basePath`。
- 未配置 `routeRoots` 时，插件兼容旧行为，扫描当前工作目录下的 `src/views`。

每个 root 内的页面路径先按相对 root 的路径生成，再追加 `basePath`。因此 `domain-demo/src/views/Overview.vue` 配置 `basePath: '/demo'` 后生成 `/demo/overview`。

## 虚拟模块

在应用的 `env.d.ts` 中声明：

```ts
declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
```

应用使用 history SPA：

```ts
import generatedRoutes from 'virtual:router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/signin', component: Signin },
    ...generatedRoutes,
    { path: '/:pathMatch(.*)*', component: NotFound },
  ],
})
```

插件不创建 Router、不注册守卫，也不负责过滤权限。认证和权限守卫由应用组合根安装。

## 页面规则

插件递归扫描 `.vue`、`.tsx`、`.jsx` 文件，并跳过：

- 任意层级名为 `components` 的目录。
- `Layout.vue`、`Signin.vue`、`403.vue`、`404.vue`。
- 页面 `<router lang="json">` 中设置 `"isRouter": false` 的文件。

页面可以通过 `<router lang="json">` 设置：

```vue
<router lang="json">
{
  "title": "项目记录",
  "permission": "demo:read",
  "routePath": "/demo/records"
}
</router>
```

支持的生成字段包括 `isRouter`、`param` 和 `routePath`；其它字段写入 `route.meta`。目录中的 `routeMeta.json` 可以提供共享元信息，页面配置优先级更高。

页面和目录建议使用 PascalCase，外部 URL 使用小写路径。生成器会对每个页面使用动态 import，并对不同 `moduleId` 的 route name 加模块前缀，避免多个 package 使用相同页面名时发生冲突。

## 热更新和检查

配置 root 下新增或修改页面时，插件会让 `virtual:router` 失效并触发页面刷新。添加新目录后如开发服务没有感知到文件系统变化，重启 Vite 即可。

建议在 CI 中执行：

```bash
pnpm --filter @unoadmin/vite-plugins type-check
pnpm --filter @unoadmin/admin-app type-check
pnpm --filter @unoadmin/admin-app build
```

需要验证 SPA 深链接时，直接访问和刷新 `/demo/overview`、`/system/users` 等 URL，并确保静态服务器将未知非静态请求回退到同一个 `index.html`。
