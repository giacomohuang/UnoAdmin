<script setup lang="ts">
/**
 * 权限管理：MillerColumns 逐级浏览 + 右侧详情面板 + 顶部搜索。
 *
 * 平台运营管理员可以新增、编辑、删除权限节点；其余账号只读。
 * 权限码是菜单、路由与按钮的共同依据，修改后需要重新登录或刷新会话才会生效。
 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Button } from '@mcistudio/unoui-vue/button'
import { Drawer } from '@mcistudio/unoui-vue/drawer'
import { Form, FormItem, type FormRules } from '@mcistudio/unoui-vue/form'
import { Autocomplete, Input, type AutocompleteDataSourceCallback, type AutocompleteSuggestion } from '@mcistudio/unoui-vue/input'
import { message } from '@mcistudio/unoui-vue/message'
import { MillerColumns, type MillerColumnsId } from '@mcistudio/unoui-vue/millercolumns'
import { Popconfirm } from '@mcistudio/unoui-vue/popconfirm'
import { Select } from '@mcistudio/unoui-vue/select'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Tag } from '@mcistudio/unoui-vue/tag'
import { Icon } from '@iconify/vue'
import type { Permission, PermissionId, SavePermission } from '@unoadmin/core'
import { computed, onMounted, reactive, ref } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import {
  descendantIds,
  flattenPermissions,
  permissionIcon,
  permissionLabel,
  permissionPath,
  type PermissionNode
} from '../../components/tree'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { permissions: permissionService } = useAdminBaseServices()

const nodes = ref<Permission[]>([])
const loading = ref(false)
const ready = ref(false)
const error = ref('')
const keyword = ref('')
const selection = ref<MillerColumnsId[]>([])

const drawerOpen = ref(false)
const saving = ref(false)
const deleting = ref(false)
const sorting = ref(false)
const formError = ref('')
const editing = ref<PermissionNode | null>(null)
const formRef = ref<InstanceType<typeof Form>>()
const form = reactive<SavePermission>({ name: '', code: '', type: 'page', scope: 'tenant', parentId: '', sortOrder: 0 })

const flat = computed(() => flattenPermissions(nodes.value))
const selected = computed<PermissionNode | undefined>(() =>
  flat.value.find((node) => node.id === String(selection.value.at(-1) ?? ''))
)
const canCreate = computed(() => session.can('permission:create'))
const canUpdate = computed(() => session.can('permission:update'))
const canDelete = computed(() => session.can('permission:delete'))
const pageCount = computed(() => flat.value.filter((node) => node.type === 'page').length)
const actionCount = computed(() => flat.value.filter((node) => node.type === 'action').length)

const typeOptions = [
  { label: '菜单', value: 'menu' },
  { label: '页面', value: 'page' },
  { label: '操作', value: 'action' }
]
const scopeOptions = [
  { label: '平台', value: 'platform' },
  { label: '租户', value: 'tenant' },
  { label: '主体', value: 'entity' }
]

const typeLabel: Record<Permission['type'], string> = { menu: '菜单', page: '页面', action: '操作' }
const scopeLabel: Record<Permission['scope'], { label: string; color: 'purple' | 'blue' | 'cyan' }> = {
  platform: { label: '平台', color: 'purple' },
  tenant: { label: '租户', color: 'blue' },
  entity: { label: '主体', color: 'cyan' }
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入权限名称' }],
  code: [
    { required: true, message: '请输入权限码' },
    { pattern: /^[a-z0-9:._-]{2,64}$/, message: '仅支持小写字母、数字与 : . _ -' }
  ]
}

/** 上级权限选项：编辑时排除自身及其后代，避免形成环。 */
const parentOptions = computed(() => {
  const blocked = editing.value ? new Set([editing.value.id, ...descendantIds(editing.value.id, flat.value)]) : new Set<string>()
  return flat.value
    .filter((node) => !blocked.has(node.id))
    .map((node) => ({ label: `${'— '.repeat(permissionPath(node.id, flat.value).length - 1)}${node.name}`, value: node.id }))
})

/** 权限类型标签，兼容旧数据缺失 type 的情况。 */
function typeStyle(type: Permission['type']) {
  return { label: typeLabel[type] ?? '页面', color: type === 'action' ? ('yellow' as const) : ('brand' as const) }
}

/** 生效层级标签。 */
function scopeStyle(scope: Permission['scope']) {
  return scopeLabel[scope] ?? { label: '租户', color: 'blue' as const }
}

/** 加载权限树并尽量保留当前选中路径。 */
async function load(selectId?: string) {
  loading.value = true
  error.value = ''
  try {
    nodes.value = await permissionService.list()
    ready.value = true
    const target = selectId ?? String(selection.value.at(-1) ?? '')
    selection.value = permissionPath(target, flat.value)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '权限目录加载失败'
  } finally {
    loading.value = false
  }
}

