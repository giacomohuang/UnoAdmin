<template>
  <ul class="flex list-none justify-center p-0 m-0" :data-org-node-list="ghost ? undefined : ''" :data-org-level="ghost ? undefined : level" :data-org-parent-id="ghost ? undefined : (parentId ?? '')">
    <li
      v-for="entry in renderEntries"
      :key="entry.key"
      class="relative flex flex-col items-center"
      :class="entry.connectorTotal > 1 ? 'px-5' : 'px-0'"
      :data-id="entry.kind === 'item' ? entry.item.id : undefined"
      :data-org-entry-kind="ghost ? undefined : entry.kind"
      :data-org-item-id="!ghost && entry.kind === 'item' ? entry.item.id : undefined"
    >
      <div v-if="!isGhostRoot && level > 1 && entry.connectorTotal > 1" data-org-connector="horizontal" class="absolute top-0" :class="horizontalLineClass(entry.connectorIndex, entry.connectorTotal)"></div>
      <div v-if="level > 1" data-org-connector="stem" class="h-5 w-px" :class="verticalLineClass(entry.connectorIndex, entry.connectorTotal)"></div>

      <div
        v-if="entry.kind === 'preview'"
        data-org-drop-preview
        class="pointer-events-none relative shrink-0"
        :style="{
          width: `${entry.size.width}px`,
          height: `${entry.size.height}px`
        }"
      >
        <div
          data-org-drop-preview-card
          class="absolute h-24 w-46 rounded-lg border-2 border-dashed border-brand bg-brand/12 shadow-[inset_0_0_0_1px_rgb(79_159_143_/_0.18)]"
          :style="{
            left: `${entry.size.cardLeft}px`,
            top: `${entry.size.cardTop}px`
          }"
        ></div>
      </div>

      <div v-else-if="entry.kind === 'item'" data-org-node-branch class="flex flex-col items-center">
        <div
          :data-org-node-card="ghost ? undefined : ''"
          :data-org-id="ghost ? undefined : entry.item.id"
          :data-org-ghost-card="ghost ? '' : undefined"
          :data-org-ghost-id="ghost ? entry.item.id : undefined"
          :role="ghost ? undefined : 'button'"
          :tabindex="ghost ? undefined : 0"
          :aria-pressed="ghost ? undefined : selectedId === entry.item.id"
          class="group/card relative z3 w-46 touch-none rounded-lg p-1 shadow-soft transition-all duration-150"
          :class="[
            ghost ? 'shadow-floating ring-2 ring-brand/30' : 'cursor-pointer hover:shadow-panel',
            levelClass(level),
            !ghost && selectedId === entry.item.id ? 'ring-2 ring-brand shadow-panel' : '',
            !ghost && draggingId === entry.item.id ? 'opacity-45 outline-2 outline-dashed outline-brand' : ''
          ]"
          @click.stop="selectNode(entry.item)"
          @keydown.enter.stop.prevent="selectNode(entry.item)"
          @keydown.space.stop.prevent="selectNode(entry.item)"
        >
          <div class="relative mb-1 flex h-6 items-center justify-between rounded-[5px_5px_3px_3px] px-1 text-xs text-white">
            <div class="relative z2 flex min-w-0 items-center gap-1">
              <span
                v-if="editable && level !== 1 && !ghost"
                data-org-interactive
                role="button"
                tabindex="0"
                class="flex size-5 shrink-0 cursor-move items-center justify-center rounded text-white/85 hover:bg-black/18 hover:text-white"
                title="拖拽移动部门"
                @pointerdown.stop.prevent="startDrag(entry.item, $event)"
                @click.stop
              >
                <span class="i-lucide:grip-vertical size-3.5"></span>
              </span>

              <span class="truncate">{{ entry.item.name }}</span>
            </div>

            <Button
              :variant="null"
              :size="null"
              :radius="null"
              v-if="editable && level !== 1 && !ghost"
              type="button"
              data-org-no-drag
              class="relative z2 hidden size-5 cursor-pointer items-center justify-center rounded border-0 bg-transparent text-white hover:bg-black/18 group-hover/card:flex"
              title="删除部门"
              @click.stop="emit('remove', entry.item)"
            >
              <span class="i-lucide:x size-3"></span>
            </Button>
          </div>

          <div class="flex min-h-16 w-full items-center gap-2 rounded bg-primary p-2 text-left text-sm text-primary">
            <span class="flex min-w-0 flex-1 flex-col gap-1">
              <span class="flex min-w-0 items-center gap-1">
                <span class="i-proicons:person-circle size-5 shrink-0 text-tertiary"></span>
                <span class="truncate" :title="entry.item.leaderName || '未设置负责人'">{{ entry.item.leaderName || '未设置负责人' }}</span>
              </span>
              <span v-if="entry.item.entityId" class="flex min-w-0 items-center gap-1 text-xs text-tertiary">
                <span class="i-lucide:shapes size-4 shrink-0"></span>
                <span class="truncate" :title="entityName(entry.item.entityId)">{{ entityName(entry.item.entityId) }}</span>
              </span>
            </span>
          </div>

          <div v-if="editable && !ghost" class="pointer-events-none invisible absolute inset-0 z5 group-hover/card:visible">
            <Button
              :variant="null"
              :size="null"
              :radius="null"
              type="button"
              data-org-no-drag
              class="pointer-events-auto absolute! bottom-0 left-1/2 flex size-4 -translate-x-1/2 translate-y-1/2 cursor-pointer items-center justify-center rounded-full border border-brand bg-primary text-brand shadow-panel hover:bg-brand hover:text-white"
              title="添加下级"
              @click.stop="emit('add', entry.item, 'child')"
            >
              <span class="i-lucide:plus size-3"></span>
            </Button>

            <Button
              :variant="null"
              :size="null"
              :radius="null"
              v-if="level !== 1"
              type="button"
              data-org-no-drag
              class="pointer-events-auto absolute! left-0 top-1/2 flex size-4 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border border-brand bg-primary text-brand shadow-panel hover:bg-brand hover:text-white"
              title="添加前置部门"
              @click.stop="emit('add', entry.item, 'previous')"
            >
              <span class="i-lucide:plus size-3"></span>
            </Button>

            <Button
              :variant="null"
              :size="null"
              :radius="null"
              v-if="level !== 1"
              type="button"
              data-org-no-drag
              class="pointer-events-auto absolute! right-0 top-1/2 flex size-4 translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border border-brand bg-primary text-brand shadow-panel hover:bg-brand hover:text-white"
              title="添加后置部门"
              @click.stop="emit('add', entry.item, 'next')"
            >
              <span class="i-lucide:plus size-3"></span>
            </Button>
          </div>
        </div>

        <div v-if="hasVisibleChildren(entry.item) || isPreviewChild(entry.item)" data-org-connector="branch" class="h-5 w-px bg-[#afaacb] dark:bg-zinc-600"></div>
        <OrgNode
          v-if="hasVisibleChildren(entry.item) || isPreviewChild(entry.item)"
          :items="entry.item.children"
          :level="level + 1"
          :selected-id="selectedId"
          :dragging-id="draggingId"
          :preview-target-id="previewTargetId"
          :preview-direction="previewDirection"
          :append-preview="isPreviewChild(entry.item)"
          :ghost="ghost"
          :preview-size="previewSize"
          :entity-names="entityNames"
          :editable="editable"
          :parent-id="entry.item.id"
          @add="(item, direction) => emit('add', item, direction)"
          @select="(item) => emit('select', item)"
          @remove="(item) => emit('remove', item)"
          @rename="(item, name) => emit('rename', item, name)"
          @drag-start="(item, event) => emit('drag-start', item, event)"
        />
      </div>
    </li>
  </ul>
