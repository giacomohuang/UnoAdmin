import type { AuthPort, SignInCredentials } from './auth'
import type { BaseServices } from './services'
import type { Account, DataScope, LegalEntity, Tenant } from './types/entities'
import type { LegalEntityId, TenantId } from './types/identifiers'

/**
 * 模块声明的一级/二级导航项。
 *
 * 一级项通常带 `icon` 但不需要 `path`（承接二级菜单），
 * 二级项带 `path` 直接导航；整体按 `permission` 过滤。
 */
export interface NavigationItem {
  id: string
  title: string
  path?: string
  icon?: string
  permission?: string
  children?: NavigationItem[]
}

/** 模块发布的权限元数据。 */
export interface PermissionDefinition {
  code: string
  label: string
  description?: string
}

/** 应用组装根消费的包级契约。 */
export interface AdminModule {
  id: string
  routeNamespace: string
  navigation: NavigationItem[]
  permissions: PermissionDefinition[]
  install?: (context: AdminModuleContext) => void | (() => void)
}

/** 可选的模块安装钩子上下文。 */
export interface AdminModuleContext {
  services: unknown
  permissions: ReadonlySet<string>
}

/** 所有服务由应用提供，包本身保持后端无关。 */
export interface AppServices {
  base: BaseServices
}

/** 基础布局消费的品牌信息。 */
export interface AdminBranding {
  productName: string
  productMark: string
}

/** 应用壳持有的认证、租户与经营主体状态。 */
export interface AdminSessionState {
  ready: boolean
  account: Account | null
  permissions: string[]
  tenants: Tenant[]
  currentTenant: Tenant | null
  legalEntities: LegalEntity[]
  currentLegalEntity: LegalEntity | null
  dataScope: DataScope
}

/** 递归标记会话状态为只读，且不依赖 Vue 类型。 */
export type DeepReadonly<T> = T extends string | number | boolean | bigint | symbol | null | undefined
  ? T
  : T extends (...args: never[]) => unknown
    ? T
    : T extends readonly (infer Item)[]
      ? readonly DeepReadonly<Item>[]
      : T extends object
        ? { readonly [Key in keyof T]: DeepReadonly<T[Key]> }
        : T

/** 应用会话方法，供布局与业务组合式函数消费。 */
export interface AdminSession {
  readonly state: DeepReadonly<AdminSessionState>
  /**
   * 通过配置的认证适配器建立会话。
   * @returns 会话状态更新后完成的 Promise。
   */
  signIn(credentials?: SignInCredentials): Promise<void>
  initialize(): Promise<void>
  /** 判断当前账号是否拥有权限码；`*` 表示全量授权。 */
  can(permission?: string): boolean
  /** 判断当前账号是否可访问指定经营主体。 */
  canAccessEntity(id: string): boolean
  switchTenant(id: TenantId): Promise<void>
  switchLegalEntity(id: LegalEntityId): Promise<void>
  /** 重新读取服务端会话，用于权限或授权范围变更后刷新。 */
  refresh(): Promise<void>
  signOut(): Promise<void>
}

/** 创建应用会话所需的外部依赖。 */
export interface CreateAdminSessionOptions {
  auth: AuthPort
}

/** 应用与业务包共享的注入令牌。 */
export const ADMIN_SESSION_KEY = Symbol.for('unoadmin.admin-session')
export const ADMIN_BRANDING_KEY = Symbol.for('unoadmin.admin-branding')
export const ADMIN_MODULES_KEY = Symbol.for('unoadmin.admin-modules')
export const ADMIN_SERVICES_KEY = Symbol.for('unoadmin.admin-services')
