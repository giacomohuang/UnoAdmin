import type {
  Account,
  AccountId,
  AdminStats,
  AppServices,
  LegalEntity,
  LegalEntityId,
  ListQuery,
  OrgId,
  OrgUnit,
  PageResult,
  Permission,
  PermissionId,
  ReorderPermissions,
  Role,
  RoleId,
  SaveAccount,
  SaveLegalEntity,
  SaveOrg,
  SavePermission,
  SaveRole,
  SaveTenant,
  Tenant,
  TenantId
} from '@unoadmin/core'

import type { ApiClient } from './api-client'

/** 把前端 ListQuery 映射为后端统一的分页查询参数。 */
function toQueryString(query: ListQuery): string {
  const params = new URLSearchParams({
    page: String(query.page),
    pageSize: String(query.pageSize)
  })
  if (query.search) params.set('search', query.search)
  if (query.tenantId) params.set('tenantId', query.tenantId)
  if (query.legalEntityId) params.set('legalEntityId', query.legalEntityId)
  if (query.status) params.set('status', query.status)
  return params.toString()
}

/** 统一的写请求封装，保持 JSON 序列化位置一致。 */
function write<Response>(client: ApiClient, path: string, method: string, body: unknown): Promise<Response> {
  return client.request<Response>(path, { method, body: JSON.stringify(body) })
}

/** 实现应用所需的全部服务，统一通过 REST 访问后端。 */
export function createHttpServices(client: ApiClient): AppServices {
  return {
    base: {
      accounts: {
        list: (query) => client.request<PageResult<Account>>(`accounts?${toQueryString(query)}`),
        create: (input: SaveAccount) => write<Account>(client, 'accounts', 'POST', input),
        update: (id: AccountId, input: SaveAccount) =>
          write<Account>(client, `accounts/${encodeURIComponent(id)}`, 'PUT', input),
        updateStatus: (id: AccountId, status: Account['status']) =>
          write<void>(client, `accounts/${encodeURIComponent(id)}/status`, 'PATCH', { status })
      },
      roles: {
        list: (query) => client.request<PageResult<Role>>(`roles?${toQueryString(query)}`),
        create: (input: SaveRole) => write<Role>(client, 'roles', 'POST', input),
        update: (id: RoleId, input: SaveRole) => write<Role>(client, `roles/${encodeURIComponent(id)}`, 'PUT', input)
      },
      permissions: {
        list: () => client.request<Permission[]>('permissions'),
        create: (input: SavePermission) => write<Permission>(client, 'permissions', 'POST', input),
        update: (id: PermissionId, input: SavePermission) =>
          write<Permission>(client, `permissions/${encodeURIComponent(id)}`, 'PUT', input),
        remove: (id: PermissionId) =>
          client.request<void>(`permissions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
        reorder: (input: ReorderPermissions) => write<Permission[]>(client, 'permissions/reorder', 'POST', input)
      },
      tenants: {
        list: (query) => client.request<PageResult<Tenant>>(`tenants?${toQueryString(query)}`),
        create: (input: SaveTenant) => write<Tenant>(client, 'tenants', 'POST', input),
        update: (id: TenantId, input: SaveTenant) => write<Tenant>(client, `tenants/${encodeURIComponent(id)}`, 'PUT', input)
      },
      legalEntities: {
        list: (query) => client.request<LegalEntity[]>(`legal-entities?${toQueryString(query)}`),
        create: (input: SaveLegalEntity) => write<LegalEntity>(client, 'legal-entities', 'POST', input),
        update: (id: LegalEntityId, input: SaveLegalEntity) =>
          write<LegalEntity>(client, `legal-entities/${encodeURIComponent(id)}`, 'PUT', input)
      },
      orgs: {
        tree: (legalEntityId: string) =>
          client.request<OrgUnit[]>(`orgs?${new URLSearchParams({ legalEntityId }).toString()}`),
        create: (input: SaveOrg) => write<OrgUnit>(client, 'orgs', 'POST', input),
        update: (id: OrgId, input: SaveOrg) => write<OrgUnit>(client, `orgs/${encodeURIComponent(id)}`, 'PUT', input)
      },
      stats: {
        get: () => client.request<AdminStats>('stats')
      }
    }
  }
}
