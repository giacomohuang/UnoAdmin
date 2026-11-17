<script setup lang="ts">
/**
 * 组织架构：编辑经营主体内部的部门、门店与班组层级。
 *
 * 组织是账户授权的粒度，也是「本人组织」数据范围的锚点；
 * 页面只允许在当前可访问的经营主体内维护。
 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Button } from '@mcistudio/unoui-vue/button'
import { Drawer } from '@mcistudio/unoui-vue/drawer'
import { Form, FormItem, type FormRules } from '@mcistudio/unoui-vue/form'
import { Input } from '@mcistudio/unoui-vue/input'
import { message } from '@mcistudio/unoui-vue/message'
import { Select } from '@mcistudio/unoui-vue/select'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Switch } from '@mcistudio/unoui-vue/switch'
import { Tag } from '@mcistudio/unoui-vue/tag'
import type { OrgId, OrgUnit, SaveOrg } from '@unoadmin/core'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import OrgTree from '../../components/OrgTree.vue'
import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { orgs: orgService } = useAdminBaseServices()

const entityId = ref(session.state.currentLegalEntity?.id ?? '')
const tree = ref<OrgUnit[]>([])
const selected = ref<OrgUnit | null>(null)
const loading = ref(false)
const ready = ref(false)
const error = ref('')

const drawerOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const editing = ref<OrgUnit | null>(null)
const formRef = ref<InstanceType<typeof Form>>()
const form = reactive<SaveOrg>({ legalEntityId: '', parentId: '', name: '', type: 'department', sort: 0, status: 'active' })

const canEdit = computed(() => session.can('org:edit'))
const entityOptions = computed(() => session.state.legalEntities.map((entity) => ({ label: entity.name, value: entity.id })))
const currentEntity = computed(() => session.state.legalEntities.find((entity) => entity.id === entityId.value))
const isStandalone = computed(() => session.state.currentTenant?.mode === 'standalone')

const rules: FormRules = {
  name: [{ required: true, message: '请输入组织名称' }]
}

/** 扁平化组织树，供上级选择与统计使用。 */
function flatten(nodes: OrgUnit[], depth = 0): Array<OrgUnit & { depth: number }> {
  return nodes.flatMap((node) => [{ ...node, depth }, ...flatten(node.children ?? [], depth + 1)])
}

const flat = computed(() => flatten(tree.value))
const nodeCount = computed(() => flat.value.length)
const parentOptions = computed(() =>
  flat.value
    .filter((node) => node.id !== editing.value?.id)
    .map((node) => ({ label: `${'— '.repeat(node.depth)}${node.name}`, value: node.id }))
)
const typeOptions = [
  { label: '公司', value: 'company' },
  { label: '部门', value: 'department' },
  { label: '班组', value: 'team' },
  { label: '门店', value: 'store' },
  { label: '项目组', value: 'project' }
]

/** 加载当前主体的组织树，并尽量保留原选中节点。 */
async function load() {
  if (!entityId.value) {
    tree.value = []
    ready.value = true
    return
  }
  loading.value = true
  error.value = ''
  try {
    tree.value = await orgService.tree(entityId.value)
    ready.value = true
    const previous = selected.value?.id
    selected.value = previous ? flat.value.find((node) => node.id === previous) ?? null : flat.value[0] ?? null
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '组织架构加载失败'
    tree.value = []
  } finally {
    loading.value = false
  }
}

