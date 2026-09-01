/** Common list-query contract for all service adapters. */
export interface ListQuery {
  page: number
  pageSize: number
  search?: string
}

/** Paginated service response. */
export interface PageResult<Item> {
  items: Item[]
  total: number
}
