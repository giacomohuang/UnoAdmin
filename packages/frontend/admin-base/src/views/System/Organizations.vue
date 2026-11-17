<template>
  <PageHeader :loading="loading || reloading || saving" :status-visible="statusReady" class="px-4 pt-4">
    <template #title
      ><span class="font-700">组织架构</span></template
    >
    <template #status>{{ nodeCount }} 个部门</template>
    <template #description>最后保存：{{ savedAtText }}</template>
    <template #actions>
      <Button variant="mono" icon="i-lucide:refresh-cw" :loading="reloading" @click="reloadOrg">重新加载</Button>
      <Button v-if="canEditOrganization" icon="i-lucide:save" :loading="saving" @click="saveOrg">保存</Button>
    </template>
  </PageHeader>

  <Skeleton v-if="initialLoading" variant="list" :rows="8" min-height="620px" item-height="64px" padded class="h-[calc(100vh-162px)]" />
  <div v-else data-org-canvas class="relative overflow-hidden bg-primary dark:bg-zinc-900 h-[calc(100vh-126px)] min-h-0">
    <div ref="viewportRef" class="relative h-full min-h-0 cursor-grab overflow-scroll [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden active:cursor-grabbing" @mousedown="startCanvasDrag" @wheel="zoomByWheel">
      <div ref="canvasRef" class="relative flex items-center justify-center">
        <div class="h-fit w-fit shrink-0 select-none origin-top" :style="{ transform: `scale(${zoomPercent / 100})` }">
          <div ref="nodesRef" class="relative flex flex-col items-center">
            <OrgNode
              v-if="org.items.length"
              :items="org.items"
              :level="1"
              :selected-id="selectedNodeId"
              :dragging-id="draggingId"
              :preview-target-id="previewTargetId"
              :preview-direction="previewDirection"
              :preview-size="dragPreviewSize"
              :entity-names="entityNames"
              :editable="canEditOrganization"
              @add="handleAdd"
              @select="openOrgDrawer"
              @remove="handleRemove"
              @rename="handleRename"
              @drag-start="startNodeDrag"
            />
            <Button :variant="null" :size="null" :radius="null" v-else-if="canEditOrganization" type="button" class="h-auto! flex cursor-pointer items-center gap-2 rounded-lg! border! border-dashed! border-brand! bg-primary! px-5! py-4! text-brand hover:bg-brand/8!" @click="createRoot">
              <span class="i-lucide:plus size-5"></span>
              新建组织
            </Button>
            <div v-else class="text-sm text-tertiary">暂无组织</div>
          </div>
        </div>
      </div>
    </div>

    <div data-org-interactive data-org-no-zoom class="absolute bottom-5 left-5 z30 flex items-center gap-2 rounded-lg border border-light bg-primary p-2 shadow-floating">
      <Button variant="mono" size="icon" icon="i-lucide:zoom-out" title="缩小" @click="zoom('out')" />
      <Button :variant="null" :size="null" :radius="null" type="button" class="font-700! min-w-[4.1428571429rem] cursor-pointer rounded-md border-0 bg-transparent px-2 text-secondary font-700 leading-8 hover:(bg-secondary text-primary)" title="重置缩放" @click="zoom('reset')"
        >{{ zoomPercent }}%</Button
      >
      <Button variant="mono" size="icon" icon="i-lucide:zoom-in" title="放大" @click="zoom('in')" />
      <Button variant="mono" size="icon" icon="i-lucide:scan" title="居中" @click="centerCanvas" />
    </div>

    <!-- Ghost 容器仅负责定位与透明度过渡，视觉层级由节点组件统一绘制。 -->
    <div
      v-if="dragGhost.visible && dragGhost.item"
      ref="dragGhostRef"
      class="pointer-events-none fixed z1000 w-fit origin-top-left select-none transition-opacity duration-75 will-change-transform"
      :class="dragGhost.ready ? 'opacity-95' : 'opacity-0'"
      :style="{ left: `${dragGhost.x}px`, top: `${dragGhost.y}px`, transform: `scale(${dragGhost.scale})` }"
    >
      <OrgNode :items="[dragGhost.item]" :level="dragGhost.level" selected-id="" dragging-id="" preview-target-id="" preview-direction="" :preview-size="dragPreviewSize" :entity-names="entityNames" :editable="canEditOrganization" ghost />
    </div>

    <Drawer v-model:visible="drawerVisible" :title="drawerTitle" size="min(400px, 100vw)" body-class="bg-primary">
      <template #header>
        <div class="min-w-0">
          <h3 class="m-0 truncate text-lg font-700 text-primary">{{ drawerTitle }}</h3>
          <div class="mt-1 truncate text-xs text-tertiary">{{ selectedNodeFullName }}</div>
        </div>
      </template>

      <div class="min-h-full p-4">
        <Form :model="form" label-position="top" size="sm" item-gap="8px">
          <section class="min-w-0">
            <div class="mb-3 flex min-h-7 items-center">
              <h4 class="m-0 text-base font-700 text-primary">部门信息</h4>
            </div>
            <FormItem label="部门全称">
              <div class="min-h-7.5 rounded-md bg-secondary px-3 py-1.5 text-sm text-secondary">{{ selectedNodeFullName }}</div>
            </FormItem>
            <FormItem label="部门名称" required>
              <Input v-model="form.name" :disabled="!canEditOrganization" autocomplete="off" />
            </FormItem>
            <FormItem label="关联主体">
              <Select v-model="form.entityId" :options="entityOptions" :loading="entitiesLoading" :disabled="!canEditOrganization" clearable filterable class="w-full" placeholder="请选择主体" />
            </FormItem>
          </section>

          <section class="mt-4 min-w-0 border-t border-light pt-4">
            <div class="mb-2 flex min-h-7 flex-wrap items-center justify-between gap-x-3 gap-y-2">
              <div class="flex min-w-0 items-baseline gap-3">
                <h4 class="m-0 shrink-0 text-base font-700 text-primary">员工列表</h4>
                <span class="text-xs text-tertiary">{{ drawerEmployees.length }} 人</span>
              </div>
              <div class="flex min-w-0 items-center gap-2">
                <span class="min-w-0 truncate text-xs text-tertiary">负责人：{{ currentLeaderText }}</span>
                <Button v-if="canEditOrganization && hasLeaderDraft" variant="link" size="sm" icon="i-lucide:x" @click="clearLeader">清除负责人</Button>
              </div>
            </div>

            <div class="max-h-25 overflow-auto rounded-md border border-light bg-primary p-1">
              <div v-if="drawerEmployeesLoading" class="flex min-h-40 items-center justify-center px-3 py-6" aria-busy="true">
                <span class="i-lucide:loader-circle size-5 animate-spin text-brand" aria-label="加载中"></span>
              </div>
              <div v-else-if="drawerEmployees.length" class="space-y-1">
                <div v-for="employee in drawerEmployees" :key="`${employee.id}-${employee.organizationId}`" class="grid min-h-13 grid-cols-[minmax(0,1fr)_96px] items-center gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-secondary">
                  <div class="min-w-0">
                    <div class="flex min-w-0 items-center gap-2">
                      <span class="max-w-32 shrink-0 truncate text-sm font-700 text-primary">{{ employee.displayName }}</span>
                      <span class="truncate font-mono text-xs text-tertiary">{{ employee.username }}</span>
                      <Tag v-if="employee.leaderSelectable && form.leaderId === employee.id" class="text-2.5!">负责人</Tag>
                    </div>
                    <div class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-tertiary">
                      <span class="i-lucide:building-2 shrink-0 text-[0.9285714286rem] text-quaternary"></span>
                      <span class="min-w-0 truncate">{{ employee.organizationFullName || employee.organizationName }}</span>
                    </div>
                  </div>
                  <div class="flex h-full items-center justify-end">
                    <Button v-if="canEditOrganization && employee.leaderSelectable && form.leaderId !== employee.id" variant="link" size="sm" @click="setLeader(employee)">设为负责人</Button>
                  </div>
                </div>
              </div>
              <div v-else class="flex min-h-40 flex-col items-center justify-center px-3 py-6 text-center text-sm text-tertiary">
                <span class="i-lucide:users mb-3 text-xs text-quaternary"></span>
                当前部门暂无员工
              </div>
            </div>
          </section>
        </Form>
      </div>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <Button variant="mono" @click="closeDrawer">{{ canEditOrganization ? '取消' : '关闭' }}</Button>
          <Button v-if="canEditOrganization" :disabled="drawerEmployeesLoading" @click="submitEditor">确定</Button>
        </div>
      </template>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
