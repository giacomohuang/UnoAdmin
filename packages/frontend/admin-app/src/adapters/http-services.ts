import type { AppServices, ListQuery, PageResult, Permission, Role, Tenant, User, UserId } from '@unoadmin/core'

import type { ApiClient } from './api-client'

/** toQueryString 把前端 ListQuery 映射为后端统一的分页查询参数。 */
function toQueryString(query: ListQuery): string {
  const params = new URLSearchParams({
    page: String(query.page),
    pageSize: String(query.pageSize),
  })
  if (query.search) params.set('search', query.search)
  return params.toString()
}

/** createHttpServices 实现应用所需服务，全部通过 REST 访问后端。 */
export function createHttpServices(client: ApiClient): AppServices {
  return {
    base: {
      users: {
        list: (query) => client.request<PageResult<User>>(`users?${toQueryString(query)}`),
        updateStatus: (id: UserId, status: User['status']) =>
          client.request(`users/${encodeURIComponent(id)}/status`, {
            method: 'PATCH',
            body: JSON.stringify({ status }),
          }),
      },
      roles: {
        list: (query) => client.request<PageResult<Role>>(`roles?${toQueryString(query)}`),
      },
      permissions: {
        list: () => client.request<Permission[]>('permissions'),
      },
      tenants: {
        list: (query) => client.request<PageResult<Tenant>>(`tenants?${toQueryString(query)}`),
      },
    },
  }
}
