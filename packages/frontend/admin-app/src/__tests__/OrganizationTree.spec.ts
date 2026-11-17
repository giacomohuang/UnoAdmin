import { describe, expect, it } from 'vitest'

import type { OrgItem } from '../../../admin-base/src/api/organizations'
import {
  addOrgItem,
  findOrgItem,
  flattenOrgItems,
  moveOrgItem,
  normalizeOrgData,
  removeOrgItem,
} from '../../../admin-base/src/components/organizationTree'

/** Build one API organization node for draft-tree tests. */
function orgNode(id: string, options: Partial<OrgItem> = {}): OrgItem {
  return {
    id,
    pid: null,
    entityId: 'entity-main',
    name: id,
    fullname: id,
    leaderName: '',
    leaderId: null,
    order: 1,
    type: 'department',
    path: id,
    status: 1,
    children: [],
    ...options,
  }
}

describe('organization draft tree', () => {
  it('adds child and sibling nodes and recomputes paths', () => {
    const draft = normalizeOrgData({ items: [orgNode('root')] })
    const child = addOrgItem(draft, 'root', 'child')
    const sibling = addOrgItem(draft, 'root', 'next')

    expect(child?.pid).toBe('root')
    expect(child?.path).toBe(`root-${child?.id}`)
    expect(sibling?.pid).toBeNull()
    expect(flattenOrgItems(draft.items)).toHaveLength(3)
  })

  it('moves a complete subtree without allowing cycles or root movement', () => {
    const draft = normalizeOrgData({ items: [
      orgNode('root', {
        children: [
          orgNode('a', { pid: 'root', path: 'root-a' }),
          orgNode('b', { pid: 'root', path: 'root-b' }),
          orgNode('c', { pid: 'root', path: 'root-c' }),
        ],
      }),
    ] })

    expect(moveOrgItem(draft, 'a', 'b', 'child')).toBe(true)
    expect(findOrgItem(draft.items, 'a')?.pid).toBe('b')
    expect(moveOrgItem(draft, 'b', 'c', 'previous')).toBe(true)
    expect(moveOrgItem(draft, 'b', 'a', 'child')).toBe(false)
    expect(moveOrgItem(draft, 'root', 'a', 'child')).toBe(false)
  })

  it('removes a non-root subtree while protecting roots', () => {
    const draft = normalizeOrgData({ items: [
      orgNode('root', {
        children: [
          orgNode('child', {
            pid: 'root',
            path: 'root-child',
            children: [orgNode('grandchild', { pid: 'child', path: 'root-child-grandchild' })],
          }),
        ],
      }),
    ] })

    expect(removeOrgItem(draft, 'root')).toBe(false)
    expect(removeOrgItem(draft, 'child')).toBe(true)
    expect(findOrgItem(draft.items, 'grandchild')).toBeNull()
  })
})