/** 完整移植 epark 组织画布、拖动插槽和抽屉交互，使用 UnoMap 会话与 API。 */
import { Button } from '@mcistudio/unoui-vue/button'
import { Drawer } from '@mcistudio/unoui-vue/drawer'
import { Form, FormItem } from '@mcistudio/unoui-vue/form'
import { Input } from '@mcistudio/unoui-vue/input'
import { message } from '@mcistudio/unoui-vue/message'
import { Select, type SelectOption } from '@mcistudio/unoui-vue/select'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Tag } from '@mcistudio/unoui-vue/tag'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import type { OrgItem, OrganizationEmployeeRecord, OrganizationRecord } from '../../api/organizations'
import { createOrganizationApi, errorMessage } from '../../api/organizations'
import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

import {
  containsOrgItem,
  normalizeOrgData,
  createRootOrgData,
  cloneOrgData,
  cloneOrgItem,
  countOrgItems,
  flattenOrgItems,
  findOrgItem,
  findOrgParent,
  getOrgLevel,
  addOrgItem,
  removeOrgItem,
  moveOrgItem,
  updateOrgItem,
  refreshOrgTree,
  toOrgForm,
  type OrgData,
  type OrgAddDirection,
  type OrgPointer,
  type OrgFormModel
} from '../../components/organizationTree'
import OrgNode from '../../components/OrgNode.vue'

/** 实际测量的卡片与整支分支边界，支持不同子树宽度的拖动占位。 */
interface OrgDragTarget {
  item: OrgItem
  rect: OrgMeasuredRect
  cardWidth: number
  cardHeight: number
  branchWidth: number
  branchHeight: number
  cardLeft: number
  cardTop: number
  visibleChildCount: number
}

/** 拖动占位保存源分支尺寸与卡片在分支内的偏移。 */
interface OrgPreviewSize {
  width: number
  height: number
  cardLeft: number
  cardTop: number
}

/** 当前插入意图，由目标节点和相对方向组成。 */
interface OrgDropPreview {
  target: OrgDragTarget
  direction: OrgAddDirection
}

/** 同级节点行的命中快照，包含预览占位和父卡片。 */
interface OrgSiblingListTarget {
  level: number
  parentId: string | null
  parent: OrgDragTarget | null
  rowRect: OrgMeasuredRect
  entries: OrgSiblingListEntry[]
  items: OrgSiblingListItem[]
}

/** 同级行的实际渲染条目；占位与真实节点使用相同几何参与计算。 */
interface OrgSiblingListEntry {
  kind: 'item' | 'preview'
  target: OrgDragTarget | null
  rect: OrgMeasuredRect
  rowRect: OrgMeasuredRect
}

/** 可作为同级插槽锚点的节点及其完整横向占用。 */
interface OrgSiblingListItem {
  target: OrgDragTarget
  rect: OrgMeasuredRect
}

/** 统一视口几何，额外保留中心点用于拖动意图评分。 */
interface OrgMeasuredRect {
  left: number
  right: number
  top: number
  bottom: number
  width: number
  height: number
  centerX: number
  centerY: number
}

/** epark 命中优先级：优先同级插槽，随后下级和距离回退。 */
type OrgDropIntent = 'current-preview' | 'same-level-slot' | 'empty-child' | 'same-level' | 'child' | 'fallback'

/** 候选插槽及其分数；粘滞阈值用来避免边界抖动。 */
interface OrgDropCandidate extends OrgDropPreview {
  intent: OrgDropIntent
  score: number
}

const sameLevelOverlapRatio = 0.56
const sameLevelCenterRatio = 0.38
const sameLevelHorizontalRatio = 1.65
const childCenterOffsetRatio = 0.46
const childHorizontalRatio = 1.6
const emptyChildHorizontalRatio = 0.74
const fallbackDistanceLimit = 240
const dropStickyScore = 24
const slotStickyScore = 96
const orgCardWidth = 190
const orgCardHeight = 86

const { session } = useAdminBaseContext()
const services = useAdminBaseServices()
const entityId = ref<string>(session.state.currentLegalEntity?.id ?? '')
const organizationApi = createOrganizationApi({ services, session, entityId })
const treeVersion = ref(0)
const persistedIds = ref(new Set<string>())
const loading = ref(false)
const statusReady = ref(false)
const initialLoading = ref(true)
const saving = ref(false)
const reloading = ref(false)
const entitiesLoading = ref(false)
const dirty = ref(false)
const org = ref<OrgData>({ items: [] })
const entityOptions = ref<SelectOption[]>([])
const entityNames = ref<Record<string, string>>({})
const savedAt = ref('')
const viewportRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)
const nodesRef = ref<HTMLElement | null>(null)
const dragGhostRef = ref<HTMLElement | null>(null)
const zoomPercent = ref(100)
const selectedNodeId = ref('')
const draggingId = ref('')
const previewTargetId = ref('')
const previewDirection = ref<OrgAddDirection | ''>('')
const dragPreviewSize = reactive<OrgPreviewSize>({
  width: 190,
  height: 86,
  cardLeft: 0,
  cardTop: 0
})
const dragCardSize = reactive({
  width: 190,
  height: 86
})
const dragTargets = ref<OrgDragTarget[]>([])
const dragListTargets = ref<OrgSiblingListTarget[]>([])
const drawerVisible = ref(false)
const drawerEmployees = ref<OrganizationEmployeeRecord[]>([])
const drawerEmployeesLoading = ref(false)
const form = reactive<OrgFormModel>({
  id: '',
  entityId: undefined,
  name: '',
  leaderId: null,
  leaderName: ''
})
let drawerEmployeeRequestId = 0
const canvasDragState = reactive({
  dragging: false,
  startX: 0,
  startY: 0,
  scrollLeft: 0,
  scrollTop: 0,
  hasMoved: false
})
const nodeDragState = reactive({
  sourceId: '',
  startX: 0,
  startY: 0,
  pointerX: 0,
  pointerY: 0,
  hasMoved: false,
  sourceLevel: 1,
  sourceParentId: null as string | null,
  sourceRect: null as OrgMeasuredRect | null,
  sourceLocalRect: null as OrgMeasuredRect | null
})
const dragGhost = reactive({
  visible: false,
  ready: false,
  item: null as OrgItem | null,
  level: 1,
  x: 0,
  y: 0,
  offsetX: 0,
  offsetY: 0,
  grabX: 0,
  grabY: 0,
  scale: 1
})

const nodeCount = ref(0)
const canEditOrganization = computed(() => session.can('org:view') && session.can('org:edit'))
const selectedNode = computed(() => (selectedNodeId.value ? findOrgItem(org.value.items, selectedNodeId.value) : null))
const drawerTitle = computed(() => form.name.trim() || selectedNode.value?.name || '部门详情')
/** 抽屉名称即时预览完整部门路径，父级仍取已确认的树草稿。 */
const selectedNodeFullName = computed(() => {
  const node = selectedNode.value
  if (!node) return '-'
  const name = form.name.trim() || node.name || '未命名部门'
  const parent = findOrgParent(org.value.items, node.id)
  return parent ? `${parent.fullname}-${name}` : name
})
const currentLeaderText = computed(() => form.leaderName.trim() || '未设置')
const hasLeaderDraft = computed(() => form.leaderId !== null || Boolean(form.leaderName.trim()))
const savedAtText = computed(() => {
  if (dirty.value) return '有未保存修改'
  if (!savedAt.value) return '尚未保存'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(savedAt.value))
})

onMounted(() => {
  window.addEventListener('resize', centerCanvas)
  void loadOrg()
  void loadEntities()
  void nextTick(centerCanvas)
})

