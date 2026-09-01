import type { Router } from 'vue-router'

import type { AdminRuntime } from '@unoadmin/runtime'

/** Install the application-level auth and permission boundary around generated routes. */
export function installRouterGuards(router: Router, runtime: AdminRuntime) {
  router.beforeEach((to) => {
    if (to.name === 'signin') return true
    if (!runtime.state.user) return { name: 'signin', query: { redirect: to.fullPath } }

    const requiredPermission = to.meta.permission
    if (typeof requiredPermission !== 'string' || runtime.can(requiredPermission)) return true
    return { name: 'forbidden' }
  })
}
