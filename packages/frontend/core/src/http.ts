import type { TenantId } from './types/identifiers'

/** Configuration injected by an application-specific HTTP adapter. */
export interface HttpClientOptions {
  baseURL: string
  getToken?: () => string | null
  getTenantId?: () => TenantId | null
  fetchImpl?: typeof fetch
}

/** A small fetch wrapper shared by service adapters. */
export interface HttpClient {
  request<Response>(path: string, init?: RequestInit): Promise<Response>
}

/** Create a backend-neutral HTTP client with auth and tenant headers. */
export function createHttpClient(options: HttpClientOptions): HttpClient {
  const fetchImpl = options.fetchImpl || fetch

  return {
    async request<Response>(path: string, init: RequestInit = {}) {
      const headers = new Headers(init.headers)
      const token = options.getToken?.()
      const tenantId = options.getTenantId?.()
      if (token) headers.set('Authorization', `Bearer ${token}`)
      if (tenantId) headers.set('X-Tenant-Id', tenantId)
      if (!headers.has('Accept')) headers.set('Accept', 'application/json')

      const response = await fetchImpl(new URL(path, options.baseURL), {
        ...init,
        headers
      })
      if (!response.ok) throw new Error(`Request failed with status ${response.status}`)
      return (await response.json()) as Response
    }
  }
}