/** 按名称或权限码搜索，返回带层级路径的候选。 */
function search(query: string, done: AutocompleteDataSourceCallback) {
  const value = query.trim().toLowerCase()
  done(
    value
      ? flat.value
          .filter((node) => `${node.name} ${node.code}`.toLowerCase().includes(value))
          .map((node) => ({
            value: node.name,
            id: node.id,
            label: permissionLabel(node.id, flat.value),
            code: node.code
          }))
      : []
  )
}

/** 选中搜索结果时展开对应层级列。 */
function locate(item: AutocompleteSuggestion) {
  selection.value = permissionPath(String(item.id), flat.value)
}

/** 同级下一个排序值，追加到末尾。 */
function nextSort(parentId: string) {
  const siblings = flat.value.filter((node) => node.parentId === parentId).map((node) => node.sort)
  return siblings.length ? Math.max(...siblings) + 1 : 0
}

/** 打开新建抽屉；parentId 为空表示根级权限。 */
function openCreate(parentId = '') {
  editing.value = null
  Object.assign(form, {
    name: '',
    code: '',
    type: parentId ? 'action' : 'page',
    scope: 'tenant',
    parentId,
    sortOrder: nextSort(parentId)
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 打开编辑抽屉。 */
function openEdit(node: PermissionNode) {
  editing.value = node
  Object.assign(form, {
    name: node.name,
    code: node.code,
    type: node.type,
    scope: nodes.value.find((item) => item.id === node.id)?.scope ?? 'tenant',
    parentId: node.parentId ?? '',
    sortOrder: node.sort
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 校验并保存权限节点。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  formError.value = ''
  saving.value = true
  try {
    const payload: SavePermission = { ...form, parentId: form.parentId || '' }
    const saved = editing.value
      ? await permissionService.update(editing.value.id as PermissionId, payload)
      : await permissionService.create(payload)
    message.success('权限已保存')
    drawerOpen.value = false
    await load(saved.id)
  } catch (cause) {
    formError.value = cause instanceof Error ? cause.message : '保存失败'
  } finally {
    saving.value = false
  }
}

/** 删除叶子权限，成功后回到父节点。 */
async function remove(node: PermissionNode) {
  if (deleting.value) return
  deleting.value = true
  try {
    await permissionService.remove(node.id as PermissionId)
    message.success('权限已删除')
    await load(node.parentId ?? '')
  } catch (cause) {
    message.error(cause instanceof Error ? cause.message : '删除失败')
  } finally {
    deleting.value = false
  }
}

/**
 * MillerColumns 拖拽结束后回调，ids 为同一列内的新顺序。
 *
 * 服务端按数组下标重写 sortOrder 并返回最新权限树，前端直接用它替换本地数据，
 * 避免 MillerColumns 就地修改的 sort 与嵌套树不一致。
 */
async function handleReorder(ids: MillerColumnsId[]) {
  if (!canUpdate.value || sorting.value || !ids.length) return
  const first = flat.value.find((node) => node.id === String(ids[0]))
  if (!first) return
  sorting.value = true
  try {
    nodes.value = await permissionService.reorder({ parentId: first.parentId ?? '', ids: ids.map(String) })
    selection.value = permissionPath(String(selection.value.at(-1) ?? ''), flat.value)
    message.success('排序已保存')
  } catch (cause) {
    message.error(cause instanceof Error ? cause.message : '排序保存失败')
    await load()
  } finally {
    sorting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>权限管理</template>
      <template #status>{{ flat.length }} 项权限 · {{ pageCount }} 页面 · {{ actionCount }} 操作</template>
      <template #description>权限由代码声明，按列逐级查看；拖拽行可调整同级顺序，角色负责组合授权。</template>
      <template #actions>
        <div class="flex shrink-0 items-center gap-2">
          <div class="w-64">
            <Autocomplete
              v-model="keyword"
              :data-source="search"
              :trigger-on-focus="false"
              clearable
              prefix-icon="i-lucide:search"
              placeholder="搜索权限名称或编码"
              aria-label="搜索权限名称或编码"
              @select="locate"
            >
              <template #suggestion="{ item }">
                <span class="block min-w-0 px-3 py-2">
                  <span class="block break-words">{{ (item as { label: string }).label }}</span>
                  <code class="text-xs text-tertiary">{{ (item as { code: string }).code }}</code>
                </span>
              </template>
            </Autocomplete>
          </div>
          <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load()" />
          <Button v-if="canCreate" icon="i-lucide:plus" @click="openCreate()">新建权限</Button>
        </div>
      </template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <Skeleton v-if="!ready && loading" variant="columns" height="32rem" show-info-panel />
    <MillerColumns
      v-else
      v-model="selection"
      :data-source="flat"
      id-key="id"
      parent-id-key="parentId"
      order-key="sort"
      height="32rem"
      min-height="24rem"
      width="100%"
      column-width="15rem"
      info-panel-width="22rem"
      show-info-panel
      radius="md"
      bordered
      :sortable="canUpdate && !saving && !deleting && !sorting"
      :empty-text="'该权限没有下级'"
      :no-data-text="'暂无权限目录'"
      aria-label="权限目录"
      @reorder="handleReorder"
    >
      <template #col-title="{ colIndex, parentId }">
        <span>第 {{ colIndex + 1 }} 级</span>
        <Button
          v-if="canCreate"
          size="icon"
          variant="mono"
          icon="i-lucide:plus"
          title="新建子权限"
          aria-label="新建子权限"
          @click.stop="openCreate(parentId === null ? '' : String(parentId))"
        />
      </template>

      <template #item-left="{ item, active }">
        <Icon
          :icon="permissionIcon(item)"
          class="h-4 w-4 shrink-0"
          :class="active ? 'text-white' : item.type === 'action' ? 'text-amber-600' : 'text-emerald-600'"
        />
        <span class="min-w-0 truncate" :title="item.name">{{ item.name }}</span>
      </template>

      <template #item-right="{ item, active }">
        <span class="shrink-0 text-[11px]" :class="active ? 'text-white/80' : 'text-tertiary'">
          {{ scopeStyle(item.scope).label }}
        </span>
      </template>

      <template #info-panel="{ item }">
        <div class="h-full min-w-0 overflow-y-auto p-5">
          <template v-if="item">
            <div class="flex items-start gap-3">
              <Icon :icon="permissionIcon(item)" class="mt-0.5 h-6 w-6 shrink-0 text-emerald-600" />
              <h2 class="m-0 min-w-0 break-words text-lg font-600">{{ item.name }}</h2>
            </div>

            <div class="my-4 flex flex-wrap gap-2">
              <Tag :color="typeStyle(item.type).color">{{ typeStyle(item.type).label }}</Tag>
              <Tag :color="scopeStyle(item.scope).color">{{ scopeStyle(item.scope).label }}级</Tag>
              <Tag v-if="item.childCount" color="gray">{{ item.childCount }} 个下级</Tag>
            </div>

            <dl class="m-0 space-y-4 text-sm">
              <div>
                <dt class="text-tertiary">权限码</dt>
                <dd class="mb-0 mt-1 break-all font-mono text-xs">{{ item.code }}</dd>
              </div>
              <div>
                <dt class="text-tertiary">上级权限</dt>
                <dd class="mb-0 mt-1 break-words">{{ permissionLabel(item.parentId, flat) || '根节点' }}</dd>
              </div>
              <div>
                <dt class="text-tertiary">节点 ID</dt>
                <dd class="mb-0 mt-1 break-all font-mono text-xs">{{ item.id }}</dd>
              </div>
            </dl>

            <div v-if="canUpdate || canDelete" class="mt-6 flex flex-wrap gap-2 border-t border-light pt-4">
              <Button v-if="canUpdate" variant="outline" icon="i-lucide:pencil" @click="openEdit(item)">编辑</Button>
              <Popconfirm
                v-if="canDelete"
                title="删除该权限？"
                description="仍被角色引用或存在子权限时需要先处理依赖。"
                ok-text="删除"
                cancel-text="取消"
                ok-type="danger"
                @confirm="remove(item)"
              >
                <Button color="red" variant="outline" icon="i-lucide:trash-2" :loading="deleting">删除</Button>
              </Popconfirm>
            </div>
          </template>

          <div v-else class="grid min-h-40 place-content-center gap-2 text-center">
            <span class="i-lucide:mouse-pointer-click mx-auto h-6 w-6 text-quaternary" aria-hidden="true" />
            <p class="m-0 text-sm text-tertiary">从左侧选择权限查看详情</p>
          </div>
        </div>
      </template>
    </MillerColumns>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑权限' : '新建权限'" size="min(32rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <div class="grid gap-x-4 sm:grid-cols-2">
            <FormItem label="权限名称" prop="name" required class="sm:col-span-2">
              <Input v-model="form.name" :maxlength="40" placeholder="例如：新建账户" />
            </FormItem>
            <FormItem label="权限码" prop="code" required class="sm:col-span-2">
              <Input v-model="form.code" :maxlength="64" placeholder="例如：account:create" />
            </FormItem>
            <FormItem label="类型">
              <Select v-model="form.type" :options="typeOptions" />
            </FormItem>
            <FormItem label="生效层级">
              <Select v-model="form.scope" :options="scopeOptions" />
            </FormItem>
            <FormItem label="上级权限" class="sm:col-span-2">
              <Select v-model="form.parentId" :options="parentOptions" clearable filterable placeholder="留空表示根节点" />
            </FormItem>
            <FormItem label="排序" class="sm:col-span-2">
              <Input v-model.number="form.sortOrder" type="number" min="0" />
            </FormItem>
          </div>
        </Form>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <Button variant="outline" :disabled="saving" @click="drawerOpen = false">取消</Button>
          <Button icon="i-lucide:check" :loading="saving" @click="save">保存</Button>
        </div>
      </template>
    </Drawer>
  </div>
</template>

<router lang="json">
{
  "title": "权限管理",
  "permission": "permission:view"
}
</router>
