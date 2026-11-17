import type { AdminModule, PermissionDefinition } from '@unoadmin/core'

/**
 * 基础管理模块发布的权限目录。
 *
 * 每个资源只保留三个权限：查看、编辑（包含新增）、删除；
 * 权限码与后端 seed 数据保持一致，页面路由与按钮引用同一批编码。
 */
const basePermissions: PermissionDefinition[] = [
  { code: 'dashboard:view', label: '查看工作台' },
  { code: 'tenant:view', label: '查看租户', description: '平台侧查看租户与采购形态' },
  { code: 'tenant:edit', label: '编辑租户', description: '新建或编辑租户，调整形态、套餐与状态' },
  { code: 'entity:view', label: '查看经营主体' },
  { code: 'entity:edit', label: '编辑经营主体', description: '新建或编辑经营主体' },
  { code: 'org:view', label: '查看组织架构' },
  { code: 'org:edit', label: '编辑组织节点', description: '新建或编辑组织节点' },
  { code: 'account:view', label: '查看账户' },
  { code: 'account:edit', label: '编辑账户', description: '新建或编辑账户，调整数据范围与角色授权' },
  { code: 'role:view', label: '查看角色' },
  { code: 'role:edit', label: '编辑角色', description: '新建或编辑角色，调整权限授权' },
  { code: 'permission:view', label: '查看权限目录' },
  { code: 'permission:edit', label: '编辑权限', description: '新建或编辑权限节点' },
  { code: 'permission:delete', label: '删除权限' }
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
        title: '系统',
        icon: 'lucide:settings-2',
        children: [
          {
            id: 'tenants',
            title: '租户',
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
            title: '账户',
            path: '/system/accounts',
            icon: 'lucide:users',
            permission: 'account:view'
          },
          {
            id: 'roles',
            title: '角色',
            path: '/system/roles',
            icon: 'lucide:shield-check',
            permission: 'role:view'
          },
          {
            id: 'permissions',
            title: '权限',
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
