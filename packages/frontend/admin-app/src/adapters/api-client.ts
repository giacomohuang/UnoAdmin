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

/** createApiClient 创建带 Bearer/租户头与 401 自动刷新重试的客户端。 */
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

  /** refresh 用 Redis 中的刷新令牌换取新令牌对；失败时清空本地会话。 */
  async function refresh(): Promise<boolean> {
    const refreshToken = options.getRefreshToken()
    if (!refreshToken) return false

    try {
      const response = await transport('auth/refresh', {
        method: 'POST',
        json: { refreshToken },
      })
      if (!response.ok) throw new ApiError(response.status, 'refresh token rejected')
      const data = (await response.json()) as { token: string; refreshToken: string }
      options.onTokens(data.token, data.refreshToken)
      return true
    } catch {
      options.onAuthFailed?.()
      return false
    }
  }

  async function send<Response>(path: string, init: RequestInit, retried: boolean): Promise<Response> {
    const response = await client(normalizePath(path), init)
    if (response.status === 401 && !retried && options.getRefreshToken()) {
      if (await refresh()) return send<Response>(path, init, true)
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
