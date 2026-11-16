import type { AuthPort, AuthSession, SignInCredentials, TenantId } from '@unoadmin/core'

import type { ApiClient } from './api-client'
import type { TokenStore } from './token-store'

/** LoginResponse 是后端登录/刷新接口返回的令牌对与会话。 */
export interface LoginResponse extends AuthSession {
  token: string
  refreshToken: string
}

/** SwitchTenantResponse 是切换租户接口返回的新令牌对。 */
export interface SwitchTenantResponse {
  token: string
  refreshToken?: string
}

/** createHttpAuth 把 AuthPort 映射到真实后端，令牌通过 TokenStore 持久化。 */
export function createHttpAuth(client: ApiClient, store: TokenStore): AuthPort {
  return {
    async signIn(credentials?: SignInCredentials) {
      if (!credentials?.username || !credentials?.password) {
        throw new Error('请输入用户名和密码')
      }

      const data = await client.request<LoginResponse>('auth/login', {
        method: 'POST',
        body: JSON.stringify(credentials),
      })
      store.setTokens(data.token, data.refreshToken)
      store.setTenant(data.currentTenantId as TenantId)
      return {
        user: data.user,
        permissions: data.permissions,
        tenants: data.tenants,
        currentTenantId: data.currentTenantId,
      }
    },

    async restore() {
      if (!store.getAccess()) return null
      try {
        return await client.request<AuthSession>('auth/restore', { method: 'POST' })
      } catch {
        // 无有效会话时按 AuthPort 契约返回 null，由路由守卫引导登录。
        return null
      }
    },

    async signOut() {
      try {
        await client.request('auth/logout', {
          method: 'POST',
          body: JSON.stringify({ refreshToken: store.getRefresh() }),
        })
      } finally {
        // 即使后端不可达也要清掉本地令牌，避免下次启动使用失效会话。
        store.clear()
      }
    },

    async switchTenant(id: TenantId) {
      const data = await client.request<SwitchTenantResponse>('auth/switch-tenant', {
        method: 'POST',
        body: JSON.stringify({ tenantId: id, refreshToken: store.getRefresh() }),
      })
      store.setTokens(data.token, data.refreshToken)
      store.setTenant(id)
    },
  }
}