</template>

<router lang="json">
{
  "isRouter": false
}
</router>

<script setup lang="ts">
import { Button } from '@mcistudio/unoui-vue/button'
/** 递归绘制 epark 部门卡片、连接线和拖动预览；不直接持久化数据。 */
import { computed } from 'vue'

import type { OrgItem } from '../api/organizations'

import type { OrgAddDirection } from './organizationTree'

/** 拖动占位保存源分支尺寸与卡片在分支内的偏移。 */
interface OrgPreviewSize {
  width: number
  height: number
  cardLeft: number
  cardTop: number
}

/** 渲染真实节点或源分支同尺寸的拖动占位。 */
type OrgBaseRenderEntry = { kind: 'item'; key: string; item: OrgItem } | { kind: 'preview'; key: string; size: OrgPreviewSize }

/** 按当前渲染条目计算的兄弟索引，保证连接线不断裂。 */
type OrgConnectorRenderEntry = OrgBaseRenderEntry & { connectorIndex: number; connectorTotal: number }
/** 递归节点组件消费的最终绘制条目。 */
type OrgRenderEntry = OrgConnectorRenderEntry

const props = withDefaults(
  defineProps<{
    items: OrgItem[]
    level: number
    parentId?: string | null
    selectedId: string
    draggingId: string
    previewTargetId: string
    previewDirection: OrgAddDirection | ''
    previewSize: OrgPreviewSize
    editable: boolean
    entityNames?: Record<string, string>
    appendPreview?: boolean
    ghost?: boolean
  }>(),
  {
    parentId: null,
    appendPreview: false,
    ghost: false
  }
)

