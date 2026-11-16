import type { AdminSession } from '@unoadmin/core'
import type { Router } from 'vue-router'

/** Install the application-level auth and permission boundary around generated routes. */
export function installRouterGuards(router: Router, session: AdminSession) {
  router.beforeEach((to) => {
    // Auth targets stay path-based so generated route names can change with namePrefix.
    if (to.path === '/signin') return true
    if (!session.state.user) return { path: '/signin', query: { redirect: to.fullPath } }

    const requiredPermission = to.meta.permission
    if (typeof requiredPermission !== 'string' || session.can(requiredPermission)) return true
    return { path: '/403' }
  })
}
