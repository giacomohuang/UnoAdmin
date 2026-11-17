<script setup lang="ts">
/**
 * 组织树递归渲染组件。
 *
 * 每个节点自行维护展开状态，父级只负责传递选中项与事件，
 * 避免把整棵树的展开状态提升到页面层。
 */
import { Tag } from '@mcistudio/unoui-vue/tag'
import { Icon } from '@iconify/vue'
import type { OrgUnit } from '@unoadmin/core'
import { ref } from 'vue'

defineOptions({ name: 'OrgTree' })

const props = withDefaults(
  defineProps<{
    nodes: OrgUnit[]
    selectedId?: string
    level?: number
  }>(),
  { selectedId: '', level: 0 }
)

const emit = defineEmits<{
  select: [node: OrgUnit]
}>()

const expanded = ref<Set<string>>(new Set(props.nodes.map((node) => node.id)))

/** 切换节点展开状态。 */
function toggle(node: OrgUnit) {
  const next = new Set(expanded.value)
  if (next.has(node.id)) next.delete(node.id)
  else next.add(node.id)
  expanded.value = next
}

/** 组织类型的中文标签。 */
function typeLabel(type: string) {
  const labels: Record<string, string> = {
    company: '公司',
    department: '部门',
    team: '班组',
    store: '门店',
    project: '项目组'
  }
  return labels[type] ?? '组织'
}

/** 业务节点使用不同图标，便于快速识别层级含义。 */
function nodeIcon(type: string) {
  if (type === 'company') return 'lucide:building-2'
  if (type === 'store') return 'lucide:store'
  if (type === 'team') return 'lucide:users-round'
  if (type === 'project') return 'lucide:folder-kanban'
  return 'lucide:network'
}</script>

<template>
  <ul class="m-0 list-none p-0">
    <li v-for="node in props.nodes" :key="node.id">
      <div
        class="group flex h-9 cursor-pointer items-center gap-1.5 rounded-md pr-2 transition-colors"
        :class="props.selectedId === node.id ? 'bg-brand-50 text-brand-700' : 'hover:bg-secondary'"
        role="treeitem"
        :aria-selected="props.selectedId === node.id"
        tabindex="0"
        @click="emit('select', node)"
        @keydown.enter="emit('select', node)"
      >
        <button
          v-if="node.children?.length"
          type="button"
          class="grid h-5 w-5 shrink-0 place-items-center rounded border-0 bg-transparent text-tertiary hover:text-primary"
          :aria-label="expanded.has(node.id) ? '收起' : '展开'"
          @click.stop="toggle(node)"
        >
          <span :class="expanded.has(node.id) ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'" class="h-3.5 w-3.5" aria-hidden="true" />
        </button>
        <span v-else class="h-5 w-5 shrink-0" aria-hidden="true" />

        <Icon :icon="nodeIcon(node.type)" class="h-4 w-4 shrink-0" :class="props.selectedId === node.id ? 'text-brand' : 'text-tertiary'" />
        <span class="min-w-0 flex-1 truncate text-sm">{{ node.name }}</span>
        <span class="shrink-0 text-[11px] text-tertiary">{{ typeLabel(node.type) }}</span>
        <Tag v-if="node.status !== 'active'" color="gray" size="sm">停用</Tag>
      </div>

      <OrgTree
        v-if="expanded.has(node.id) && node.children?.length"
        class="ml-5 border-l border-light pl-1"
        :nodes="node.children"
        :selected-id="props.selectedId"
        :level="props.level + 1"
        @select="emit('select', $event)"
      />
    </li>
  </ul>
</template>