onBeforeUnmount(() => {
  drawerEmployeeRequestId += 1
  window.removeEventListener('resize', centerCanvas)
  stopCanvasDrag()
  stopNodeDrag()
})

/** 首次加载才显示骨架屏，后续操作保留当前画布。 */
async function loadOrg() {
  statusReady.value = false
  loading.value = true
  try {
    applyOrg(await organizationApi.get())
    dirty.value = false
    void nextTick(centerCanvas)
  } catch (error) {
    message.error(errorMessage(error))
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

/** 原子提交整树；版本冲突保留本地草稿，成功后刷新账户范围。 */
async function saveOrg() {
  if (!ensureOrganizationEditPermission()) return
  if (saving.value || reloading.value || loading.value) return
  statusReady.value = false
  saving.value = true
  try {
    applyOrg(await organizationApi.save({ items: org.value.items, expectedTreeVersion: treeVersion.value }))
    await session.refresh()
    dirty.value = false
    message.success('组织架构已保存')
  } catch (error) {
    message.error(errorMessage(error))
  } finally {
    saving.value = false
  }
}

/** 主动丢弃本地草稿并恢复服务端快照，保持 epark 重新加载行为。 */
async function reloadOrg() {
  if (reloading.value || saving.value || loading.value) return
  statusReady.value = false
  reloading.value = true
  try {
    applyOrg(await organizationApi.get())
    dirty.value = false
    message.success('已重新加载组织架构')
    void nextTick(centerCanvas)
  } catch (error) {
    message.error(errorMessage(error))
  } finally {
    reloading.value = false
  }
}

/** 读取组织管理允许关联的主体，无需额外持有主体管理权限。 */
async function loadEntities() {
  entitiesLoading.value = true
  try {
    const entities = await organizationApi.entities()
    entityOptions.value = entities.map((entity) => ({
      label: `${entity.name} · ${entity.code}`,
      value: entity.id
    }))
    entityNames.value = Object.fromEntries(entities.map((entity) => [entity.id, entity.name]))
  } catch (error) {
    message.error(errorMessage(error))
  } finally {
    entitiesLoading.value = false
  }
}

/** 保存返回的版本和持久化标识供下一次整树提交与员工查询使用。 */
function applyOrg(record: OrganizationRecord) {
  treeVersion.value = record.version
  persistedIds.value = new Set(flattenOrgItems(record.items).map((item) => item.id))
  org.value = normalizeOrgData(record)
  nodeCount.value = countOrgItems(record.items)
  statusReady.value = true
  savedAt.value = record.updatedAt || ''
  syncSelectedDrawer()
}

/** 新增节点只写入本地草稿并刷新画布，避免打断连续添加操作。 */
function handleAdd(item: OrgItem, direction: OrgAddDirection) {
  if (!ensureOrganizationEditPermission()) return
  const next = addOrgItem(org.value, item.id, direction)
  if (!next) return
  syncOrg()
}

function handleRemove(item: OrgItem) {
  if (!ensureOrganizationEditPermission()) return
  const removingSelectedNode = selectedNodeId.value === item.id
  if (!removeOrgItem(org.value, item.id)) return
  syncOrg()
  if (removingSelectedNode) {
    selectedNodeId.value = ''
    drawerEmployees.value = []
    closeDrawer()
  }
}

function handleRename(item: OrgItem, name: string) {
  if (!ensureOrganizationEditPermission()) return
  const target = findOrgItem(org.value.items, item.id)
  if (!target) return
  target.name = name
  refreshOrgTree(org.value)
  syncOrg()
}

/** 只允许移动非根节点；首次越过拖动阈值后才创建幽灵分支。 */
function startNodeDrag(item: OrgItem, event: PointerEvent) {
  if (!ensureOrganizationEditPermission()) return
  if (item.pid === null) return
  event.preventDefault()
  event.stopPropagation()
  stopCanvasDrag()
  nodeDragState.sourceId = item.id
  nodeDragState.startX = event.clientX
  nodeDragState.startY = event.clientY
  nodeDragState.pointerX = event.clientX
  nodeDragState.pointerY = event.clientY
  nodeDragState.hasMoved = false
  nodeDragState.sourceLevel = getOrgLevel(item)
  nodeDragState.sourceParentId = item.pid
  nodeDragState.sourceRect = null
  nodeDragState.sourceLocalRect = null
  draggingId.value = ''
  dragTargets.value = []
  dragListTargets.value = []
  dragGhost.visible = false
  dragGhost.ready = false
  dragGhost.item = null
  previewTargetId.value = ''
  previewDirection.value = ''
  document.addEventListener('pointermove', dragNode)
  document.addEventListener('pointerup', dropNode, true)
  document.addEventListener('pointercancel', cancelNodeDrag, true)
}

/** 创建首个根节点时保持画布焦点，详情仍由用户点击节点后打开。 */
function createRoot() {
  if (!ensureOrganizationEditPermission()) return
  org.value = createRootOrgData()
  syncOrg()
}

function getDropDirection(target: OrgDragTarget, dragRect: OrgMeasuredRect): OrgAddDirection {
  if (target.item.pid === null) return 'child'
  const source = findOrgItem(org.value.items, nodeDragState.sourceId)
  const offsetX = dragRect.centerX - target.rect.centerX
  if (Math.abs(offsetX) <= target.rect.width * 0.12 && source?.pid === target.item.pid) {
    return source.order < target.item.order ? 'next' : 'previous'
  }
  return offsetX < 0 ? 'previous' : 'next'
}

function activateNodeDrag(event: OrgPointer) {
  const sourceId = nodeDragState.sourceId
  const item = findOrgItem(org.value.items, sourceId)
  if (!item) return
  const source = getSourceDragTarget(sourceId)
  if (source) {
    setDragPreviewSize(source)
    nodeDragState.sourceRect = source.rect
    nodeDragState.sourceLocalRect = toCanvasLocalRect(source.rect)
  } else {
    resetDragPreviewSize()
    nodeDragState.sourceRect = null
    nodeDragState.sourceLocalRect = null
  }
  draggingId.value = sourceId
  dragTargets.value = []
  dragListTargets.value = []
  previewTargetId.value = ''
  previewDirection.value = ''
  resetDragGhostPosition(item, event, source)
  void nextTick(() => {
    alignDragGhostToPointer(event)
    dragTargets.value = collectDragTargets(sourceId)
    dragListTargets.value = collectSiblingListTargets(dragTargets.value)
    let nextPreview = updateDropPreview(event, { refreshTargets: false })
    if (!nextPreview) {
      nextPreview = getOriginalDropPreview(sourceId, dragTargets.value)
      previewTargetId.value = nextPreview?.target.item.id || ''
      previewDirection.value = nextPreview?.direction || ''
    }
    resizeCanvas()
  })
}

function resetDragGhostPosition(item: OrgItem, event: OrgPointer, source: OrgDragTarget | null) {
  const rect = source?.rect
  dragGhost.item = cloneOrgItem(item)
  dragGhost.level = nodeDragState.sourceLevel
  dragGhost.visible = true
  dragGhost.ready = false
  dragGhost.scale = rect && source ? rect.width / source.cardWidth : getCanvasScale()
  dragGhost.grabX = rect ? event.clientX - rect.left : (dragCardSize.width / 2) * dragGhost.scale
  dragGhost.grabY = rect ? event.clientY - rect.top : (dragCardSize.height / 2) * dragGhost.scale
  dragGhost.offsetX = dragGhost.grabX
  dragGhost.offsetY = dragGhost.grabY
  dragGhost.x = event.clientX - dragGhost.offsetX
  dragGhost.y = event.clientY - dragGhost.offsetY
}

function alignDragGhostToPointer(event: OrgPointer) {
  const root = dragGhostRef.value
  if (!root || !dragGhost.item) return
  const card = root.querySelector<HTMLElement>(`[data-org-ghost-card][data-org-ghost-id="${dragGhost.item.id}"]`)
  if (!card) return
  const rootRect = root.getBoundingClientRect()
  const cardRect = card.getBoundingClientRect()
  dragGhost.offsetX = cardRect.left - rootRect.left + dragGhost.grabX
  dragGhost.offsetY = cardRect.top - rootRect.top + dragGhost.grabY
  dragGhost.x = event.clientX - dragGhost.offsetX
  dragGhost.y = event.clientY - dragGhost.offsetY
  dragGhost.ready = true
}

function getSourceDragTarget(id: string) {
  const card = document.querySelector<HTMLElement>(`[data-org-node-card][data-org-id="${id}"]`)
  return card ? toDragTarget(card) : null
}

function setDragPreviewSize(target: OrgDragTarget) {
  dragPreviewSize.width = target.branchWidth
  dragPreviewSize.height = target.branchHeight
  dragPreviewSize.cardLeft = target.cardLeft
  dragPreviewSize.cardTop = target.cardTop
  dragCardSize.width = target.cardWidth
  dragCardSize.height = target.cardHeight
}

function resetDragPreviewSize() {
  dragPreviewSize.width = orgCardWidth
  dragPreviewSize.height = orgCardHeight
  dragPreviewSize.cardLeft = 0
  dragPreviewSize.cardTop = 0
  dragCardSize.width = orgCardWidth
  dragCardSize.height = orgCardHeight
}

function dragNode(event: PointerEvent) {
  if (!nodeDragState.sourceId) return
  event.preventDefault()
  nodeDragState.pointerX = event.clientX
  nodeDragState.pointerY = event.clientY
  const moved = Math.abs(event.clientX - nodeDragState.startX) + Math.abs(event.clientY - nodeDragState.startY)
  if (!nodeDragState.hasMoved) {
    if (moved <= 4) return
    nodeDragState.hasMoved = true
    activateNodeDrag(event)
    return
  }
  dragGhost.visible = nodeDragState.hasMoved
  if (nodeDragState.hasMoved && !dragGhost.ready) alignDragGhostToPointer(event)
  dragGhost.x = event.clientX - dragGhost.offsetX
  dragGhost.y = event.clientY - dragGhost.offsetY
  updateDropPreview(event, { refreshTargets: false })
}

/** 使用最后一个有效预览提交本地移动，并吞掉 pointerup 后的误点击。 */
function dropNode(event: PointerEvent) {
  event.preventDefault()
  const sourceId = nodeDragState.sourceId
  const hasMoved = nodeDragState.hasMoved
  const fallback = hasMoved && previewTargetId.value ? null : hasMoved ? getDropPreview(event) : null
  const targetId = previewTargetId.value || fallback?.target.item.id || ''
  const direction = previewDirection.value || fallback?.direction || ''
  stopNodeDrag()
  if (hasMoved) suppressNextClick()
  if (!sourceId || !hasMoved || !targetId || !direction || targetId === sourceId) return
  if (moveOrgItem(org.value, sourceId, targetId, direction)) syncOrg()
}

function cancelNodeDrag() {
  stopNodeDrag()
}

function stopNodeDrag() {
  nodeDragState.sourceId = ''
  nodeDragState.pointerX = 0
  nodeDragState.pointerY = 0
  nodeDragState.hasMoved = false
  nodeDragState.sourceLevel = 1
  nodeDragState.sourceParentId = null
  nodeDragState.sourceRect = null
  nodeDragState.sourceLocalRect = null
  draggingId.value = ''
  dragGhost.visible = false
  dragGhost.ready = false
  dragGhost.item = null
  previewTargetId.value = ''
  previewDirection.value = ''
  dragTargets.value = []
  dragListTargets.value = []
  document.removeEventListener('pointermove', dragNode)
  document.removeEventListener('pointerup', dropNode, true)
  document.removeEventListener('pointercancel', cancelNodeDrag, true)
}

function updateDropPreview(event: OrgPointer, options: { refreshTargets?: boolean } = {}) {
  const { targets, listTargets } = options.refreshTargets === false ? getDropTargetSnapshot() : refreshDropTargets()
  const preview = getDropPreview(event, { targets, listTargets })
  previewTargetId.value = preview?.target.item.id || ''
  previewDirection.value = preview?.direction || ''
  return preview
}

function getDropPreview(event: OrgPointer, options: { dragRect?: OrgMeasuredRect | null; targets?: OrgDragTarget[]; listTargets?: OrgSiblingListTarget[] } = {}): OrgDropPreview | null {
  const dragRect = options.dragRect ?? getCurrentDragRect(event)
  if (!dragRect) return null
  const targets = options.targets ?? getDropTargets()
  const candidate = findDropCandidate(dragRect, targets, options.listTargets ?? getDropListTargets(targets))
  if (!candidate) return null
  return {
    target: candidate.target,
    direction: candidate.direction
  }
}

function canMoveToTarget(target: OrgItem) {
  const source = findOrgItem(org.value.items, nodeDragState.sourceId)
  if (!source || source.id === target.id) return false
  return !containsOrgItem(source, target.id)
}

function getDropTargets() {
  return dragTargets.value.length ? dragTargets.value : collectDragTargets(nodeDragState.sourceId)
}

function getDropListTargets(targets: OrgDragTarget[]) {
  return dragListTargets.value.length ? dragListTargets.value : collectSiblingListTargets(targets)
}

function refreshDropTargets() {
  const targets = collectDragTargets(nodeDragState.sourceId)
  const listTargets = collectSiblingListTargets(targets)
  dragTargets.value = targets
  dragListTargets.value = listTargets
  return { targets, listTargets }
}

function getDropTargetSnapshot() {
  const targets = getDropTargets()
  return {
    targets,
    listTargets: getDropListTargets(targets)
  }
}

/** 保持 epark 插槽优先级及粘滞阈值，避免拖动时同级/下级预览跳变。 */
function findDropCandidate(dragRect: OrgMeasuredRect, targets: OrgDragTarget[], listTargets: OrgSiblingListTarget[]) {
  const candidates = listTargets.flatMap((list) => {
    const candidate = createSameLevelSlotCandidate(list, dragRect)
    return candidate ? [candidate] : []
  })
  const currentPreviewCandidate = previewDirection.value === 'child' ? null : createCurrentDropPreviewCandidate(dragRect, targets, listTargets)
  if (currentPreviewCandidate) candidates.push(currentPreviewCandidate)
  candidates.push(
    ...targets.flatMap((target) => {
      if (!canMoveToTarget(target.item)) return []
      const candidate = createDropCandidate(target, dragRect)
      return candidate ? [candidate] : []
    })
  )
  if (!candidates.length) return null

  candidates.sort(compareDropCandidates)
  const best = candidates[0]
  if (!best) return null
  const current = candidates.find((candidate) => candidate.target.item.id === previewTargetId.value && candidate.direction === previewDirection.value)
  if (current && getDropIntentRank(current.intent) === getDropIntentRank(best.intent) && current.score <= best.score + getDropStickyScore(current.intent)) return current
  return best
}

function createSameLevelSlotCandidate(list: OrgSiblingListTarget, dragRect: OrgMeasuredRect): OrgDropCandidate | null {
  if (list.level <= 1 || list.items.length === 0) return null
  if (list.level > nodeDragState.sourceLevel && !hasDownwardLevelIntent(dragRect)) return null
  if (!isDragInSiblingRow(dragRect, list.rowRect)) return null

  const items = list.items
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) return null

  if (!isDragInSiblingListIntentBounds(list, dragRect)) return null

  const slotCandidates = createSameLevelSlotCandidates(list, dragRect)
  if (!slotCandidates.length) return null
  slotCandidates.sort((a, b) => a.score - b.score)
  return slotCandidates[0] || null
}

function createSameLevelSlotCandidates(list: OrgSiblingListTarget, dragRect: OrgMeasuredRect) {
  const candidates: OrgDropCandidate[] = []
  const items = list.items
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) return candidates

  pushSameLevelSlotCandidate(candidates, list, first, 'previous', first.rect.left, dragRect)

  for (let index = 0; index < items.length - 1; index += 1) {
    const previous = items[index]
    const next = items[index + 1]
    if (!previous || !next) continue
    pushSameLevelSlotCandidate(candidates, list, previous, 'next', (previous.rect.right + next.rect.left) / 2, dragRect)
  }

  pushSameLevelSlotCandidate(candidates, list, last, 'next', last.rect.right, dragRect)
  const previewCandidate = createCurrentPreviewSlotCandidate(list, dragRect)
  if (previewCandidate) candidates.push(previewCandidate)
  return candidates
}

