import type { AdminSession } from '@unoadmin/core'
import type { Router } from 'vue-router'

/** 应用级登录与权限边界，包裹自动生成的路由表。 */
export function installRouterGuards(router: Router, session: AdminSession) {
  router.beforeEach((to) => {
    // 登录页按路径判断，生成的路由名会随 namePrefix 变化。
    if (to.path === '/signin') return true
    if (!session.state.account) return { path: '/signin', query: { redirect: to.fullPath } }

    const requiredPermission = to.meta.permission
    if (typeof requiredPermission !== 'string' || session.can(requiredPermission)) return true
    return { path: '/403' }
  })
}
