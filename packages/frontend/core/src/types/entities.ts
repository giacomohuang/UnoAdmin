import type { AccountId, LegalEntityId, OrgId, PermissionId, RoleId, TenantId } from './identifiers'

/** 租户形态：集团统一采购，或单体公司单独采购。 */
export type TenantMode = 'group' | 'standalone'

/** 数据可见范围：整个租户、被授权主体，或本人组织。 */
export type DataScope = 'tenant' | 'entity' | 'self'

/** 经营主体类型：集团总部或独立经营公司。 */
export type LegalEntityType = 'GROUP' | 'COMPANY'

/** 通用状态值，租户、主体与角色共用。 */
export type LifecycleStatus = 'active' | 'suspended'

/**
 * 租户：采购、计费与数据隔离的最外层边界。
 *
 * - `group` 集团租户下可以有多个 `LegalEntity`，集团层面可统一查看数据；
 * - `standalone` 单体租户下有且只有一个 `LegalEntity`。
 */
export interface Tenant {
  id: TenantId
  name: string
  code: string
  mode: TenantMode
  plan: string
  contact: string
  status: LifecycleStatus
  entityCount?: number
  accountCount?: number
  legalEntities?: LegalEntity[]
}

/** 经营主体：租户内部相互隔离的经营实体。 */
export interface LegalEntity {
  id: LegalEntityId
  tenantId: TenantId
  parentId: string
  type: LegalEntityType
  name: string
  code: string
  address: string
  status: LifecycleStatus
}

/** 组织节点：经营主体内部的部门、门店、项目组等结构。 */
export interface OrgUnit {
  id: OrgId
  tenantId: TenantId
  legalEntityId: LegalEntityId
  parentId: string
  name: string
  type: string
  path: string[]
  sort: number
  status: LifecycleStatus
  children?: OrgUnit[]
}

/** 管理端账户，携带租户、主体与组织授权范围。 */
export interface Account {
  id: AccountId
  username: string
  displayName: string
  email: string
  phone: string
  status: 'active' | 'disabled'
  tenantId: TenantId
  dataScope: DataScope
  platformAdmin: boolean
  legalEntityIds: LegalEntityId[]
  orgIds: OrgId[]
  roleIds: RoleId[]
  roleNames: string[]
  legalEntityNames: string[]
}

/** 角色及其权限码与数据范围。 */
export interface Role {
  id: RoleId
  tenantId: TenantId
  legalEntityId: string
  code: string
  name: string
  description: string
  status: LifecycleStatus
  dataScope: DataScope
  permissionCodes: string[]
  userCount: number
  builtIn: boolean
}

/** 权限节点，可渲染为树。 */
export interface Permission {
  id: PermissionId
  name: string
  code: string
  type: 'menu' | 'page' | 'action'
  scope: 'platform' | 'tenant' | 'entity'
  children?: Permission[]
}
