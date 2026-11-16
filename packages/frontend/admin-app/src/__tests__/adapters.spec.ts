import type { AuthSession, TenantId } from '@unoadmin/core'
import { describe, expect, it, vi } from 'vitest'

import { ApiError, createApiClient, type ApiClient } from '../adapters/api-client'
import { createHttpAuth } from '../adapters/http-auth'
import { createHttpServices } from '../adapters/http-services'
import { createTokenStore, type TokenStore } from '../adapters/token-store'

/** 创建内存 Storage，隔离浏览器 localStorage 状态。 */
function memoryStorage(): Storage {
  const data = new Map<string, string>()
  return {
    get length() {
      return data.size
    },
    clear: () => data.clear(),
    getItem: (key) => data.get(key) ?? null,
    key: (index) => [...data.keys()][index] ?? null,
    removeItem: (key) => data.delete(key),
    setItem: (key, value) => {
      data.set(key, value)
    },
  }
}

/** 构造最小登录响应，供认证适配器测试使用。 */
function loginResponse(): AuthSession & { token: string; refreshToken: string } {
  return {
    token: 'access-1',
    refreshToken: 'refresh-1',
    user: {
      id: 'user-001' as AuthSession['user']['id'],
      username: 'lin.wei',
      displayName: '林薇',
      email: 'lin.wei@example.com',
      status: 'active',
      roleNames: ['平台管理员'],
    },
    permissions: ['*'],
    tenants: [{ id: 'tenant-main' as TenantId, name: '总部空间', code: 'HQ', status: 'active' }],
    currentTenantId: 'tenant-main' as TenantId,
  }
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

describe('token store', () => {
  it('persists tokens and tenant, then clears them all', () => {
    const store = createTokenStore(memoryStorage())
    store.setTokens('access', 'refresh')
    store.setTenant('tenant-main' as TenantId)

    expect(store.getAccess()).toBe('access')
    expect(store.getRefresh()).toBe('refresh')
    expect(store.getTenant()).toBe('tenant-main')

    store.clear()
    expect(store.getAccess()).toBeNull()
    expect(store.getRefresh()).toBeNull()
    expect(store.getTenant()).toBeNull()
  })
})

describe('api client', () => {
  it('refreshes the token and retries the original request on 401', async () => {
    const store = createTokenStore(memoryStorage())
    store.setTokens('expired-access', 'refresh-1')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'unauthorized' }, 401))
      .mockResolvedValueOnce(jsonResponse({ token: 'new-access', refreshToken: 'refresh-2' }))
      .mockResolvedValueOnce(jsonResponse({ items: [], total: 0 }))

    const client = createApiClient({
      baseURL: 'http://api.test/api/v1',
      getAccessToken: () => store.getAccess(),
      getRefreshToken: () => store.getRefresh(),
      getTenantId: () => 'tenant-main' as TenantId,
      onTokens: (access, refresh) => store.setTokens(access, refresh),
      fetchImpl: fetchMock as unknown as typeof fetch,
    })

    await expect(client.request('users?page=1&pageSize=10')).resolves.toEqual({ items: [], total: 0 })
    expect(fetchMock).toHaveBeenCalledTimes(3)
    const retryRequest = fetchMock.mock.calls[2]?.[0] as Request
    expect(retryRequest.url).toBe('http://api.test/api/v1/users?page=1&pageSize=10')
    expect(retryRequest.headers.get('Authorization')).toBe('Bearer new-access')
    expect(retryRequest.headers.get('X-Tenant-Id')).toBe('tenant-main')
    expect(store.getRefresh()).toBe('refresh-2')
  })

  it('clears the session when refresh fails', async () => {
    const onAuthFailed = vi.fn()
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'unauthorized' }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: 'expired refresh token' }, 401))
    const client = createApiClient({
      baseURL: 'http://api.test/api/v1/',
      getAccessToken: () => 'expired',
      getRefreshToken: () => 'refresh-1',
      getTenantId: () => null,
      onTokens: () => {},
      onAuthFailed,
      fetchImpl: fetchMock as unknown as typeof fetch,
    })

    await expect(client.request('users')).rejects.toBeInstanceOf(ApiError)
    expect(onAuthFailed).toHaveBeenCalledOnce()
  })

  it('coalesces concurrent 401s into a single refresh', async () => {
    const store = createTokenStore(memoryStorage())
    store.setTokens('expired-access', 'refresh-1')
    const refreshCalls: string[] = []
    const fetchMock = vi.fn(async (request: Request) => {
      if (request.url.endsWith('/auth/refresh')) {
        refreshCalls.push(request.url)
        return jsonResponse({ token: 'new-access', refreshToken: 'refresh-2' })
      }
      if (request.headers.get('Authorization') === 'Bearer new-access') {
        return jsonResponse({ items: [], total: 0 })
      }
      return jsonResponse({ error: 'unauthorized' }, 401)
    })

    const client = createApiClient({
      baseURL: 'http://api.test/api/v1/',
      getAccessToken: () => store.getAccess(),
      getRefreshToken: () => store.getRefresh(),
      getTenantId: () => null,
      onTokens: (access, refresh) => store.setTokens(access, refresh),
      fetchImpl: fetchMock as unknown as typeof fetch,
    })

    const results = await Promise.all([
      client.request('users'),
      client.request('roles'),
      client.request('tenants'),
    ])

    expect(results).toEqual([
      { items: [], total: 0 },
      { items: [], total: 0 },
      { items: [], total: 0 },
    ])
    // 三个并发 401 只应触发一次刷新，否则服务端会因轮换把彼此的刷新令牌作废。
    expect(refreshCalls).toHaveLength(1)
    expect(store.getRefresh()).toBe('refresh-2')
  })

  it('replays with the token another request already refreshed', async () => {
    const store = createTokenStore(memoryStorage())
    store.setTokens('expired-access', 'refresh-1')
    let refreshCalls = 0
    const fetchMock = vi.fn(async (request: Request) => {
      if (request.url.endsWith('/auth/refresh')) {
        refreshCalls += 1
        return jsonResponse({ token: 'unexpected', refreshToken: 'refresh-9' })
      }
      if (request.headers.get('Authorization') === 'Bearer new-access') {
        return jsonResponse({ items: [], total: 0 })
      }
      // 模拟并发同伴在本次 401 返回前已完成刷新。
      store.setTokens('new-access', 'refresh-2')
      return jsonResponse({ error: 'unauthorized' }, 401)
    })

    const client = createApiClient({
      baseURL: 'http://api.test/api/v1/',
      getAccessToken: () => store.getAccess(),
      getRefreshToken: () => store.getRefresh(),
      getTenantId: () => null,
      onTokens: (access, refresh) => store.setTokens(access, refresh),
      fetchImpl: fetchMock as unknown as typeof fetch,
    })

    await expect(client.request('users')).resolves.toEqual({ items: [], total: 0 })
    // 令牌已被轮换过，不应再打 refresh，直接重放即可。
    expect(refreshCalls).toBe(0)
    expect(store.getRefresh()).toBe('refresh-2')
  })

  it('keeps the session when refresh fails for network reasons', async () => {
    store.setTokens('expired-access', 'refresh-1')
    const onAuthFailed = vi.fn()
    const fetchMock = vi.fn(async (request: Request) => {
      if (request.url.endsWith('/auth/refresh')) throw new TypeError('network down')
      return jsonResponse({ error: 'unauthorized' }, 401)
    })

    const client = createApiClient({
      baseURL: 'http://api.test/api/v1/',
      getAccessToken: () => store.getAccess(),
      getRefreshToken: () => store.getRefresh(),
      getTenantId: () => null,
      onTokens: (access, refresh) => store.setTokens(access, refresh),
      onAuthFailed,
      fetchImpl: fetchMock as unknown as typeof fetch,
    })

    await expect(client.request('users')).rejects.toBeInstanceOf(ApiError)
    expect(onAuthFailed).not.toHaveBeenCalled()
    expect(store.getRefresh()).toBe('refresh-1')
  })
})

