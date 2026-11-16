import type { AdminRuntime } from '@unoadmin/runtime'
import type { Router } from 'vue-router'

/** Install the application-level auth and permission boundary around generated routes. */
export function installRouterGuards(router: Router, runtime: AdminRuntime) {
  router.beforeEach((to) => {
    // Auth targets stay path-based so generated route names can change with namePrefix.
    if (to.path === '/signin') return true
    if (!runtime.state.user) return { path: '/signin', query: { redirect: to.fullPath } }

    const requiredPermission = to.meta.permission
    if (typeof requiredPermission !== 'string' || runtime.can(requiredPermission)) return true
    return { path: '/403' }
  })
}
