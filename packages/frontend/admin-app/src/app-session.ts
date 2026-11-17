import { inject, reactive, readonly } from 'vue'

import {
  ADMIN_SESSION_KEY,
  type AdminSession,
  type AdminSessionState,
  type AuthSession,
  type CreateAdminSessionOptions,
  type LegalEntityId,
  type TenantId
} from '@unoadmin/core'

/**
 * 为具体 SPA 应用创建认证、租户与经营主体状态。
 *
 * 状态只在这里维护；页面通过注入的会话读取，切换上下文后服务端会重新下发
 * 可切换范围，因此前端不需要自行推算数据边界。
 */
export function createAdminSession(options: CreateAdminSessionOptions): AdminSession {
  const state = reactive<AdminSessionState>({
    ready: false,
    account: null,
    permissions: [],
    tenants: [],
    currentTenant: null,
    legalEntities: [],
    currentLegalEntity: null,
    dataScope: 'entity'
  })

  /** 用服务端会话覆盖本地状态，空会话表示登出。 */
  const apply = (session: AuthSession | null) => {
    state.account = session?.account ?? null
    state.permissions = session?.permissions ?? []
    state.tenants = session?.tenants ?? []
    state.legalEntities = session?.legalEntities ?? []
    state.dataScope = session?.dataScope ?? 'entity'
    state.currentTenant = session?.currentTenantId
      ? state.tenants.find((tenant) => tenant.id === session.currentTenantId) ?? null
      : null
    state.currentLegalEntity = session?.currentLegalEntityId
      ? state.legalEntities.find((entity) => entity.id === session.currentLegalEntityId) ?? null
      : null
  }

  return {
    state: readonly(state),
    async signIn(credentials) {
      apply(await options.auth.signIn(credentials))
    },
    async initialize() {
      apply(await options.auth.restore())
      state.ready = true
    },
    can(permission) {
      if (!permission) return true
      return state.permissions.includes('*') || state.permissions.includes(permission)
    },
    canAccessEntity(id) {
      if (state.account?.platformAdmin || state.dataScope === 'tenant') return true
      return state.legalEntities.some((entity) => entity.id === id)
    },
    async switchTenant(id: TenantId) {
      const session = await options.auth.switchTenant?.(id)
      if (session) {
        apply(session)
        return
      }
      const tenant = state.tenants.find((item) => item.id === id)
      if (!tenant) throw new Error(`Tenant ${id} is not available to the current account`)
      state.currentTenant = tenant
    },
    async switchLegalEntity(id: LegalEntityId) {
      const session = await options.auth.switchLegalEntity?.(id)
      if (session) {
        apply(session)
        return
      }
      const entity = state.legalEntities.find((item) => item.id === id)
      if (!entity) throw new Error(`Legal entity ${id} is not available to the current account`)
      state.currentLegalEntity = entity
    },
    async refresh() {
      const session = await options.auth.restore()
      if (session) apply(session)
    },
    async signOut() {
      await options.auth.signOut()
      apply(null)
    }
  }
}

/** 在 admin-app 组件中读取应用会话。 */
export function useAppSession(): AdminSession {
  const session = inject(ADMIN_SESSION_KEY)
  if (!session) throw new Error('Admin session is not provided')
  return session as AdminSession
}
