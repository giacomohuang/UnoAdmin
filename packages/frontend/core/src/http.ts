import ky from 'ky'

import type { LegalEntityId, TenantId } from './types/identifiers'

/** Configuration injected by an application-specific HTTP adapter. */
export interface HttpClientOptions {
  baseURL: string
  getToken?: () => string | null
  getTenantId?: () => TenantId | null
  /** 当前经营主体；后端会校验它是否在令牌允许的范围内。 */
  getLegalEntityId?: () => LegalEntityId | null
  fetchImpl?: typeof fetch
}

/** A JSON HTTP client shared by service adapters. */
export interface HttpClient {
  request<Response>(path: string, init?: RequestInit): Promise<Response>
}

/** Create a Ky-backed HTTP client with auth and tenant headers. */
export function createHttpClient(options: HttpClientOptions): HttpClient {
  const client = ky.create({
    prefix: options.baseURL.endsWith('/') ? options.baseURL : `${options.baseURL}/`,
    retry: 0,
    timeout: false,
    throwHttpErrors: false,
    ...(options.fetchImpl ? { fetch: options.fetchImpl } : {}),
    hooks: {
      beforeRequest: [
        ({ request }) => {
          const token = options.getToken?.()
          const tenantId = options.getTenantId?.()
          const legalEntityId = options.getLegalEntityId?.()
          if (token) request.headers.set('Authorization', `Bearer ${token}`)
          if (tenantId) request.headers.set('X-Tenant-Id', tenantId)
          if (legalEntityId) request.headers.set('X-Legal-Entity-Id', legalEntityId)
          if (!request.headers.has('Accept')) request.headers.set('Accept', 'application/json')
          if (request.body && !request.headers.has('Content-Type')) {
            request.headers.set('Content-Type', 'application/json')
          }
        },
      ],
    },
  })

  return {
    async request<Response>(path: string, init: RequestInit = {}) {
      const response = await client(path.replace(/^\/+/, ''), init)
      if (!response.ok) throw new Error(`Request failed with status ${response.status}`)
      return (await response.json()) as Response
    }
  }
}
