import type { BaseServices } from './services'
import type { FileStoragePort } from './storage'

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

/** Context passed to an optional module installer hook. */
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
