import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import OrgNode from '../../../admin-base/src/components/OrgNode.vue'
import type { OrgItem } from '../../../admin-base/src/api/organizations'

/** Build a two-level organization draft for the UnoMap node contract test. */
function draft(): OrgItem[] {
  return [
    {
      id: 'root',
      pid: null,
      entityId: 'entity-main',
      name: '总部',
      fullname: '总部',
      leaderName: '',
      leaderId: null,
      order: 1,
      type: 'company',
      path: 'root',
      status: 1,
      children: [
        {
          id: 'child-a',
          pid: 'root',
          entityId: 'entity-main',
          name: '研发部',
          fullname: '总部-研发部',
          leaderName: '',
          leaderId: null,
          order: 1,
          type: 'department',
          path: 'root-child-a',
          status: 1,
          children: [],
        },
        {
          id: 'child-b',
          pid: 'root',
          entityId: 'entity-main',
          name: '市场部',
          fullname: '总部-市场部',
          leaderName: '',
          leaderId: null,
          order: 2,
          type: 'department',
          path: 'root-child-b',
          status: 1,
          children: [],
        },
      ],
    },
  ]
}

describe('UnoMap organization node', () => {
  it('renders layered cards, connectors and insertion actions', async () => {
    const wrapper = mount(OrgNode, {
      props: {
        items: draft(),
        level: 1,
        selectedId: 'root',
        draggingId: '',
        previewTargetId: '',
        previewDirection: '',
        previewSize: { width: 190, height: 86, cardLeft: 0, cardTop: 0 },
        editable: true,
        entityNames: { 'entity-main': '总部' },
      },
    })

    expect(wrapper.findAll('[data-org-node-card]')).toHaveLength(3)
    expect(wrapper.findAll('[data-org-connector="horizontal"]')).toHaveLength(2)
    expect(wrapper.find('[data-org-id="root"]').classes()).toContain('ring-2')

    await wrapper.find('[title="添加下级"]').trigger('click')
    const emitted = wrapper.emitted<[unknown, string]>('add')
    expect(emitted?.[0]?.[1]).toBe('child')
  })
})