function getSiblingListIntentBounds(list: OrgSiblingListTarget) {
  const left = Math.min(...list.items.map((item) => item.rect.left))
  const right = Math.max(...list.items.map((item) => item.rect.right))
  return { left, right }
}

function isDragInSiblingListIntentBounds(list: OrgSiblingListTarget, dragRect: OrgMeasuredRect) {
  const bounds = getSiblingListIntentBounds(list)
  const horizontalPadding = Math.max(dragRect.width * 1.2, scaleCanvasDistance(220))
  if (dragRect.centerX < bounds.left - horizontalPadding || dragRect.centerX > bounds.right + horizontalPadding) return false

  const parentRect = list.parent?.rect
  if (!parentRect || list.level <= 2) return true
  const parentPadding = Math.max(parentRect.width * 1.15, dragRect.width, scaleCanvasDistance(230))
  return dragRect.centerX >= parentRect.centerX - parentPadding && dragRect.centerX <= parentRect.centerX + parentPadding
}

function pushSameLevelSlotCandidate(candidates: OrgDropCandidate[], list: OrgSiblingListTarget, item: OrgSiblingListItem, direction: OrgAddDirection, slotX: number, dragRect: OrgMeasuredRect) {
  const target = item.target
  if (!canMoveToTarget(target.item)) return null
  const verticalDistance = getVerticalDistanceToRect(dragRect.centerY, list.rowRect)
  const horizontalDistance = Math.abs(dragRect.centerX - slotX)
  const crossParentPenalty = getCrossParentSlotPenalty(list, dragRect)
  candidates.push({
    target,
    direction,
    intent: 'same-level-slot',
    score: verticalDistance * 6 + horizontalDistance * 0.45 + crossParentPenalty
  })
}