/** 打开新建抽屉；父级默认为当前选中节点。 */
function openCreate(parent?: OrgUnit | null) {
  editing.value = null
  Object.assign(form, {
    legalEntityId: entityId.value,
    parentId: parent?.id ?? '',
    name: '',
    type: parent ? 'department' : 'company',
    sort: 0,
    status: 'active'
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 打开编辑抽屉。 */
function openEdit(node: OrgUnit) {
  editing.value = node
  Object.assign(form, {
    legalEntityId: node.legalEntityId,
    parentId: node.parentId,
    name: node.name,
    type: node.type,
    sort: node.sort,
    status: node.status
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 校验并保存组织节点。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  formError.value = ''
  saving.value = true
  try {
    const payload: SaveOrg = { ...form, legalEntityId: entityId.value }
    if (editing.value) await orgService.update(editing.value.id as OrgId, payload)
    else {
      const created = await orgService.create(payload)
      selected.value = created
    }
    message.success('组织节点已保存')
    drawerOpen.value = false
    await load()
  } catch (cause) {
    formError.value = cause instanceof Error ? cause.message : '保存失败'
  } finally {
    saving.value = false
  }
}

/** 切换主体时重新加载组织树。 */
watch(entityId, () => {
  selected.value = null
  void load()
})

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>组织架构</template>
      <template #status>{{ nodeCount }} 个组织节点</template>
      <template #description>{{ currentEntity?.name || '未选择主体' }} · 组织是账户授权与「本人组织」数据范围的锚点</template>
      <template #actions>
        <Select v-if="entityOptions.length > 1" v-model="entityId" :options="entityOptions" size="sm" width="200px" aria-label="选择经营主体" />
        <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
        <Button v-if="canEdit" icon="i-lucide:plus" @click="openCreate(null)">新增根组织</Button>
      </template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />
    <Alert v-else-if="!entityId" type="warning" show-icon class="mb-4" title="请先在顶部选择经营主体，组织架构按主体隔离维护。" />

    <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_21rem]">
      <section class="min-w-0 rounded-xl border border-light bg-primary p-3">
        <Skeleton v-if="!ready && loading" variant="list" :rows="8" />
        <div v-else-if="!tree.length" class="grid place-items-center gap-2 py-16 text-center">
          <span class="i-lucide:network h-8 w-8 text-quaternary" aria-hidden="true" />
          <p class="m-0 text-sm text-tertiary">当前主体还没有组织节点</p>
          <Button v-if="canEdit" variant="outline" size="sm" icon="i-lucide:plus" @click="openCreate(null)">新增根组织</Button>
        </div>
        <OrgTree v-else :nodes="tree" :selected-id="selected?.id" @select="selected = $event" />
      </section>

      <aside class="min-w-0 rounded-xl border border-light bg-primary p-4">
        <template v-if="selected">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <h2 class="m-0 truncate text-base font-600">{{ selected.name }}</h2>
              <span class="mt-1 block text-xs text-tertiary">{{ selected.type }} · 排序 {{ selected.sort }}</span>
            </div>
            <Tag :color="selected.status === 'active' ? 'green' : 'gray'">{{ selected.status === 'active' ? '正常' : '停用' }}</Tag>
          </div>
          <dl class="m-0 mt-4 grid gap-2.5 text-sm">
            <div class="flex items-start justify-between gap-3">
              <dt class="shrink-0 text-tertiary">组织 ID</dt>
              <dd class="m-0 truncate font-mono text-xs">{{ selected.id }}</dd>
            </div>
            <div class="flex items-start justify-between gap-3">
              <dt class="shrink-0 text-tertiary">上级组织</dt>
              <dd class="m-0 truncate">{{ flat.find((node) => node.id === selected?.parentId)?.name || '无（根节点）' }}</dd>
            </div>
            <div class="flex items-start justify-between gap-3">
              <dt class="shrink-0 text-tertiary">层级深度</dt>
              <dd class="m-0">{{ selected.path.length }} 级</dd>
            </div>
            <div class="flex items-start justify-between gap-3">
              <dt class="shrink-0 text-tertiary">下级组织</dt>
              <dd class="m-0">{{ selected.children?.length ?? 0 }} 个</dd>
            </div>
          </dl>
          <div v-if="canEdit" class="mt-4 flex flex-wrap gap-2 border-t border-light pt-4">
            <Button v-if="canEdit" variant="outline" size="sm" icon="i-lucide:plus" @click="openCreate(selected)">新增子组织</Button>
            <Button v-if="canEdit" size="sm" icon="i-lucide:pencil" @click="openEdit(selected)">编辑</Button>
          </div>
          <p v-if="isStandalone" class="mb-0 mt-3 text-xs text-tertiary">单体租户的组织通常与唯一主体同名，可继续按部门细分。</p>
        </template>
        <div v-else class="grid place-items-center gap-2 py-12 text-center">
          <span class="i-lucide:mouse-pointer-click h-6 w-6 text-quaternary" aria-hidden="true" />
          <p class="m-0 text-sm text-tertiary">从左侧选择一个组织节点查看详情</p>
        </div>
      </aside>
    </div>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑组织节点' : '新增组织节点'" size="min(30rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <FormItem label="组织名称" prop="name" required>
            <Input v-model="form.name" :maxlength="60" placeholder="例如：售后服务部" />
          </FormItem>
          <FormItem label="上级组织">
            <Select v-model="form.parentId" :options="parentOptions" clearable placeholder="留空表示根组织" />
          </FormItem>
          <FormItem label="组织类型">
            <Select v-model="form.type" :options="typeOptions" />
          </FormItem>
          <FormItem label="排序">
            <Input v-model.number="form.sort" type="number" min="0" />
          </FormItem>
          <FormItem label="状态">
            <Switch v-model="form.status" active-value="active" inactive-value="suspended" active-text="正常" inactive-text="停用" />
          </FormItem>
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
  "title": "组织架构",
  "permission": "org:view"
}
</router>
