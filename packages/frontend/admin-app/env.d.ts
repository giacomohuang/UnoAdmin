/// <reference types="vite/client" />

declare module 'virtual:router' {
  import type { RouteRecordRaw } from 'vue-router'

  const routes: RouteRecordRaw[]
  export default routes
}
