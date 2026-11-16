import { inject, reactive, readonly } from 'vue'

import {
  ADMIN_SESSION_KEY,
  type AdminSession,
  type AdminSessionState,
  type AuthSession,
  type CreateAdminSessionOptions,
} from '@unoadmin/core'

/** Create authentication and tenant state for the concrete SPA application. */
export function createAdminSession(options: CreateAdminSessionOptions): AdminSession {
  const state = reactive<AdminSessionState>({
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

  return {
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
}

/** Read the application session inside admin-app components. */
export function useAppSession(): AdminSession {
  const session = inject(ADMIN_SESSION_KEY)
  if (!session) throw new Error('Admin session is not provided')
  return session as AdminSession
}
