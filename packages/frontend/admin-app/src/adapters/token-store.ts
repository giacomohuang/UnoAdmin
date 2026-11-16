import type { TenantId } from '@unoadmin/core'

/** TokenStore 保存访问令牌、刷新令牌与当前租户，供所有 HTTP 适配器共享。 */
export interface TokenStore {
  getAccess(): string | null
  getRefresh(): string | null
  getTenant(): TenantId | null
  setTokens(access: string, refresh?: string): void
  setTenant(id: TenantId): void
  clear(): void
}

const ACCESS_KEY = 'unoadmin.accessToken'
const REFRESH_KEY = 'unoadmin.refreshToken'
const TENANT_KEY = 'unoadmin.tenantId'

/** 在测试或隐私模式下退化为内存存储，避免 localStorage 不可用时崩溃。 */
function resolveStorage(): Storage | null {
  try {
    return globalThis.localStorage ?? null
  } catch {
    return null
  }
}

/** createTokenStore 创建令牌存储；不传 storage 时使用浏览器 localStorage。 */
export function createTokenStore(storage: Storage | null = resolveStorage()): TokenStore {
  let memory: Record<string, string> = {}

  const read = (key: string): string | null => storage?.getItem(key) ?? memory[key] ?? null
  const write = (key: string, value: string): void => {
    if (storage) storage.setItem(key, value)
    else memory[key] = value
  }
  const remove = (key: string): void => {
    if (storage) storage.removeItem(key)
    else delete memory[key]
  }

  return {
    getAccess: () => read(ACCESS_KEY),
    getRefresh: () => read(REFRESH_KEY),
    getTenant: () => (read(TENANT_KEY) as TenantId | null) ?? null,
    setTokens(access, refresh) {
      write(ACCESS_KEY, access)
      if (refresh) write(REFRESH_KEY, refresh)
    },
    setTenant: (id) => write(TENANT_KEY, id),
    clear() {
      remove(ACCESS_KEY)
      remove(REFRESH_KEY)
      remove(TENANT_KEY)
      memory = {}
    },
  }
}

/** 应用运行时使用的共享令牌存储。 */
export const tokenStore: TokenStore = createTokenStore()
