import type {
  AdminSession,
  BaseServices,
  LegalEntity,
  OrgId,
  OrgUnit,
  SaveOrg,
} from '@unoadmin/core'
import type { Ref } from 'vue'

/** UnoMap organization node shape used unchanged by the imported frontend. */
export interface OrgItem {
  id: string
  pid: string | null
  entityId: string | null
  name: string
  fullname: string
  leaderName: string
  leaderId?: string | null
  order: number
  status: number
  path: string
  type?: string
  children: OrgItem[]
}

/** Whole-tree response shape expected by the UnoMap page. */
export interface OrganizationRecord {
  items: OrgItem[]
  version: number
  updatedAt: string | null
}

/** Employee row shape retained for source compatibility with UnoMap. */
export interface OrganizationEmployeeRecord {
  id: string
  username: string
  displayName: string
  organizationId: string
  organizationName: string
  organizationFullName: string
  leaderSelectable: boolean
}

/** Minimal legal-entity option contract consumed by the organization drawer. */
export interface OrganizationEntityOption {
  id: string
  name: string
  code: string
}

/** Dependencies used to bridge UnoMap organization API calls to UnoAdmin services. */
export interface OrganizationApiOptions {
  services: BaseServices
  session: AdminSession
  entityId: Ref<string>
}

/** Create the UnoMap-compatible organization API over the current UnoAdmin services. */
export function createOrganizationApi(options: OrganizationApiOptions) {
  const persistedIds = new Set<string>()

  /** Load the current legal entity's tree and map it into UnoMap's editable shape. */
  async function get(): Promise<OrganizationRecord> {
    if (!options.entityId.value) return { items: [], version: 0, updatedAt: null }
    const nodes = await options.services.orgs.tree(options.entityId.value)
    const items = nodes.map(toOrgItem)
    persistedIds.clear()
    for (const item of flattenOrgItems(items)) persistedIds.add(item.id)
    return { items, version: 0, updatedAt: null }
  }

  /** Persist a draft using create-before-child ordering and delete removed server ids. */
  async function save(data: { items: OrgItem[]; expectedTreeVersion: number }): Promise<OrganizationRecord> {
    const before = new Set(persistedIds)
    await persistItems(data.items, '')
    const current = new Set(flattenOrgItems(data.items).map((item) => item.id))
    for (const id of before) {
      if (!current.has(id)) await options.services.orgs.remove(id as OrgId)
    }
    return get()
  }

  /** Recursively persist one UnoMap tree branch. */
  async function persistItems(items: OrgItem[], parentId: string) {
    for (const item of items) {
      const payload: SaveOrg = {
        legalEntityId: options.entityId.value,
        parentId,
        name: item.name.trim(),
        type: item.type ?? 'department',
        sort: item.order,
        status: item.status ? 'active' : 'suspended',
      }
      if (!payload.name) throw new Error('组织名称不能为空')
      if (persistedIds.has(item.id)) {
        await options.services.orgs.update(item.id as OrgId, payload)
      } else {
        const created = await options.services.orgs.create(payload)
        item.id = created.id
        item.pid = created.parentId || null
        item.path = created.path.join('-')
        persistedIds.add(created.id)
      }
      await persistItems(item.children, item.id)
    }
  }

  return {
    get,
    save,
    /** Employee directory is not yet exposed by the UnoAdmin backend. */
    employees: async (_id: string): Promise<OrganizationEmployeeRecord[]> => [],
    /** The current organization tree is scoped to one legal entity. */
    entities: async (): Promise<OrganizationEntityOption[]> =>
      options.session.state.legalEntities
        .filter((entity) => entity.id === options.entityId.value)
        .map(toEntityOption),
  }
}

/** Convert one API organization node into the exact UnoMap node shape. */
function toOrgItem(node: OrgUnit): OrgItem {
  return {
    id: node.id,
    pid: node.parentId || null,
    entityId: node.legalEntityId || null,
    name: node.name,
    fullname: node.name,
    leaderName: '',
    leaderId: null,
    order: node.sort,
    status: node.status === 'active' ? 1 : 0,
    path: node.path.join('-'),
    type: node.type,
    children: (node.children ?? []).map(toOrgItem),
  }
}

/** Flatten an UnoMap organization tree. */
function flattenOrgItems(items: OrgItem[]): OrgItem[] {
  return items.flatMap((item) => [item, ...flattenOrgItems(item.children)])
}

/** Convert a session legal entity into the drawer's entity option contract. */
function toEntityOption(entity: LegalEntity): OrganizationEntityOption {
  return { id: entity.id, name: entity.name, code: entity.code }
}

/** Extract a user-facing message from adapter failures. */
export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return typeof error === 'string' ? error : '操作失败'
}
