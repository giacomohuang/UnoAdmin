import type { AdminModule, PermissionDefinition } from '@unoadmin/core'

export * from './runtime'

const basePermissions: PermissionDefinition[] = [
  { code: 'user:read', label: '查看用户' },
  { code: 'user:write', label: '管理用户' },
  { code: 'role:read', label: '查看角色' },
  { code: 'role:grant', label: '分配角色权限' },
  { code: 'permission:read', label: '查看权限' },
  { code: 'tenant:read', label: '查看租户' },
  { code: 'tenant:manage', label: '管理租户' },
]

/** Create the reusable users, authorization, tenant, and Layout module. */
export function createAdminBaseModule(): AdminModule {
  return {
    id: 'admin-base',
    routeNamespace: 'admin-base',
    navigation: [
      { id: 'dashboard', label: '工作台', to: '/dashboard' },
      {
        id: 'system',
        label: '系统管理',
        children: [
          { id: 'users', label: '用户管理', to: '/system/users', permission: 'user:read' },
          { id: 'roles', label: '角色管理', to: '/system/roles', permission: 'role:read' },
          { id: 'permissions', label: '权限管理', to: '/system/permissions', permission: 'permission:read' },
          { id: 'tenants', label: '租户管理', to: '/system/tenants', permission: 'tenant:read' },
        ],
      },
    ],
    permissions: basePermissions,
  }
}

export { default as AdminLayout } from './views/Layout.vue'
