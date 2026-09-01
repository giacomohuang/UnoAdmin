import type {
  AppServices,
  AuthPort,
  AuthSession,
  FileStoragePort,
  ListQuery,
  PageResult,
  Permission,
  Role,
  Tenant,
  User,
} from '@unoadmin/core'
import type { DemoOverview, DemoRecord, DemoServices } from '@unoadmin/domain-demo'

type DemoAppServices = AppServices & { demo: DemoServices }

const tenants: Tenant[] = [
  { id: 'tenant-main' as Tenant['id'], name: '总部空间', code: 'HQ', status: 'active' },
  { id: 'tenant-lab' as Tenant['id'], name: '创新实验室', code: 'LAB', status: 'active' },
  { id: 'tenant-archive' as Tenant['id'], name: '归档空间', code: 'ARCHIVE', status: 'suspended' },
]

const users: User[] = [
  { id: 'user-001' as User['id'], username: 'lin.wei', displayName: '林薇', email: 'lin.wei@example.com', status: 'active', roleNames: ['平台管理员'] },
  { id: 'user-002' as User['id'], username: 'chen.yu', displayName: '陈宇', email: 'chen.yu@example.com', status: 'active', roleNames: ['运营经理'] },
  { id: 'user-003' as User['id'], username: 'zhao.ming', displayName: '赵明', email: 'zhao.ming@example.com', status: 'disabled', roleNames: ['只读访客'] },
]

const roles: Role[] = [
  { id: 'role-admin' as Role['id'], name: '平台管理员', description: '管理全部基础能力和业务模块', permissionCodes: ['*'], userCount: 1 },
  { id: 'role-operator' as Role['id'], name: '运营经理', description: '查看业务数据并执行日常运营操作', permissionCodes: ['user:read', 'role:read', 'tenant:read', 'demo:read'], userCount: 1 },
  { id: 'role-viewer' as Role['id'], name: '只读访客', description: '仅查看被授权的页面', permissionCodes: ['demo:read'], userCount: 1 },
]

const permissions: Permission[] = [
  {
    id: 'permission-system' as Permission['id'],
    name: '系统管理',
    code: 'system',
    type: 'menu',
    children: [
      { id: 'permission-user' as Permission['id'], name: '用户管理', code: 'user:read', type: 'page', children: [{ id: 'permission-user-write' as Permission['id'], name: '编辑用户', code: 'user:write', type: 'action' }] },
      { id: 'permission-role' as Permission['id'], name: '角色管理', code: 'role:read', type: 'page', children: [{ id: 'permission-role-grant' as Permission['id'], name: '角色授权', code: 'role:grant', type: 'action' }] },
      { id: 'permission-tenant' as Permission['id'], name: '租户管理', code: 'tenant:read', type: 'page' },
    ],
  },
  {
    id: 'permission-demo' as Permission['id'],
    name: '示例业务',
    code: 'demo',
    type: 'menu',
    children: [{ id: 'permission-demo-read' as Permission['id'], name: '示例看板', code: 'demo:read', type: 'page' }],
  },
]

const demoOverview: DemoOverview = { activeProjects: 18, pendingTasks: 7, monthlyGrowth: 12.8 }
const demoRecords: DemoRecord[] = [
  { id: 'project-001', name: '春季运营计划', owner: '林薇', status: 'active', updatedAt: '2026-09-01 09:24' },
  { id: 'project-002', name: '服务目录升级', owner: '陈宇', status: 'active', updatedAt: '2026-08-31 16:40' },
  { id: 'project-003', name: '历史数据整理', owner: '赵明', status: 'paused', updatedAt: '2026-08-29 11:08' },
]

const currentSession: AuthSession = {
  user: users[0]!,
  permissions: ['*'],
  tenants,
  currentTenantId: tenants[0]!.id,
}

/** A deterministic local auth adapter lets the skeleton run without a backend. */
export const authAdapter: AuthPort = {
  async signIn() {
    return currentSession
  },
  async restore() {
    return currentSession
  },
  async signOut() {
    // A real adapter would revoke or clear its refresh-token session here.
  },
}

/** All mock services share the same contracts as future REST or GraphQL adapters. */
export const appServices: DemoAppServices = {
  base: {
    users: {
      async list(query) {
        return paginate(users.filter((user) => matches(user, query.search)), query)
      },
      async updateStatus(id, status) {
        const user = users.find((item) => item.id === id)
        if (user) user.status = status
      },
    },
    roles: {
      async list(query) {
        return paginate(roles.filter((role) => matches(role, query.search)), query)
      },
    },
    permissions: {
      async list() {
        return permissions
      },
    },
    tenants: {
      async list(query) {
        return paginate(tenants.filter((tenant) => matches(tenant, query.search)), query)
      },
    },
  },
  storage: createMockStorage(),
  demo: {
    async getOverview() {
      return demoOverview
    },
    async listRecords(query) {
      return paginate(demoRecords.filter((record) => matches(record, query.search)), query)
    },
  },
}

function matches(value: object, search?: string) {
  if (!search) return true
  return Object.values(value).some((item) => String(item).toLowerCase().includes(search.toLowerCase()))
}

function paginate<Item>(items: Item[], query: ListQuery): PageResult<Item> {
  const start = (query.page - 1) * query.pageSize
  return { items: items.slice(start, start + query.pageSize), total: items.length }
}

function createMockStorage(): FileStoragePort {
  return {
    async upload({ file, onProgress }) {
      onProgress?.(25)
      await wait(30)
      onProgress?.(70)
      await wait(30)
      onProgress?.(100)
      return {
        file: {
          id: `file-${Date.now()}`,
          name: file.name,
          size: file.size,
          contentType: file.type || 'application/octet-stream',
          url: URL.createObjectURL(file),
        },
      }
    },
  }
}

function wait(milliseconds: number) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds))
}