function getCrossParentSlotPenalty(list: OrgSiblingListTarget, dragRect: OrgMeasuredRect) {
  const parentRect = list.parent?.rect
  if (!parentRect || list.level <= 2) return 0
  const horizontalLimit = Math.max(parentRect.width * 1.05, dragRect.width * 0.9, scaleCanvasDistance(210))
  const horizontalOverflow = Math.max(0, Math.abs(dragRect.centerX - parentRect.centerX) - horizontalLimit)
  const verticalOverflow = Math.max(0, parentRect.bottom - dragRect.centerY)
  return horizontalOverflow * 1.35 + verticalOverflow * 3
}

function createCurrentPreviewSlotCandidate(list: OrgSiblingListTarget, dragRect: OrgMeasuredRect): OrgDropCandidate | null {
  const previewIndex = list.entries.findIndex((entry) => entry.kind === 'preview')
  if (previewIndex < 0) return null
  const preview = list.entries[previewIndex]
  if (!preview) return null

  const previous = findSiblingEntryTarget(list.entries, previewIndex, -1)
  const next = findSiblingEntryTarget(list.entries, previewIndex, 1)
  const target = previous || next
  const direction: OrgAddDirection = previous ? 'next' : 'previous'
  if (!target || previewTargetId.value !== target.item.id || previewDirection.value !== direction || !canMoveToTarget(target.item)) return null

  const verticalDistance = getVerticalDistanceToRect(dragRect.centerY, preview.rowRect)
  const horizontalDistance = Math.abs(dragRect.centerX - preview.rect.centerX)
  const crossParentPenalty = getCrossParentSlotPenalty(list, dragRect)
  return {
    target,
    direction,
    intent: 'same-level-slot',
    score: verticalDistance * 4 + horizontalDistance * 0.45 + crossParentPenalty
  }
}

function findSiblingEntryTarget(entries: OrgSiblingListEntry[], startIndex: number, step: 1 | -1) {
  for (let index = startIndex + step; index >= 0 && index < entries.length; index += step) {
    const entry = entries[index]
    if (entry?.kind === 'item' && entry.target) return entry.target
  }
  return null
}

function createCurrentDropPreviewCandidate(dragRect: OrgMeasuredRect, targets: OrgDragTarget[], listTargets: OrgSiblingListTarget[]): OrgDropCandidate | null {
  if (!previewTargetId.value || !previewDirection.value) return null
  const preview = document.querySelector<HTMLElement>('[data-org-drop-preview]')
  if (!preview) return null
  const target = targets.find((item) => item.item.id === previewTargetId.value)
  if (!target || !canMoveToTarget(target.item)) return null
  if (!canUseCurrentPreview(preview, target, dragRect)) return null
  const list = listTargets.find((item) => item.level === getOrgLevel(target.item) && item.parentId === target.item.pid)
  if (list && !isDragInSiblingListIntentBounds(list, dragRect)) return null

  const previewRect = toMeasuredRect(preview.getBoundingClientRect())
  if (!isDragNearCurrentPreview(dragRect, previewRect)) return null

  return {
    target,
    direction: previewDirection.value,
    intent: 'current-preview',
    score: getPointRectDistance(dragRect.centerX, dragRect.centerY, previewRect) * 0.2
  }
}

function createDropCandidate(target: OrgDragTarget, dragRect: OrgMeasuredRect): OrgDropCandidate | null {
  const rect = target.rect
  const minHeight = Math.max(1, Math.min(dragRect.height, rect.height))
  const deltaY = dragRect.centerY - rect.centerY
  const horizontalDistance = Math.abs(dragRect.centerX - rect.centerX)
  const overlapRatio = getVerticalOverlapRatio(dragRect, rect)
  const sameLevelLimit = minHeight * sameLevelCenterRatio
  const sameLevelHorizontalLimit = Math.max(rect.width * sameLevelHorizontalRatio, scaleCanvasDistance(300))
  const childLimit = minHeight * childCenterOffsetRatio
  const childHorizontalLimit = Math.max(rect.width * childHorizontalRatio, scaleCanvasDistance(260))

  if (target.visibleChildCount === 0 && canUseChildCandidate(target, dragRect) && isDragBelowTarget(dragRect, rect) && isDragAlignedForEmptyChild(dragRect, rect)) {
    return {
      target,
      direction: 'child',
      intent: 'empty-child',
      score: Math.abs(dragRect.top - rect.bottom) * 2 + horizontalDistance
    }
  }

  if (target.item.pid !== null && horizontalDistance <= sameLevelHorizontalLimit && (Math.abs(deltaY) <= sameLevelLimit || (overlapRatio >= sameLevelOverlapRatio && Math.abs(deltaY) <= childLimit))) {
    return {
      target,
      direction: getDropDirection(target, dragRect),
      intent: 'same-level',
      score: Math.abs(deltaY) * 4 + horizontalDistance
    }
  }

  if (canUseChildCandidate(target, dragRect) && deltaY >= childLimit && horizontalDistance <= childHorizontalLimit) {
    return {
      target,
      direction: 'child',
      intent: 'child',
      score: Math.abs(dragRect.centerY - (rect.bottom + minHeight * 0.35)) * 2 + horizontalDistance
    }
  }

  const distance = getPointRectDistance(dragRect.centerX, dragRect.centerY, rect)
  if (distance > fallbackDistanceLimit) return null
  return {
    target,
    direction: getFallbackDropDirection(target, dragRect),
    intent: 'fallback',
    score: distance
  }
}

function getFallbackDropDirection(target: OrgDragTarget, dragRect: OrgMeasuredRect): OrgAddDirection {
  if (target.item.pid === null) return 'child'
  if (target.visibleChildCount === 0 && canUseChildCandidate(target, dragRect) && isDragBelowTarget(dragRect, target.rect) && isDragAlignedForEmptyChild(dragRect, target.rect)) return 'child'
  const deltaY = dragRect.centerY - target.rect.centerY
  const horizontalDistance = Math.abs(dragRect.centerX - target.rect.centerX)
  if (canUseChildCandidate(target, dragRect) && deltaY >= target.rect.height * childCenterOffsetRatio && horizontalDistance <= Math.max(target.rect.width * childHorizontalRatio, scaleCanvasDistance(260))) return 'child'
  return getDropDirection(target, dragRect)
}

