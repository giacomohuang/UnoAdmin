/** A branded string keeps identifiers from different resources type-safe. */
export type Brand<Value, Name extends string> = Value & { readonly __brand: Name }

/** Stable identifiers shared by adapters, pages, and route metadata. */
export type UserId = Brand<string, 'UserId'>
export type RoleId = Brand<string, 'RoleId'>
export type PermissionId = Brand<string, 'PermissionId'>
export type TenantId = Brand<string, 'TenantId'>

/** A tenant visible to the current authenticated session. */
export interface Tenant {
  id: TenantId
  name: string
  code: string
  status: 'active' | 'suspended'
}

/** The compact user model used by the base administration screens. */
export interface User {
  id: UserId
  username: string
  displayName: string
  email: string
  status: 'active' | 'disabled'
  roleNames: string[]
}

/** A role and its effective permission codes. */
export interface Role {
  id: RoleId
  name: string
  description: string
  permissionCodes: string[]
  userCount: number
}

/** A permission node that can be rendered as a tree. */
export interface Permission {
  id: PermissionId
  name: string
  code: string
  type: 'menu' | 'page' | 'action'
  children?: Permission[]
}

/** Common list-query contract for all service adapters. */
export interface ListQuery {
  page: number
  pageSize: number
  search?: string
}

/** Paginated service response. */
export interface PageResult<Item> {
  items: Item[]
  total: number
}

/** Services required by the reusable base administration package. */
export interface BaseServices {
  users: UserService
  roles: RoleService
  permissions: PermissionService
  tenants: TenantService
}

export interface UserService {
  list(query: ListQuery): Promise<PageResult<User>>
  updateStatus(id: UserId, status: User['status']): Promise<void>
}

export interface RoleService {
  list(query: ListQuery): Promise<PageResult<Role>>
}

export interface PermissionService {
  list(): Promise<Permission[]>
}

export interface TenantService {
  list(query: ListQuery): Promise<PageResult<Tenant>>
}

/** Upload adapter injected into the shared FileUpload component. */
export interface FileStoragePort {
  upload(input: UploadInput, signal?: AbortSignal): Promise<UploadResult>
  remove?(file: StoredFile): Promise<void>
  getPreviewUrl?(file: StoredFile): Promise<string>
}

export interface UploadInput {
  file: File
  onProgress?: (progress: number) => void
}

export interface StoredFile {
  id: string
  name: string
  size: number
  contentType: string
  url: string
}

export interface UploadResult {
  file: StoredFile
}

/** Authentication state restored by an application-specific auth adapter. */
export interface AuthSession {
  user: User
  permissions: string[]
  tenants: Tenant[]
  currentTenantId: TenantId | null
}

export interface AuthPort {
  restore(): Promise<AuthSession | null>
  signOut(): Promise<void>
}

/** A navigation item is declared by a module and filtered at runtime. */
export interface NavigationItem {
  id: string
  label: string
  to?: string
  permission?: string
  children?: NavigationItem[]
}

/** Permission metadata published by a module. */
export interface PermissionDefinition {
  code: string
  label: string
  description?: string
}

/** The package-level contract consumed by the application composition root. */
export interface AdminModule {
  id: string
  routeNamespace: string
  navigation: NavigationItem[]
  permissions: PermissionDefinition[]
  install?: (context: AdminModuleContext) => void | (() => void)
}

export interface AdminModuleContext {
  services: unknown
  permissions: ReadonlySet<string>
}

/** All services are supplied by the application so packages remain backend-neutral. */
export interface AppServices {
  base: BaseServices
  storage: FileStoragePort
}

/** Optional brand values consumed by the base layout. */
export interface AdminBranding {
  productName: string
  productMark: string
}
