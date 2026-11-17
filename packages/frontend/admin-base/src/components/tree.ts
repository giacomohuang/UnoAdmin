/**
 * 权限树在 MillerColumns 中的扁平化与授权级联规则。
 *
 * MillerColumns 需要扁平数据源（id / parentId / sort），接口返回的是嵌套树，
 * 这里统一完成转换，页面与授权组件共用同一套遍历逻辑。
 */
import type { Permission } from '@unoadmin/core'

/** 扁平后的权限节点，附带 MillerColumns 需要的父级与排序字段。 */
export type PermissionNode = {
  id: string
  parentId: string | null
  name: string
  code: string
  type: Permission['type']
  sort: number
  childCount: number
} & Record<string, unknown>

/** 先序展开权限树；sort 为同级顺序，供 MillerColumns 排序列使用。 */
export function flattenPermissions(nodes: Permission[], parentId: string | null = null): PermissionNode[] {
  return nodes.flatMap((node, index) => [
    {
      id: node.id,
      parentId,
      name: node.name,
      code: node.code,
      type: node.type,
      sort: index,
      childCount: node.children?.length ?? 0
    },
    ...flattenPermissions(node.children ?? [], node.id)
  ])
}

/** 获取从根节点到目标节点的 ID 路径，遇到环或缺失节点时提前终止。 */
export function permissionPath(id: string | null, nodes: PermissionNode[]): string[] {
  const byId = new Map(nodes.map((node) => [node.id, node]))
  const result: string[] = []
  const visited = new Set<string>()
  let cursor = id
  while (cursor && byId.has(cursor) && !visited.has(cursor)) {
    visited.add(cursor)
    result.unshift(cursor)
    cursor = byId.get(cursor)?.parentId ?? null
  }
  return result
}

/** 把节点路径转换为「系统管理 / 角色管理」形式的名称路径。 */
export function permissionLabel(id: string | null, nodes: PermissionNode[]): string {
  const byId = new Map(nodes.map((node) => [node.id, node]))
  return permissionPath(id, nodes)
    .map((key) => byId.get(key)?.name ?? key)
    .join(' / ')
}

/** 广度优先收集全部后代 ID，避免环导致死循环。 */export function descendantIds(id: string, nodes: PermissionNode[]): string[] {
  const result: string[] = []
  const visited = new Set<string>([id])
  const pending = [id]
  while (pending.length) {
    const parent = pending.shift()
    for (const node of nodes) {
      if (node.parentId === parent && !visited.has(node.id)) {
        visited.add(node.id)
        result.push(node.id)
        pending.push(node.id)
      }
    }
  }
  return result
}

/** 权限类型对应的图标，菜单/页面/操作使用不同语义。 */
export function permissionIcon(node: { type: Permission['type'] }): string {
  if (node.type === 'action') return 'lucide:mouse-pointer-click'
  if (node.type === 'menu') return 'lucide:folder-tree'
  return 'lucide:file-text'
}

/** 活动行使用白色边框，避免 Checkbox 的品牌色轮廓融入 MillerColumns 的选中背景。 */
export function millerColumnsCheckboxClass(active: boolean): string {
  return active ? '!border-white' : ''
}

/**
 * 按父子级联规则切换授权，返回新的权限码数组。
 *
 * - 勾选父节点会连带选中全部后代；
 * - 取消父节点会取消整个子树；
 * - 勾选子节点前必须先勾选父节点。
 */
export function toggleGrant(id: string, codes: string[], nodes: PermissionNode[]): string[] {
  const node = nodes.find((item) => item.id === id)
  if (!node) return codes

  const selected = new Set(nodes.filter((item) => codes.includes(item.code)).map((item) => item.id))
  const below = descendantIds(id, nodes)

  if (selected.has(id)) {
    for (const key of [id, ...below]) selected.delete(key)
  } else {
    if (node.parentId && !selected.has(node.parentId)) return codes
    selected.add(id)
    for (const key of below) selected.add(key)
  }

  return nodes.filter((item) => selected.has(item.id)).map((item) => item.code)
}