function isDragAlignedForEmptyChild(dragRect: OrgMeasuredRect, targetRect: OrgMeasuredRect) {
  const horizontalDistance = Math.abs(dragRect.centerX - targetRect.centerX)
  const horizontalLimit = Math.max(targetRect.width * emptyChildHorizontalRatio, dragRect.width * 0.55, scaleCanvasDistance(140))
  if (horizontalDistance <= horizontalLimit) return true
  return getHorizontalOverlapRatio(dragRect, targetRect) >= 0.42
}

function canUseCurrentPreview(preview: HTMLElement, target: OrgDragTarget, dragRect: OrgMeasuredRect) {
  if (previewDirection.value === 'child') return canUseChildCandidate(target, dragRect)
  const list = preview.closest<HTMLElement>('[data-org-node-list]')
  const level = list ? Number(list.dataset.orgLevel) : 0
  if (!Number.isFinite(level) || level <= nodeDragState.sourceLevel) return true
  return hasDownwardLevelIntent(dragRect)
}

function canUseChildCandidate(target: OrgDragTarget, dragRect: OrgMeasuredRect) {
  if (getOrgLevel(target.item) < nodeDragState.sourceLevel) return true
  return hasDownwardLevelIntent(dragRect)
}

function hasDownwardLevelIntent(dragRect: OrgMeasuredRect) {
  const sourceRect = nodeDragState.sourceRect
  if (!sourceRect) return true
  const verticalMove = dragRect.centerY - sourceRect.centerY
  const horizontalMove = Math.abs(dragRect.centerX - sourceRect.centerX)
  const verticalThreshold = Math.max(scaleCanvasDistance(28), Math.min(scaleCanvasDistance(72), dragRect.height * 0.68))
  if (verticalMove < verticalThreshold) return false
  return verticalMove >= Math.min(horizontalMove, verticalThreshold)
}

function isDragBelowTarget(dragRect: OrgMeasuredRect, targetRect: OrgMeasuredRect) {
  const tolerance = Math.min(scaleCanvasDistance(12), targetRect.height * 0.16)
  return dragRect.top >= targetRect.bottom - tolerance || dragRect.centerY >= targetRect.bottom + tolerance
}

function compareDropCandidates(a: OrgDropCandidate, b: OrgDropCandidate) {
  const intentRank = getDropIntentRank(a.intent) - getDropIntentRank(b.intent)
  return intentRank || a.score - b.score
}

function getDropIntentRank(intent: OrgDropIntent) {
  if (intent === 'current-preview') return 1
  if (intent === 'same-level-slot') return 1
  if (intent === 'empty-child') return 1
  if (intent === 'same-level') return 3
  if (intent === 'child') return 4
  return 5
}

function getDropStickyScore(intent: OrgDropIntent) {
  if (intent === 'current-preview') return scaleCanvasDistance(dropStickyScore)
  return scaleCanvasDistance(intent === 'same-level-slot' ? slotStickyScore : dropStickyScore)
}

function isDragInSiblingRow(dragRect: OrgMeasuredRect, rowRect: OrgMeasuredRect) {
  if (getVerticalOverlapRatio(dragRect, rowRect) >= 0.26) return true
  const verticalPadding = Math.min(scaleCanvasDistance(18), rowRect.height * 0.22)
  return dragRect.centerY >= rowRect.top - verticalPadding && dragRect.centerY <= rowRect.bottom + verticalPadding
}

function getVerticalOverlapRatio(a: OrgMeasuredRect, b: OrgMeasuredRect) {
  const overlap = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top)
  if (overlap <= 0) return 0
  return overlap / Math.max(1, Math.min(a.height, b.height))
}

function getHorizontalOverlapRatio(a: OrgMeasuredRect, b: OrgMeasuredRect) {
  const overlap = Math.min(a.right, b.right) - Math.max(a.left, b.left)
  if (overlap <= 0) return 0
  return overlap / Math.max(1, Math.min(a.width, b.width))
}

function isDragNearCurrentPreview(dragRect: OrgMeasuredRect, previewRect: OrgMeasuredRect) {
  if (getVerticalOverlapRatio(dragRect, previewRect) >= 0.32 && getHorizontalOverlapRatio(dragRect, previewRect) >= 0.32) return true
  const horizontalInset = Math.max(scaleCanvasDistance(36), dragRect.width * 0.3)
  const verticalInset = Math.max(scaleCanvasDistance(24), dragRect.height * 0.35)
  return dragRect.centerX >= previewRect.left - horizontalInset && dragRect.centerX <= previewRect.right + horizontalInset && dragRect.centerY >= previewRect.top - verticalInset && dragRect.centerY <= previewRect.bottom + verticalInset
}

function getPointRectDistance(x: number, y: number, rect: OrgMeasuredRect) {
  const dx = x < rect.left ? rect.left - x : x > rect.right ? x - rect.right : 0
  const dy = y < rect.top ? rect.top - y : y > rect.bottom ? y - rect.bottom : 0
  return Math.sqrt(dx * dx + dy * dy)
}

function getVerticalDistanceToRect(y: number, rect: OrgMeasuredRect) {
  if (y < rect.top) return rect.top - y
  if (y > rect.bottom) return y - rect.bottom
  return 0
}

function getCurrentDragRect(event: OrgPointer) {
  return getProjectedDraggedCardRect(event) || getGhostCardRect() || getFallbackDraggedCardRect(event)
}

function getProjectedDraggedCardRect(event: OrgPointer) {
  if (!dragGhost.item) return null
  const width = dragCardSize.width * dragGhost.scale
  const height = dragCardSize.height * dragGhost.scale
  return createMeasuredRect(event.clientX - dragGhost.grabX, event.clientY - dragGhost.grabY, width, height)
}

function getGhostCardRect() {
  const root = dragGhostRef.value
  if (!root || !dragGhost.item) return null
  const card = root.querySelector<HTMLElement>(`[data-org-ghost-card][data-org-ghost-id="${dragGhost.item.id}"]`)
  return card ? toMeasuredRect(card.getBoundingClientRect()) : null
}

function getFallbackDraggedCardRect(event: OrgPointer) {
  const source = getSourceDragTarget(nodeDragState.sourceId)
  if (source) return offsetMeasuredRect(source.rect, event.clientX - nodeDragState.startX, event.clientY - nodeDragState.startY)
  const width = dragCardSize.width * dragGhost.scale
  const height = dragCardSize.height * dragGhost.scale
  return createMeasuredRect(event.clientX - dragGhost.grabX, event.clientY - dragGhost.grabY, width, height)
}

function getOriginalDropPreview(sourceId: string, targets: OrgDragTarget[]): OrgDropPreview | null {
  const source = findOrgItem(org.value.items, sourceId)
  if (!source) return null
  const parent = findOrgParent(org.value.items, source.id)
  const siblings = parent ? parent.children : org.value.items
  const sourceIndex = siblings.findIndex((item) => item.id === source.id)
  const previous = sourceIndex > 0 ? siblings[sourceIndex - 1] : null
  const next = sourceIndex >= 0 ? siblings[sourceIndex + 1] : null
  const targetById = (id: string | undefined) => targets.find((target) => target.item.id === id && canMoveToTarget(target.item)) || null
  const previousTarget = targetById(previous?.id)
  if (previousTarget) return { target: previousTarget, direction: 'next' }
  const nextTarget = targetById(next?.id)
  if (nextTarget) return { target: nextTarget, direction: 'previous' }
  const parentTarget = targetById(parent?.id)
  return parentTarget ? { target: parentTarget, direction: 'child' } : null
}

function collectDragTargets(sourceId: string) {
  const source = findOrgItem(org.value.items, sourceId)
  const cards = Array.from(document.querySelectorAll<HTMLElement>('[data-org-node-card]'))
  return cards.flatMap((card) => {
    const target = toDragTarget(card)
    if (!target || target.item.id === sourceId || (source && containsOrgItem(source, target.item.id))) return []
    return [target]
  })
}

