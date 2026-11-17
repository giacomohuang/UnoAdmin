import type { AdminStats, ListQuery, PageResult } from './query'
import type { Account, LegalEntity, OrgUnit, Permission, Role, Tenant } from './types/entities'
import type { AccountId, LegalEntityId, OrgId, PermissionId, RoleId, TenantId } from './types/identifiers'

/** 可复用基础管理包所需的全部服务端口。 */
export interface BaseServices {
  accounts: AccountService
  roles: RoleService
  permissions: PermissionService
  tenants: TenantService
  legalEntities: LegalEntityService
  orgs: OrgService
  stats: StatsService
}

/** 账户服务契约。 */
export interface AccountService {
  list(query: ListQuery): Promise<PageResult<Account>>
  create(input: SaveAccount): Promise<Account>
  update(id: AccountId, input: SaveAccount): Promise<Account>
  updateStatus(id: AccountId, status: Account['status']): Promise<void>
}

/** 角色服务契约。 */
export interface RoleService {
  list(query: ListQuery): Promise<PageResult<Role>>
  create(input: SaveRole): Promise<Role>
  update(id: RoleId, input: SaveRole): Promise<Role>
}

/** 权限服务契约。 */
export interface PermissionService {
  list(): Promise<Permission[]>
  create(input: SavePermission): Promise<Permission>
  update(id: PermissionId, input: SavePermission): Promise<Permission>
  remove(id: PermissionId): Promise<void>
  /** 按拖拽结果重写同级权限顺序，返回最新权限树。 */
  reorder(input: ReorderPermissions): Promise<Permission[]>
}

/** 租户服务契约。 */
export interface TenantService {
  list(query: ListQuery): Promise<PageResult<Tenant>>
  create(input: SaveTenant): Promise<Tenant>
  update(id: TenantId, input: SaveTenant): Promise<Tenant>
}

/** 经营主体服务契约。 */
export interface LegalEntityService {
  list(query: ListQuery): Promise<LegalEntity[]>
  create(input: SaveLegalEntity): Promise<LegalEntity>
  update(id: LegalEntityId, input: SaveLegalEntity): Promise<LegalEntity>
}

/** 组织架构服务契约。 */
export interface OrgService {
  tree(legalEntityId: string): Promise<OrgUnit[]>
  create(input: SaveOrg): Promise<OrgUnit>
  update(id: OrgId, input: SaveOrg): Promise<OrgUnit>
}

/** 工作台指标服务契约。 */
export interface StatsService {
  get(): Promise<AdminStats>
}

/** 新建或更新租户允许提交的字段。 */
export interface SaveTenant {
  name: string
  code: string
  mode: Tenant['mode']
  plan: string
  contact: string
  status: Tenant['status']
}

/** 新建或更新经营主体允许提交的字段。 */
export interface SaveLegalEntity {
  tenantId: string
  parentId: string
  type: LegalEntity['type']
  name: string
  code: string
  address: string
  status: LegalEntity['status']
}

/** 新建或更新组织节点允许提交的字段。 */
export interface SaveOrg {
  legalEntityId: string
  parentId: string
  name: string
  type: string
  sort: number
  status: OrgUnit['status']
}

/** 新建或更新账户允许提交的字段。 */
export interface SaveAccount {
  username: string
  displayName: string
  password?: string
  email: string
  phone: string
  status: Account['status']
  dataScope: Account['dataScope']
  legalEntityIds: string[]
  orgIds: string[]
  roleIds: string[]
}

/** 新建或更新角色允许提交的字段。 */
export interface SaveRole {
  code: string
  name: string
  description: string
  status: Role['status']
  dataScope: Role['dataScope']
  legalEntityId: string
  permissionCodes: string[]
}

/** 新建或更新权限节点允许提交的字段。 */
export interface SavePermission {
  name: string
  code: string
  type: Permission['type']
  scope: Permission['scope']
  parentId: string
  sortOrder: number
}

/** 同级权限拖拽排序的请求体。 */
export interface ReorderPermissions {
  parentId: string
  ids: string[]
}