defineOptions({
  name: 'OrgNode'
})

const emit = defineEmits<{
  add: [item: OrgItem, direction: OrgAddDirection]
  select: [item: OrgItem]
  remove: [item: OrgItem]
  rename: [item: OrgItem, name: string]
  'drag-start': [item: OrgItem, event: PointerEvent]
}>()

const isGhostRoot = computed(() => props.ghost && props.parentId === null)

/** 拖动时以等尺寸占位替换源分支，连接线按占位后的兄弟数重算。 */
const renderEntries = computed<OrgRenderEntry[]>(() => {
  const entries: OrgBaseRenderEntry[] = []

  for (const item of props.items) {
    if (!props.ghost && props.draggingId === item.id) continue
    if (isPreviewBefore(item)) entries.push(createPreviewEntry())
    entries.push({ kind: 'item', key: item.id, item })
    if (isPreviewAfter(item)) entries.push(createPreviewEntry())
  }

  if (props.appendPreview) entries.push(createPreviewEntry())
  return withConnectorIndexes(entries)
})

function isPreviewBefore(item: OrgItem) {
  return props.previewTargetId === item.id && props.previewDirection === 'previous'
}

function isPreviewAfter(item: OrgItem) {
  return props.previewTargetId === item.id && props.previewDirection === 'next'
}

function isPreviewChild(item: OrgItem) {
  return props.previewTargetId === item.id && props.previewDirection === 'child'
}

function hasVisibleChildren(item: OrgItem) {
  return item.children.some((child) => props.ghost || child.id !== props.draggingId)
}

function createPreviewEntry(): OrgBaseRenderEntry {
  return {
    kind: 'preview',
    key: `preview-${props.level}-${props.previewTargetId}-${props.previewDirection}`,
    size: props.previewSize
  }
}

function withConnectorIndexes(entries: OrgBaseRenderEntry[]): OrgRenderEntry[] {
  const connectorTotal = entries.length
  let connectorIndex = 0
  return entries.map((entry) => {
    const next = {
      ...entry,
      connectorIndex,
      connectorTotal
    }
    connectorIndex += 1
    return next
  })
}

function startDrag(item: OrgItem, event: PointerEvent) {
  if (!props.editable) return
  if (event.button !== 0) return
  emit('drag-start', item, event)
}

function selectNode(item: OrgItem) {
  if (props.ghost) return
  emit('select', item)
}

function levelClass(level: number) {
  const classes = ['bg-[#4f9f8f] dark:bg-[#347568]', 'bg-[#8b7acb] dark:bg-[#685b9f]', 'bg-[#d6816f] dark:bg-[#9f5a4f]', 'bg-[#6597ad] dark:bg-[#466d80]', 'bg-[#b98974] dark:bg-[#856154]', 'bg-[#7aa36f] dark:bg-[#57784f]', 'bg-[#d49d57] dark:bg-[#98713e]', 'bg-[#6f93c5] dark:bg-[#506f99]']
  return classes[(level - 1) % classes.length]
}

function entityName(entityId: string) {
  return props.entityNames?.[entityId] || `主体 #${entityId}`
}

function horizontalLineClass(index: number, total: number) {
  if (total <= 1) return ''
  if (index === 0) return 'left-1/2 right-0 h-5 rounded-tl-md border-l border-t border-[#afaacb] dark:border-zinc-600'
  if (index === total - 1) return 'left-0 right-1/2 h-5 rounded-tr-md border-r border-t border-[#afaacb] dark:border-zinc-600'
  return 'left-0 right-0 h-px bg-[#afaacb] dark:bg-zinc-600'
}

/** 首尾节点的竖线由圆角线段一并绘制，避免直线覆盖圆弧。 */
function verticalLineClass(index: number, total: number) {
  if (isGhostRoot.value || total <= 1) return isGhostRoot.value ? '' : 'bg-[#afaacb] dark:bg-zinc-600'
  return index === 0 || index === total - 1 ? 'invisible' : 'bg-[#afaacb] dark:bg-zinc-600'
}
</script>