/** 按实际分支边界建立同级插槽快照，预览占位也参与行几何计算。 */
function collectSiblingListTargets(targets: OrgDragTarget[]) {
  const targetsById = new Map(targets.map((target) => [target.item.id, target]))
  const lists = Array.from(document.querySelectorAll<HTMLElement>('[data-org-node-list]'))
  return lists.flatMap((list) => {
    const level = Number(list.dataset.orgLevel)
    const parentId = list.dataset.orgParentId ? list.dataset.orgParentId : null
    if (!Number.isFinite(level)) return []
    const entries = Array.from(list.children).flatMap<OrgSiblingListEntry>((entry) => {
      if (!isHTMLElement(entry)) return []
      if (entry.dataset.orgEntryKind === 'preview') {
        const preview = entry.querySelector<HTMLElement>('[data-org-drop-preview]')
        const previewCard = entry.querySelector<HTMLElement>('[data-org-drop-preview-card]')
        const rect = toMeasuredRect(entry.getBoundingClientRect())
        const rowRect = previewCard ? toMeasuredRect(previewCard.getBoundingClientRect()) : preview ? toMeasuredRect(preview.getBoundingClientRect()) : rect
        return [{ kind: 'preview' as const, target: null, rect, rowRect }]
      }
      if (entry.dataset.orgEntryKind !== 'item') return []
      const id = entry.dataset.orgItemId || ''
      const target = targetsById.get(id)
      return target ? [{ kind: 'item' as const, target, rect: toMeasuredRect(entry.getBoundingClientRect()), rowRect: target.rect }] : []
    })
    const items = entries.flatMap((entry) => (entry.kind === 'item' && entry.target ? [{ target: entry.target, rect: entry.rect }] : []))
    const rowRect = mergeMeasuredRects(entries.map((entry) => entry.rowRect))
    if (!rowRect || items.length === 0) return []
    return [
      {
        level,
        parentId,
        parent: parentId === null ? null : targetsById.get(parentId) || null,
        rowRect,
        entries,
        items
      }
    ]
  })
}

function isHTMLElement(value: Element): value is HTMLElement {
  return 'dataset' in value && typeof value.getBoundingClientRect === 'function'
}

function mergeMeasuredRects(rects: OrgMeasuredRect[]) {
  if (!rects.length) return null
  const left = Math.min(...rects.map((rect) => rect.left))
  const right = Math.max(...rects.map((rect) => rect.right))
  const top = Math.min(...rects.map((rect) => rect.top))
  const bottom = Math.max(...rects.map((rect) => rect.bottom))
  return createMeasuredRect(left, top, right - left, bottom - top)
}

function toDragTarget(card: HTMLElement): OrgDragTarget | null {
  const id = card.dataset.orgId || ''
  if (!id) return null
  const item = findOrgItem(org.value.items, id)
  if (!item) return null
  const rect = toMeasuredRect(card.getBoundingClientRect())
  const branch = card.closest<HTMLElement>('[data-org-node-branch]')
  const branchBox = branch ? measureBranchBox(branch, card) : null
  return {
    item,
    rect,
    cardWidth: card.offsetWidth || orgCardWidth,
    cardHeight: card.offsetHeight || orgCardHeight,
    branchWidth: branchBox?.width || card.offsetWidth || orgCardWidth,
    branchHeight: branchBox?.height || card.offsetHeight || orgCardHeight,
    cardLeft: branchBox?.cardLeft || 0,
    cardTop: branchBox?.cardTop || 0,
    visibleChildCount: getVisibleChildCount(item)
  }
}

function measureBranchBox(branch: HTMLElement, card: HTMLElement) {
  const branchRect = branch.getBoundingClientRect()
  const cardRect = card.getBoundingClientRect()
  const scale = getCanvasScale()
  if (!scale) return null
  return {
    width: branchRect.width / scale,
    height: branchRect.height / scale,
    cardLeft: (cardRect.left - branchRect.left) / scale,
    cardTop: (cardRect.top - branchRect.top) / scale
  }
}

function getVisibleChildCount(item: OrgItem) {
  const sourceId = nodeDragState.sourceId
  return item.children.filter((child) => child.id !== sourceId).length
}

function toMeasuredRect(rect: DOMRect | OrgMeasuredRect): OrgMeasuredRect {
  return createMeasuredRect(rect.left, rect.top, rect.width, rect.height)
}

function offsetMeasuredRect(rect: OrgMeasuredRect, x: number, y: number) {
  return createMeasuredRect(rect.left + x, rect.top + y, rect.width, rect.height)
}

function toCanvasLocalRect(rect: OrgMeasuredRect) {
  const nodes = nodesRef.value
  const scale = getCanvasScale()
  if (!nodes || !scale) return null
  const nodesRect = nodes.getBoundingClientRect()
  return createMeasuredRect((rect.left - nodesRect.left) / scale, (rect.top - nodesRect.top) / scale, rect.width / scale, rect.height / scale)
}

function fromCanvasLocalRect(rect: OrgMeasuredRect) {
  const nodes = nodesRef.value
  const scale = getCanvasScale()
  if (!nodes) return null
  const nodesRect = nodes.getBoundingClientRect()
  return createMeasuredRect(nodesRect.left + rect.left * scale, nodesRect.top + rect.top * scale, rect.width * scale, rect.height * scale)
}

function createMeasuredRect(left: number, top: number, width: number, height: number): OrgMeasuredRect {
  return {
    left,
    top,
    right: left + width,
    bottom: top + height,
    width,
    height,
    centerX: left + width / 2,
    centerY: top + height / 2
  }
}

function getCanvasScale() {
  return zoomPercent.value / 100
}

function scaleCanvasDistance(value: number) {
  return value * getCanvasScale()
}

function suppressNextClick() {
  document.addEventListener('click', suppressClick, { capture: true, once: true })
  window.setTimeout(() => {
    document.removeEventListener('click', suppressClick, true)
  }, 120)
}

function suppressClick(event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()
}

function openOrgDrawer(item: OrgItem) {
  if (drawerVisible.value && selectedNodeId.value === item.id) return
  selectedNodeId.value = item.id
  applyDrawerForm(item)
  drawerVisible.value = true
  void loadDrawerEmployees(item.id)
}

function closeDrawer() {
  drawerVisible.value = false
}

function syncSelectedDrawer() {
  if (!selectedNodeId.value) return
  const item = findOrgItem(org.value.items, selectedNodeId.value)
  if (!item) {
    selectedNodeId.value = ''
    drawerVisible.value = false
    drawerEmployees.value = []
    return
  }
  if (!drawerVisible.value) return
  applyDrawerForm(item)
  void loadDrawerEmployees(item.id)
}

function applyDrawerForm(item: OrgItem) {
  Object.assign(form, toOrgForm(item))
}

/** 未保存的新节点没有员工；请求序号防止快速切换节点时串写抽屉。 */
async function loadDrawerEmployees(organizationId: string) {
  const requestId = drawerEmployeeRequestId + 1
  drawerEmployeeRequestId = requestId
  drawerEmployeesLoading.value = true
  drawerEmployees.value = []
  try {
    const employees = persistedIds.value.has(organizationId) ? await organizationApi.employees(organizationId) : []
    if (!isCurrentDrawerEmployeeRequest(requestId, organizationId)) return
    drawerEmployees.value = employees
    syncLeaderDraftWithEmployees(employees)
  } catch {
    if (isCurrentDrawerEmployeeRequest(requestId, organizationId)) {
      drawerEmployees.value = []
      message.error('员工列表加载失败')
    }
  } finally {
    if (isCurrentDrawerEmployeeRequest(requestId, organizationId)) drawerEmployeesLoading.value = false
  }
}

function isCurrentDrawerEmployeeRequest(requestId: number, organizationId: string) {
  return drawerEmployeeRequestId === requestId && selectedNodeId.value === organizationId
}

