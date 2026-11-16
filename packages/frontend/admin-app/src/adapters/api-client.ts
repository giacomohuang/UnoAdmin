import ky from 'ky'

/** ApiError 携带后端返回的 HTTP 状态码，方便上层区分未授权与业务错误。 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/** ApiClient 是与具体业务无关的 JSON 请求客户端。 */
export interface ApiClient {
  request<Response>(path: string, init?: RequestInit): Promise<Response>
}

/** ApiClientOptions 注入令牌来源、刷新回调与可替换的 fetch 实现。 */
export interface ApiClientOptions {
  baseURL: string
  getAccessToken(): string | null
  getRefreshToken(): string | null
  getTenantId(): string | null
  onTokens(access: string, refresh?: string): void
  onAuthFailed?(): void
  fetchImpl?: typeof fetch
}

/** createApiClient 创建带 Bearer/租户头与 401 自动刷新重试的客户端，并发 401 共享同一次刷新。 */
export function createApiClient(options: ApiClientOptions): ApiClient {
  const baseURL = options.baseURL.endsWith('/') ? options.baseURL : `${options.baseURL}/`
  const transport = ky.create({
    prefix: baseURL,
    retry: 0,
    timeout: false,
    throwHttpErrors: false,
    ...(options.fetchImpl ? { fetch: options.fetchImpl } : {}),
  })
  const client = transport.extend({
    hooks: {
      beforeRequest: [
        ({ request }) => {
          const token = options.getAccessToken()
          const tenantId = options.getTenantId()
          if (token) request.headers.set('Authorization', `Bearer ${token}`)
          if (tenantId) request.headers.set('X-Tenant-Id', tenantId)
          if (!request.headers.has('Accept')) request.headers.set('Accept', 'application/json')
          if (request.body && !request.headers.has('Content-Type')) {
            request.headers.set('Content-Type', 'application/json')
          }
        },
      ],
    },
  })

  /** Normalize path segments because Ky's prefix option owns URL joining. */
  const normalizePath = (path: string): string => path.replace(/^\/+/, '')

  /**
   * performRefresh 用 Redis 中的刷新令牌换取新令牌对。
   * 只有服务端明确拒绝刷新令牌才判定会话失效；网络故障保留本地令牌，避免偶发抖动踢人下线。
   */
  async function performRefresh(): Promise<boolean> {
    const refreshToken = options.getRefreshToken()
    if (!refreshToken) return false

    // superseded 判断刷新期间令牌是否已被登录/切租户替换，旧响应不得覆盖新会话。
    const superseded = (): boolean => options.getRefreshToken() !== refreshToken

    let response: Response
    try {
      response = await transport('auth/refresh', {
        method: 'POST',
        json: { refreshToken },
      })
    } catch {
      return false
    }

    // 401/403 表示刷新令牌已过期或已被轮换，此时会话不可恢复。
    if (response.status === 401 || response.status === 403) {
      // 令牌已被新会话接管时不算失败，交给调用方用新令牌重放。
      if (superseded()) return true
      options.onAuthFailed?.()
      return false
    }
    if (!response.ok) return false

    try {
      const data = (await response.json()) as { token: string; refreshToken?: string }
      if (superseded()) return true
      options.onTokens(data.token, data.refreshToken)
      return true
    } catch {
      return false
    }
  }

  /** refreshing 缓存进行中的刷新任务，保证并发 401 只打出一次 auth/refresh。 */
  let refreshing: Promise<boolean> | null = null

  /** refresh 复用同一个在飞任务；服务端会轮换刷新令牌，重复请求会互相作废。 */
  function refresh(): Promise<boolean> {
    if (!refreshing) {
      refreshing = performRefresh().finally(() => {
        refreshing = null
      })
    }
    return refreshing
  }

  async function send<Response>(path: string, init: RequestInit, retried: boolean): Promise<Response> {
    // 记录本次请求实际携带的令牌，用于识别「并发请求里已经有人刷新成功」。
    const tokenUsed = options.getAccessToken()
    const response = await client(normalizePath(path), init)
    if (response.status === 401 && !retried && options.getRefreshToken()) {
      // 令牌已被其他请求换新则直接重放；否则复用同一个刷新任务，避免重复轮换。
      const refreshed = options.getAccessToken() !== tokenUsed || (await refresh())
      if (refreshed) return send<Response>(path, init, true)
    }
    if (!response.ok) {
      let message = `Request failed with status ${response.status}`
      try {
        const body = (await response.json()) as { error?: string }
        if (body.error) message = body.error
      } catch {
        // 非 JSON 错误体保留默认文案。
      }
      throw new ApiError(response.status, message)
    }
    return (await response.json()) as Response
  }

  return {
    request<Response>(path: string, init: RequestInit = {}): Promise<Response> {
      return send<Response>(path, init, false)
    },
  }
}
