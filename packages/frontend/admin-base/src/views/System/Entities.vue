<script setup lang="ts">
/**
 * 经营主体管理：租户内部相互隔离的经营实体。
 *
 * 集团租户可维护一个集团总部 + 多个经营公司；单体租户只有唯一主体，
 * 由新建租户时自动生成，本页只允许查看与编辑。
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
import { Table, type TableColumn } from '@mcistudio/unoui-vue/table'
import { Tag } from '@mcistudio/unoui-vue/tag'
import type { LegalEntity, LegalEntityId, SaveLegalEntity } from '@unoadmin/core'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { legalEntities: entityService } = useAdminBaseServices()

const items = ref<LegalEntity[]>([])
const loading = ref(false)
const ready = ref(false)
const error = ref('')

const drawerOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const formRef = ref<InstanceType<typeof Form>>()
const editing = ref<LegalEntity | null>(null)
const form = reactive<SaveLegalEntity>({
  tenantId: '',
  parentId: '',
  type: 'COMPANY',
  name: '',
  code: '',
  address: '',
  status: 'active'
})

const tenant = computed(() => session.state.currentTenant)
const isStandalone = computed(() => tenant.value?.mode === 'standalone')
const canEdit = computed(() => session.can('entity:edit'))
const groupEntity = computed(() => items.value.find((entity) => entity.type === 'GROUP') ?? null)

const rules: FormRules = {
  name: [{ required: true, message: '请输入主体名称' }],
  code: [{ required: true, message: '请输入主体编码' }]
}

const columns = computed<TableColumn<LegalEntity>[]>(() => [
  { key: 'name', title: '经营主体', minWidth: 220 },
  { key: 'type', title: '类型', width: 110 },
  { key: 'code', title: '编码', width: 130 },
  { key: 'parentId', title: '上级主体', minWidth: 160 },
  { key: 'address', title: '注册地址', minWidth: 200 },
  { key: 'status', title: '状态', width: 90 },
  ...(canEdit.value ? [{ key: 'actions', title: '操作', width: 90, align: 'center' as const, fixed: 'right' as const }] : [])
])

/** 集团主体排在前面，其下的经营公司紧随其后并缩进展示。 */
const rows = computed(() => {
  const groups = items.value.filter((entity) => entity.type === 'GROUP')
  const result: LegalEntity[] = []
  const seen = new Set<string>()
  for (const group of groups) {
    result.push(group)
    seen.add(group.id)
    for (const company of items.value.filter((entity) => entity.parentId === group.id)) {
      result.push(company)
      seen.add(company.id)
    }
  }
  for (const rest of items.value) if (!seen.has(rest.id)) result.push(rest)
  return result
})

const parentOptions = computed(() =>
  items.value
    .filter((entity) => entity.type === 'GROUP' && entity.id !== editing.value?.id)
    .map((entity) => ({ label: `${entity.name}（${entity.code}）`, value: entity.id }))
)

/** 主体 ID 到名称的映射；表格行里的 parentId 是普通字符串，因此键类型放宽为 string。 */
const parentNames = computed(() => new Map<string, string>(items.value.map((entity) => [entity.id, entity.name])))

/** 加载当前租户下被授权的主体。 */
async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    items.value = await entityService.list({ page: 1, pageSize: 100, tenantId: tenant.value?.id })
    ready.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '经营主体加载失败'
  } finally {
    loading.value = false
  }
}

