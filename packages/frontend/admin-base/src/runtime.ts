import { inject, reactive, readonly, type App, type DeepReadonly, type InjectionKey } from 'vue'

import type {
  AdminBranding,
  AdminModule,
  AppServices,
  AuthPort,
  AuthSession,
  NavigationItem,
  Tenant,
  TenantId,
  User,
} from '@unoadmin/core'

/** Public reactive state shared by the Layout and package pages. */
export interface AdminRuntimeState {
  ready: boolean
  user: User | null
  permissions: string[]
  tenants: Tenant[]
  currentTenant: Tenant | null
}

/** Runtime services exposed to all installed modules. */
export interface AdminRuntime<Services extends AppServices = AppServices> {
  readonly branding: AdminBranding
  readonly modules: readonly AdminModule[]
  readonly services: Services
  readonly state: DeepReadonly<AdminRuntimeState>
  /** Establish a session through the configured authentication adapter. */
  signIn(): Promise<void>
  initialize(): Promise<void>
  can(permission?: string): boolean
  getNavigation(): NavigationItem[]
  switchTenant(id: TenantId): Promise<void>
  signOut(): Promise<void>
}

/** Options used by an application to create one runtime instance. */
export interface CreateAdminRuntimeOptions<Services extends AppServices = AppServices> {
  auth: AuthPort
  branding: AdminBranding
  modules: AdminModule[]
  services: Services
}

/** Injection key used by package components without relying on a global singleton. */
export const ADMIN_RUNTIME_KEY: InjectionKey<AdminRuntime> = Symbol('unoadmin.runtime')

/** Create the application-scoped runtime consumed by all packages. */
export function createAdminRuntime<Services extends AppServices>(
  options: CreateAdminRuntimeOptions<Services>,
): AdminRuntime<Services> {
  const state = reactive<AdminRuntimeState>({
    ready: false,
    user: null,
    permissions: [],
    tenants: [],
    currentTenant: null,
  })

  const setSession = (session: AuthSession | null) => {
    state.user = session?.user || null
    state.permissions = session?.permissions || []
    state.tenants = session?.tenants || []
    state.currentTenant = session
      ? session.tenants.find((tenant) => tenant.id === session.currentTenantId) || null
      : null
  }

  const runtime: AdminRuntime<Services> = {
    branding: options.branding,
    modules: options.modules,
    services: options.services,
    state: readonly(state),
    async signIn() {
      setSession(await options.auth.signIn())
    },
    async initialize() {
      setSession(await options.auth.restore())
      state.ready = true
    },
    can(permission) {
      if (!permission) return true
      return state.permissions.includes('*') || state.permissions.includes(permission)
    },
    getNavigation() {
      return options.modules.flatMap((module) => filterNavigation(module.navigation, runtime.can))
    },
    async switchTenant(id) {
      const tenant = state.tenants.find((item) => item.id === id)
      if (!tenant) throw new Error(`Tenant ${id} is not available to the current user`)
      state.currentTenant = tenant
    },
    async signOut() {
      await options.auth.signOut()
      setSession(null)
    },
  }

  return runtime
}

/** Install a runtime on the Vue application root. */
export function provideAdminRuntime<Services extends AppServices>(
  app: App,
  runtime: AdminRuntime<Services>,
): App {
  return app.provide(ADMIN_RUNTIME_KEY, runtime)
}

/** Read the nearest application runtime from a package component. */
export function useAdminRuntime<Services extends AppServices = AppServices>(): AdminRuntime<Services> {
  const runtime = inject(ADMIN_RUNTIME_KEY)
  if (!runtime) throw new Error('Admin runtime is not provided')
  return runtime as AdminRuntime<Services>
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
