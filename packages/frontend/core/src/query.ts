/** 所有列表服务共用的查询契约。 */
export interface ListQuery {
  page: number
  pageSize: number
  search?: string
  /** 平台管理员可按租户过滤；普通账户会被后端强制收敛到自身租户。 */
  tenantId?: string
  /** 按经营主体过滤（例如只查看某公司的账户）。 */
  legalEntityId?: string
  /** 按状态过滤，取值由各页面决定。 */
  status?: string
}

/** 分页服务响应。 */
export interface PageResult<Item> {
  items: Item[]
  total: number
}

/** 顶部工作台使用的规模指标。 */
export interface AdminStats {
  tenants: number
  legalEntities: number
  accounts: number
  roles: number
}
