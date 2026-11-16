/// <reference types="vite/client" />

/** 后端 REST 根地址，默认指向本地 Go 服务的 /api/v1/。 */
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
}

declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
