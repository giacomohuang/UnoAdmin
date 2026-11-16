import { inject } from 'vue'

import {
  ADMIN_BRANDING_KEY,
  ADMIN_MODULES_KEY,
  ADMIN_SESSION_KEY,
  type AdminBranding,
  type AdminModule,
  type AdminSession,
  type NavigationItem,
} from '@unoadmin/core'

/** Application contexts consumed by the reusable admin-base shell. */
export interface AdminBaseContext {
  session: AdminSession
  branding: AdminBranding
  modules: readonly AdminModule[]
  getNavigation(): NavigationItem[]
}

/** Compose only the application contexts needed by admin-base. */
export function useAdminBaseContext(): AdminBaseContext {
  const session = inject(ADMIN_SESSION_KEY) as AdminSession | undefined
  const branding = inject(ADMIN_BRANDING_KEY) as AdminBranding | undefined
  const modules = inject(ADMIN_MODULES_KEY) as readonly AdminModule[] | undefined
  if (!session || !branding || !modules) throw new Error('Admin application context is not provided')

  return {
    session,
    branding,
    modules,
    getNavigation() {
      return modules.flatMap((module) => filterNavigation(module.navigation, session.can))
    },
  }
}

/** Filter navigation recursively while preserving declared ordering. */
function filterNavigation(items: NavigationItem[], can: (permission?: string) => boolean): NavigationItem[] {
  return items.flatMap((item) => {
    if (!can(item.permission)) return []
    const children = item.children ? filterNavigation(item.children, can) : undefined
    if (item.children && !children?.length && !item.to) return []
    return [{ ...item, children }]
  })
}