describe('http auth adapter', () => {
  it('stores tokens on sign-in and clears them on sign-out failure', async () => {
    const store: TokenStore = createTokenStore(memoryStorage())
    const client: ApiClient = {
      request: vi.fn().mockImplementation((path: string) => {
        if (path === 'auth/login') return Promise.resolve(loginResponse())
        if (path === 'auth/logout') return Promise.reject(new Error('offline'))
        return Promise.resolve(null)
      }),
    }
    const auth = createHttpAuth(client, store)

    const session = await auth.signIn({ username: 'lin.wei', password: 'admin123' })
    expect(session.user.username).toBe('lin.wei')
    expect(store.getAccess()).toBe('access-1')
    expect(store.getRefresh()).toBe('refresh-1')
    expect(store.getTenant()).toBe('tenant-main')

    await expect(auth.signOut()).rejects.toThrow('offline')
    expect(store.getAccess()).toBeNull()
  })

  it('returns null when no token is stored', async () => {
    const store = createTokenStore(memoryStorage())
    const auth = createHttpAuth({ request: vi.fn() } as unknown as ApiClient, store)
    await expect(auth.restore()).resolves.toBeNull()
  })
})

describe('http services adapter', () => {
  it('maps list queries and status updates to REST calls', async () => {
    const request = vi.fn().mockResolvedValue({ items: [], total: 0 })
    const services = createHttpServices({ request } as unknown as ApiClient)

    await services.base.users.list({ page: 2, pageSize: 5, search: '林' })
    expect(request).toHaveBeenCalledWith('users?page=2&pageSize=5&search=%E6%9E%97')

    await services.base.users.updateStatus('user-001' as AuthSession['user']['id'], 'disabled')
    expect(request).toHaveBeenCalledWith('users/user-001/status', {
      method: 'PATCH',
      body: '{"status":"disabled"}',
    })
  })
})
