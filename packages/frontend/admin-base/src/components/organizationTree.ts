/** epark 组织树的本地编辑规则；仅将数值 ID 适配为 UnoMap 字符串 ID。 */
import type { OrgItem } from '../api/organizations'

/** 整树编辑草稿；抽屉确认只修改草稿，页面保存才提交服务端。 */
export interface OrgData {
  items: OrgItem[]
}

/** epark 节点插入位置：下级、前置同级或后置同级。 */
export type OrgAddDirection = 'child' | 'previous' | 'next'
/** 拖动计算只消费指针坐标，兼容真实事件与缩放后的重算。 */
export type OrgPointer = Pick<MouseEvent, 'clientX' | 'clientY'>

/** 抽屉独立草稿，取消时不修改画布中的既有节点。 */
export interface OrgFormModel {
  id: string
  entityId: string | undefined
  name: string
  leaderId: string | null
  leaderName: string
}

/** 检测后代关系，阻止将节点拖入自身子树。 */
export function containsOrgItem(item: OrgItem, id: string): boolean {
  return item.id === id || item.children.some((child) => containsOrgItem(child, id))
}

/** 规范化服务端快照并重算派生路径，不复用可变响应对象。 */
export function normalizeOrgData(value: unknown): OrgData {
  if (isOrgData(value)) {
    const data = { items: normalizeOrgItems(value.items) }
    refreshOrgTree(data)
    return data
  }

  return { items: [] }
}

/** 创建尚未保存的根节点，沿用 epark 的本地草稿流程。 */
export function createRootOrgData(): OrgData {
  const root = createOrgItem(nextOrgId(), null, 1)
  root.name = '新组织'
  const data = { items: [root] }
  refreshOrgTree(data)
  return data
}

/** 深拷贝整树，隔离画布编辑与服务端响应引用。 */
export function cloneOrgData(value: OrgData): OrgData {
  return JSON.parse(JSON.stringify(value)) as OrgData
}

/** 深拷贝拖动分支，避免幽灵节点参与实际树变更。 */
export function cloneOrgItem(value: OrgItem): OrgItem {
  return JSON.parse(JSON.stringify(value)) as OrgItem
}

/** 统计已返回快照中的节点总数。 */
export function countOrgItems(items: OrgItem[]) {
  return flattenOrgItems(items).length
}

/** 按显示顺序展开树，供统计和持久化标识集合使用。 */
export function flattenOrgItems(items: OrgItem[]) {
  const result: OrgItem[] = []
  visitOrgItems(items, (item) => result.push(item))
  return result
}

/** 按父子顺序遍历整树，提供当前节点的父级上下文。 */
export function visitOrgItems(items: OrgItem[], visitor: (item: OrgItem, parent: OrgItem | null) => void, parent: OrgItem | null = null) {
  for (const item of items) {
    visitor(item, parent)
    visitOrgItems(item.children || [], visitor, item)
  }
}

/** 按稳定标识查找草稿节点，缺失时返回空。 */
export function findOrgItem(items: OrgItem[], id: string): OrgItem | null {
  let matched: OrgItem | null = null
  visitOrgItems(items, (item) => {
    if (item.id === id) matched = item
  })
  return matched
}

/** 从嵌套树查找父节点，根节点返回空。 */
export function findOrgParent(items: OrgItem[], id: string, parent: OrgItem | null = null): OrgItem | null {
  for (const item of items) {
    if (item.id === id) return parent
    const matched = findOrgParent(item.children || [], id, item)
    if (matched) return matched
  }
  return null
}

/** 由派生路径计算画布层级，根节点为第一层。 */
export function getOrgLevel(item: OrgItem) {
  return item.path ? item.path.split('-').filter(Boolean).length : 1
}

/** 立即插入本地节点并重排兄弟顺序，稍后统一保存。 */
export function addOrgItem(data: OrgData, targetId: string, direction: OrgAddDirection) {
  const target = findOrgItem(data.items, targetId)
  if (!target) return null

  const parent = direction === 'child' ? target : findOrgParent(data.items, target.id)
  const siblings = direction === 'child' ? target.children : parent ? parent.children : data.items
  const targetIndex = siblings.findIndex((item) => item.id === target.id)
  const insertIndex = direction === 'child' ? siblings.length : targetIndex + (direction === 'next' ? 1 : 0)
  const next = createOrgItem(nextOrgId(), parent, insertIndex + 1)
  siblings.splice(insertIndex, 0, next)
  resequenceOrgItems(siblings)
  refreshOrgTree(data)
  return next
}

/** 删除节点及其子树的本地草稿；根节点保持不可删除。 */
export function removeOrgItem(data: OrgData, id: string) {
  const parent = findOrgParent(data.items, id)
  const siblings = parent ? parent.children : data.items
  const index = siblings.findIndex((item) => item.id === id)
  if (index < 0 || siblings[index]?.pid === null) return false
  siblings.splice(index, 1)
  resequenceOrgItems(siblings)
  refreshOrgTree(data)
  return true
}