/** 打开抽屉；单体租户默认只允许编辑现有主体。 */
function open(record?: LegalEntity) {
  editing.value = record ?? null
  Object.assign(form, {
    tenantId: record?.tenantId ?? tenant.value?.id ?? '',
    parentId: record?.parentId ?? groupEntity.value?.id ?? '',
    type: record?.type ?? (groupEntity.value ? 'COMPANY' : 'GROUP'),
    name: record?.name ?? '',
    code: record?.code ?? '',
    address: record?.address ?? '',
    status: record?.status ?? 'active'
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 校验并保存主体，集团总部只能有一个。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  formError.value = ''
  saving.value = true
  try {
    const payload: SaveLegalEntity = { ...form, parentId: form.type === 'GROUP' ? '' : form.parentId }
    if (editing.value) await entityService.update(editing.value.id as LegalEntityId, payload)
    else await entityService.create(payload)
    message.success('经营主体已保存')
    drawerOpen.value = false
    await Promise.allSettled([load(), session.refresh()])
  } catch (cause) {
    formError.value = cause instanceof Error ? cause.message : '保存失败'
  } finally {
    saving.value = false
  }
}

/** 集团主体不挂在其它主体下，切换类型时清空上级。 */
watch(
  () => form.type,
  (type) => {
    if (type === 'GROUP') form.parentId = ''
    else if (!form.parentId && groupEntity.value) form.parentId = groupEntity.value.id
  }
)

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>经营主体</template>
      <template #status>{{ items.length }} 个主体</template>
      <template #description>{{ tenant?.name }} · {{ isStandalone ? '单体租户只有唯一主体' : '集团租户可有多个独立经营主体' }}</template>
      <template #actions>
        <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
        <Button v-if="canEdit && !isStandalone" icon="i-lucide:plus" @click="open()">新建主体</Button>
      </template>
    </PageHeader>

    <Alert
      v-if="isStandalone"
      type="info"
      show-icon
      class="mb-4"
      title="单体租户在创建租户时已生成唯一经营主体，如需多个主体请把租户形态调整为集团。"
    />
    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <Skeleton v-if="!ready && loading" variant="table" :rows="6" :columns="7" />
    <Table v-else :rows="rows" :columns="columns" row-key="id" min-width="1080px" empty-text="暂无经营主体" :aria-busy="loading">
      <template #cell-name="{ row }">
        <span class="flex min-w-0 items-center gap-2" :class="row.type === 'COMPANY' ? 'pl-5' : ''">
          <span v-if="row.type === 'GROUP'" class="i-lucide:building h-4 w-4 shrink-0 text-violet-600" aria-hidden="true" />
          <span v-else class="h-4 w-4 shrink-0 border-l border-b border-dashed border-medium" aria-hidden="true" />
          <span class="truncate font-medium">{{ row.name }}</span>
        </span>
      </template>
      <template #cell-type="{ row }">
        <Tag :color="row.type === 'GROUP' ? 'purple' : 'cyan'">{{ row.type === 'GROUP' ? '集团总部' : '经营公司' }}</Tag>
      </template>
      <template #cell-code="{ row }"><span class="font-mono text-xs">{{ row.code }}</span></template>
      <template #cell-parentId="{ row }">{{ row.parentId ? parentNames.get(row.parentId) ?? row.parentId : '-' }}</template>
      <template #cell-address="{ row }"><span class="line-clamp-2 text-sm text-secondary">{{ row.address || '-' }}</span></template>
      <template #cell-status="{ row }">
        <Tag :color="row.status === 'active' ? 'green' : 'gray'">{{ row.status === 'active' ? '正常' : '暂停' }}</Tag>
      </template>
      <template #cell-actions="{ row }">
        <Button variant="link" size="sm" icon="i-lucide:pencil" @click="open(row)">编辑</Button>
      </template>
    </Table>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑经营主体' : '新建经营主体'" size="min(34rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <div class="grid gap-x-4 sm:grid-cols-2">
            <FormItem label="主体名称" prop="name" required class="sm:col-span-2">
              <Input v-model="form.name" :maxlength="80" placeholder="例如：汉唐华东实业有限公司" />
            </FormItem>
            <FormItem label="主体编码" prop="code" required>
              <Input v-model="form.code" :maxlength="32" placeholder="例如：HT-EAST" />
            </FormItem>
            <FormItem label="主体类型">
              <Select
                v-model="form.type"
                :options="[
                  { label: '集团总部', value: 'GROUP' },
                  { label: '经营公司', value: 'COMPANY' }
                ]"
                :disabled="isStandalone"
              />
            </FormItem>
            <FormItem v-if="form.type === 'COMPANY'" label="上级集团" class="sm:col-span-2">
              <Select v-model="form.parentId" :options="parentOptions" clearable placeholder="选择集团总部主体" />
            </FormItem>
            <FormItem label="注册地址" class="sm:col-span-2">
              <Input v-model="form.address" :maxlength="120" placeholder="选填" />
            </FormItem>
            <FormItem label="状态" class="sm:col-span-2">
              <Switch v-model="form.status" active-value="active" inactive-value="suspended" active-text="正常" inactive-text="暂停" />
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
  "title": "经营主体",
  "permission": "entity:view"
}
</router>
