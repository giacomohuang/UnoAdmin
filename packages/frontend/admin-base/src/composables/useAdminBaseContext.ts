import { inject } from 'vue'

import {
  ADMIN_BRANDING_KEY,
  ADMIN_MODULES_KEY,
  ADMIN_SESSION_KEY,
  type AdminBranding,
  type AdminModule,
  type AdminSession,
  type NavigationItem
} from '@unoadmin/core'

/** 可复用基础外壳消费的应用上下文。 */
export interface AdminBaseContext {
  session: AdminSession
  branding: AdminBranding
  modules: readonly AdminModule[]
  /** 按当前账号权限过滤后的导航树。 */
  getNavigation(): NavigationItem[]
  /** 找到某个路由所属的一级导航项，用于面包屑与二级菜单标题。 */
  resolveNavigation(path: string): { parent?: NavigationItem; current?: NavigationItem }
}

/** 只组合 admin-base 需要的应用上下文。 */
export function useAdminBaseContext(): AdminBaseContext {
  const session = inject(ADMIN_SESSION_KEY) as AdminSession | undefined
  const branding = inject(ADMIN_BRANDING_KEY) as AdminBranding | undefined
  const modules = inject(ADMIN_MODULES_KEY) as readonly AdminModule[] | undefined
  if (!session || !branding || !modules) throw new Error('Admin application context is not provided')

  const getNavigation = () => modules.flatMap((module) => filterNavigation(module.navigation, session.can))

  return {
    session,
    branding,
    modules,
    getNavigation,
    resolveNavigation(path) {
      const items = getNavigation()
      for (const item of items) {
        if (samePath(item.path, path)) return { current: item }
        for (const child of item.children ?? []) {
          if (samePath(child.path, path)) return { parent: item, current: child }
        }
      }
      return {}
    }
  }
}

/** 递归过滤导航，保持模块声明的顺序；空分组不展示。 */
function filterNavigation(items: NavigationItem[], can: (permission?: string) => boolean): NavigationItem[] {
  return items.flatMap((item) => {
    if (!can(item.permission)) return []
    const children = item.children ? filterNavigation(item.children, can) : undefined
    if (item.children && !children?.length && !item.path) return []
    return [{ ...item, children }]
  })
}

/** 详情子路径也视为所属导航项激活。 */
function samePath(target: string | undefined, path: string): boolean {
  if (!target) return false
  return path === target || path.startsWith(`${target}/`)
}
