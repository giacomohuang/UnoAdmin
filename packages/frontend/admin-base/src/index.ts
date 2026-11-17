import type { AdminModule, PermissionDefinition } from '@unoadmin/core'

/**
 * 基础管理模块发布的权限目录。
 *
 * 权限码与后端 seed 数据保持一致；页面路由的 `permission` 元数据引用同一批编码，
 * 因此角色授权变更后菜单、路由与按钮会同步收敛。
 */
const basePermissions: PermissionDefinition[] = [
  { code: 'dashboard:view', label: '查看工作台' },
  { code: 'tenant:view', label: '查看租户', description: '平台侧查看租户与采购形态' },
  { code: 'tenant:create', label: '新建租户' },
  { code: 'tenant:update', label: '编辑租户', description: '调整租户形态、套餐与状态' },
  { code: 'entity:view', label: '查看经营主体' },
  { code: 'entity:create', label: '新建经营主体' },
  { code: 'entity:update', label: '编辑经营主体' },
  { code: 'org:view', label: '查看组织架构' },
  { code: 'org:create', label: '新建组织节点' },
  { code: 'org:update', label: '编辑组织节点' },
  { code: 'account:view', label: '查看账户' },
  { code: 'account:create', label: '新建账户' },
  { code: 'account:update', label: '编辑账户', description: '调整数据范围与角色授权' },
  { code: 'role:view', label: '查看角色' },
  { code: 'role:create', label: '新建角色' },
  { code: 'role:update', label: '角色授权' },
  { code: 'permission:view', label: '查看权限目录' }
]

/**
 * 创建可复用的基础管理模块。
 *
 * 一级导航只承接二级菜单，二级导航直接对应页面路由，
 * 与 UnoMap 管理端的「图标栏 + 子菜单」交互保持一致。
 */
export function createAdminBaseModule(): AdminModule {
  return {
    id: 'admin-base',
    routeNamespace: 'admin-base',
    navigation: [
      {
        id: 'dashboard',
        title: '工作台',
        path: '/dashboard',
        icon: 'lucide:layout-dashboard',
        permission: 'dashboard:view'
      },
      {
        id: 'system',
        title: '系统管理',
        icon: 'lucide:settings-2',
        children: [
          {
            id: 'tenants',
            title: '租户管理',
            path: '/system/tenants',
            icon: 'lucide:briefcase-business',
            permission: 'tenant:view'
          },
          {
            id: 'entities',
            title: '经营主体',
            path: '/system/entities',
            icon: 'lucide:building-2',
            permission: 'entity:view'
          },
          {
            id: 'organizations',
            title: '组织架构',
            path: '/system/organizations',
            icon: 'lucide:network',
            permission: 'org:view'
          },
          {
            id: 'accounts',
            title: '账户管理',
            path: '/system/accounts',
            icon: 'lucide:users',
            permission: 'account:view'
          },
          {
            id: 'roles',
            title: '角色管理',
            path: '/system/roles',
            icon: 'lucide:shield-check',
            permission: 'role:view'
          },
          {
            id: 'permissions',
            title: '权限管理',
            path: '/system/permissions',
            icon: 'lucide:key-round',
            permission: 'permission:view'
          }
        ]
      }
    ],
    permissions: basePermissions
  }
}

export { default as AdminLayout } from './views/Layout.vue'
