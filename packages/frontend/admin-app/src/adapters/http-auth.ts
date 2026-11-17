import type { AuthPort, AuthSession, LegalEntityId, SignInCredentials, TenantId } from '@unoadmin/core'

import type { ApiClient } from './api-client'
import type { TokenStore } from './token-store'

/** 登录/刷新接口返回的令牌对与会话。 */
export interface LoginResponse extends AuthSession {
  token: string
  refreshToken: string
}

/** 切换上下文接口只返回新的令牌对，会话需再读取一次。 */
export interface SwitchContextResponse {
  token: string
  refreshToken?: string
}

/**
 * 把 AuthPort 映射到真实后端，令牌与当前上下文通过 TokenStore 持久化。
 *
 * 切换租户/主体后主动调用 restore，让服务端重新计算可切换范围与数据范围，
 * 避免前端用旧的主体列表渲染新的租户上下文。
 */
export function createHttpAuth(client: ApiClient, store: TokenStore): AuthPort {
  /** 保存上下文并返回最新会话。 */
  async function applyContext(session: AuthSession): Promise<AuthSession> {
    store.setContext(session.currentTenantId, session.currentLegalEntityId)
    return session
  }

  return {
    async signIn(credentials?: SignInCredentials) {
      if (!credentials?.username || !credentials?.password) {
        throw new Error('请输入账号和密码')
      }

      const data = await client.request<LoginResponse>('auth/login', {
        method: 'POST',
        body: JSON.stringify(credentials)
      })
      store.setTokens(data.token, data.refreshToken)
      store.setContext(data.currentTenantId, data.currentLegalEntityId)
      return data
    },

    async restore() {
      if (!store.getAccess()) return null
      try {
        const session = await client.request<AuthSession>('auth/restore', { method: 'POST' })
        return await applyContext(session)
      } catch {
        // 无有效会话时按 AuthPort 契约返回 null，由路由守卫引导登录。
        return null
      }
    },

    async signOut() {
      try {
        await client.request('auth/logout', {
          method: 'POST',
          body: JSON.stringify({ refreshToken: store.getRefresh() })
        })
      } finally {
        // 即使后端不可达也要清掉本地令牌，避免下次启动使用失效会话。
        store.clear()
      }
    },

    async switchTenant(id: TenantId) {
      const data = await client.request<SwitchContextResponse>('auth/switch-tenant', {
        method: 'POST',
        body: JSON.stringify({ tenantId: id, refreshToken: store.getRefresh() })
      })
      store.setTokens(data.token, data.refreshToken)
      const session = await client.request<AuthSession>('auth/restore', { method: 'POST' })
      return await applyContext(session)
    },

    async switchLegalEntity(id: LegalEntityId) {
      const data = await client.request<SwitchContextResponse>('auth/switch-legal-entity', {
        method: 'POST',
        body: JSON.stringify({ legalEntityId: id, refreshToken: store.getRefresh() })
      })
      store.setTokens(data.token, data.refreshToken)
      const session = await client.request<AuthSession>('auth/restore', { method: 'POST' })
      return await applyContext(session)
    }
  }
}