/** 按目标插槽移动整支子树；根节点和循环移动均不允许。 */
export function moveOrgItem(data: OrgData, sourceId: string, targetId: string, direction: OrgAddDirection) {
  const source = findOrgItem(data.items, sourceId)
  const target = findOrgItem(data.items, targetId)
  if (!source || !target || source.id === target.id || source.pid === null) return false
  if (containsOrgItem(source, target.id)) return false

  const sourceParent = findOrgParent(data.items, source.id)
  const sourceSiblings = sourceParent ? sourceParent.children : data.items
  const sourceIndex = sourceSiblings.findIndex((item) => item.id === source.id)
  if (sourceIndex < 0) return false
  sourceSiblings.splice(sourceIndex, 1)
  resequenceOrgItems(sourceSiblings)

  const targetParent = direction === 'child' ? target : findOrgParent(data.items, target.id)
  const targetSiblings = direction === 'child' ? target.children : targetParent ? targetParent.children : data.items
  const targetIndex = targetSiblings.findIndex((item) => item.id === target.id)
  const insertIndex = direction === 'child' ? targetSiblings.length : Math.max(0, targetIndex + (direction === 'next' ? 1 : 0))
  targetSiblings.splice(insertIndex, 0, source)
  resequenceOrgItems(targetSiblings)
  refreshOrgTree(data)
  return true
}

/** 将已确认的抽屉字段合并到树草稿并重算全称。 */
export function updateOrgItem(data: OrgData, nextForm: OrgFormModel) {
  const item = findOrgItem(data.items, nextForm.id)
  if (!item) return false
  item.name = nextForm.name.trim()
  item.leaderId = nextForm.leaderId ?? null
  item.leaderName = nextForm.leaderName.trim()
  item.entityId = nextForm.entityId ?? null
  refreshOrgTree(data)
  return true
}

/** 以嵌套关系为准重建父级、连续排序和派生全称/路径。 */
export function refreshOrgTree(data: OrgData) {
  const refresh = (items: OrgItem[], parent: OrgItem | null) => {
    items.sort((a, b) => a.order - b.order || a.id.localeCompare(b.id))
    items.forEach((item, index) => {
      item.pid = parent?.id ?? null
      item.order = index + 1
      item.fullname = parent ? `${parent.fullname}-${item.name}` : item.name
      item.path = parent ? `${parent.path}-${item.id}` : `${item.id}`
      item.children = normalizeOrgItems(item.children || [])
      refresh(item.children, item)
    })
  }
  data.items = normalizeOrgItems(data.items)
  refresh(data.items, null)
}

/** 为抽屉创建独立字段快照。 */
export function toOrgForm(item: OrgItem): OrgFormModel {
  return {
    id: item.id,
    entityId: item.entityId ?? undefined,
    name: item.name,
    leaderId: item.leaderId ?? null,
    leaderName: item.leaderName
  }
}

/** 补齐节点默认字段，保留字符串标识及已有停用状态。 */
export function normalizeOrgItems(items: unknown): OrgItem[] {
  if (!Array.isArray(items)) return []
  return items.filter(isPlainObject).map((item, index) => ({
    id: toStringValue(item.id, ''),
    pid: item.pid === null || item.pid === undefined ? null : toStringValue(item.pid, ''),
    entityId: item.entityId === null || item.entityId === undefined ? null : toStringValue(item.entityId, ''),
    name: toStringValue(item.name, '未命名部门'),
    fullname: toStringValue(item.fullname, toStringValue(item.name, '未命名部门')),
    leaderId: item.leaderId === null || item.leaderId === undefined ? null : toStringValue(item.leaderId, ''),
    leaderName: toStringValue(item.leaderName, ''),
    order: toNumber(item.order, index + 1),
    status: toNumber(item.status, 1),
    path: toStringValue(item.path, ''),
    children: normalizeOrgItems(item.children)
  }))
}

/** 创建默认部门草稿并继承父节点的路径和全称前缀。 */
export function createOrgItem(id: string, parent: OrgItem | null, order: number): OrgItem {
  return {
    id,
    pid: parent?.id ?? null,
    entityId: null,
    leaderId: null,
    name: '新部门',
    fullname: parent ? `${parent.fullname}-新部门` : '新部门',
    leaderName: '',
    order,
    status: 1,
    path: parent ? `${parent.path}-${id}` : `${id}`,
    children: []
  }
}

/** 使兄弟排序从一连续递增，匹配 epark 整树保存规则。 */
export function resequenceOrgItems(items: OrgItem[]) {
  items.forEach((item, index) => {
    item.order = index + 1
  })
}

/** 用 96 位随机值生成稳定的 MongoDB ObjectID 格式草稿标识，避免整树重载后绑定错位。 */
export function nextOrgId() {
  return Array.from(crypto.getRandomValues(new Uint8Array(12)), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

/** 校验远端响应具有可遍历的节点数组。 */
export function isOrgData(value: unknown): value is { items: unknown } {
  return isPlainObject(value) && Array.isArray(value.items)
}

/** 排除数组与空值，保护响应边界解析。 */
export function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

/** 读取非空字符串，缺省字段使用业务默认值。 */
export function toStringValue(value: unknown, fallback: string) {
  return typeof value === 'string' && value.trim() ? value : fallback
}

/** 读取有限数字，避免无效排序值污染画布。 */
export function toNumber(value: unknown, fallback: number) {
  const next = Number(value)
  return Number.isFinite(next) ? next : fallback
}