/** 负责人仅能从当前节点直接员工中选择，后代员工仅展示。 */
function syncLeaderDraftWithEmployees(employees: OrganizationEmployeeRecord[]) {
  if (form.leaderId !== null) {
    const selectedLeader = employees.find((employee) => employee.id === form.leaderId && employee.leaderSelectable)
    if (selectedLeader) {
      form.leaderName = selectedLeader.displayName
      return
    }
    form.leaderId = null
    return
  }
  const leaderName = form.leaderName.trim()
  if (!leaderName) return
  const matchedEmployees = employees.filter((employee) => employee.leaderSelectable && employee.displayName === leaderName)
  if (matchedEmployees.length !== 1) return
  const [matchedEmployee] = matchedEmployees
  if (!matchedEmployee) return
  form.leaderId = matchedEmployee.id
  form.leaderName = matchedEmployee.displayName
}

function setLeader(employee: OrganizationEmployeeRecord) {
  if (!employee.leaderSelectable) return
  form.leaderId = employee.id
  form.leaderName = employee.displayName
}

function clearLeader() {
  form.leaderId = null
  form.leaderName = ''
}

/** 确定只更新本地草稿；关闭和取消均不提前保存服务端。 */
function submitEditor() {
  if (!ensureOrganizationEditPermission()) return
  if (!form.name.trim()) {
    message.warning('请填写部门名称')
    return
  }
  if (!updateOrgItem(org.value, form)) return
  syncOrg()
  drawerVisible.value = false
}

function zoom(mode: 'out' | 'in' | 'reset') {
  const previousZoom = zoomPercent.value
  const nextZoom = mode === 'out' ? Math.max(50, previousZoom - 25) : mode === 'in' ? Math.min(300, previousZoom + 25) : 100
  if (setZoomPercent(nextZoom, previousZoom)) refreshDraggingAfterZoom()
}

function setZoomPercent(nextZoom: number, previousZoom: number = zoomPercent.value) {
  if (nextZoom === previousZoom) return false
  zoomPercent.value = nextZoom
  syncDraggingScaleAfterZoom(nextZoom / previousZoom)
  return true
}

/** 拖动时缩放同步抓取偏移，避免节点在指针下跳动。 */
function syncDraggingScaleAfterZoom(scaleRatio: number) {
  if (!nodeDragState.sourceId || !nodeDragState.hasMoved || !dragGhost.item || scaleRatio <= 0) return
  dragGhost.scale *= scaleRatio
  dragGhost.grabX *= scaleRatio
  dragGhost.grabY *= scaleRatio
  dragGhost.offsetX *= scaleRatio
  dragGhost.offsetY *= scaleRatio
  dragGhost.x = nodeDragState.pointerX - dragGhost.offsetX
  dragGhost.y = nodeDragState.pointerY - dragGhost.offsetY
}

function refreshDraggingAfterZoom() {
  if (!nodeDragState.sourceId || !nodeDragState.hasMoved) return
  const pointer = getNodeDragPointer()
  void nextTick(() => {
    alignDragGhostToPointer(pointer)
    nodeDragState.sourceRect = getSourceDragRectAfterZoom()
    updateDropPreview(pointer)
  })
}

function getSourceDragRectAfterZoom() {
  if (!nodeDragState.sourceLocalRect) return null
  return fromCanvasLocalRect(nodeDragState.sourceLocalRect)
}

function getNodeDragPointer(): OrgPointer {
  return {
    clientX: nodeDragState.pointerX || nodeDragState.startX,
    clientY: nodeDragState.pointerY || nodeDragState.startY
  }
}

/** 以鼠标所指的树内容为缩放锚点，并同步拖动中的幽灵坐标。 */
function zoomByWheel(event: WheelEvent) {
  if (isWheelControlTarget(event.target)) return
  const viewport = viewportRef.value
  const nodes = nodesRef.value
  if (!viewport) return
  event.preventDefault()

  const nextZoom = clampZoom(zoomPercent.value + (event.deltaY > 0 ? -10 : 10))
  if (nextZoom === zoomPercent.value) return

  const previousNodeRect = nodes?.getBoundingClientRect()
  const previousScale = zoomPercent.value / 100
  const anchorX = previousNodeRect ? (event.clientX - previousNodeRect.left) / previousScale : 0
  const anchorY = previousNodeRect ? (event.clientY - previousNodeRect.top) / previousScale : 0

  setZoomPercent(nextZoom, zoomPercent.value)
  void nextTick(() => {
    resizeCanvas()
    const nextNodeRect = nodes?.getBoundingClientRect()
    if (!previousNodeRect || !nextNodeRect) return
    const nextScale = nextZoom / 100
    const nextAnchorX = nextNodeRect.left + anchorX * nextScale
    const nextAnchorY = nextNodeRect.top + anchorY * nextScale
    viewport.scrollLeft += nextAnchorX - event.clientX
    viewport.scrollTop += nextAnchorY - event.clientY
    refreshDraggingAfterZoom()
  })
}

function clampZoom(value: number) {
  return Math.min(300, Math.max(50, value))
}

function centerCanvas() {
  const viewport = viewportRef.value
  const canvas = canvasRef.value
  const nodes = nodesRef.value
  if (!viewport || !canvas || !nodes) return

  resizeCanvas()
  viewport.scrollLeft = Math.max(0, (canvas.offsetWidth - viewport.clientWidth) / 2)
  viewport.scrollTop = Math.max(0, (canvas.offsetHeight - viewport.clientHeight) / 2)
}

/** 扩展可平移画布边界，为大树和放大后的分支留出滚动空间。 */
function resizeCanvas() {
  const viewport = viewportRef.value
  const canvas = canvasRef.value
  const nodes = nodesRef.value
  if (!viewport || !canvas || !nodes) return

  const scaledWidth = nodes.offsetWidth * (zoomPercent.value / 100)
  const scaledHeight = nodes.offsetHeight * (zoomPercent.value / 100)
  canvas.style.width = `${Math.max(scaledWidth, viewport.clientWidth) * 1.6}px`
  canvas.style.height = `${Math.max(scaledHeight, viewport.clientHeight) * 1.6}px`
}

function startCanvasDrag(event: MouseEvent) {
  if (isInteractiveTarget(event.target)) return
  const viewport = viewportRef.value
  if (!viewport) return
  event.preventDefault()
  canvasDragState.dragging = true
  canvasDragState.startX = event.clientX
  canvasDragState.startY = event.clientY
  canvasDragState.scrollLeft = viewport.scrollLeft
  canvasDragState.scrollTop = viewport.scrollTop
  canvasDragState.hasMoved = false
  document.addEventListener('mousemove', dragCanvas)
  document.addEventListener('mouseup', stopCanvasDrag, true)
}

function dragCanvas(event: MouseEvent) {
  if (!canvasDragState.dragging) return
  const moved = Math.abs(event.clientX - canvasDragState.startX) + Math.abs(event.clientY - canvasDragState.startY)
  canvasDragState.hasMoved ||= moved > 4
  const viewport = viewportRef.value
  if (!viewport) return
  viewport.scrollLeft = canvasDragState.scrollLeft + canvasDragState.startX - event.clientX
  viewport.scrollTop = canvasDragState.scrollTop + canvasDragState.startY - event.clientY
}

function stopCanvasDrag() {
  if (canvasDragState.hasMoved) suppressNextClick()
  canvasDragState.dragging = false
  canvasDragState.hasMoved = false
  document.removeEventListener('mousemove', dragCanvas)
  document.removeEventListener('mouseup', stopCanvasDrag, true)
}

function isInteractiveTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false
  return Boolean(target.closest('button,input,textarea,select,[role="button"],[data-org-interactive]'))
}

function isWheelControlTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false
  return Boolean(target.closest('input,textarea,select,[data-org-no-zoom]'))
}

function syncOrg() {
  org.value = cloneOrgData(org.value)
  dirty.value = true
  void nextTick(resizeCanvas)
}

function ensureOrganizationEditPermission() {
  if (canEditOrganization.value) return true
  message.warning('无权编辑组织架构')
  return false
}
</script>

<router lang="json">
{
  "title": "system.organizations.title",
  "permission": "org:view"
}
</router>
