<script setup lang="ts">
/**
 * 租户管理：维护采购与数据隔离的最外层边界。
 *
 * 集团客户一个租户挂多个经营主体；单体客户一个租户只挂一个主体，
 * 新建单体租户时后端会自动生成唯一主体。
 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Button } from '@mcistudio/unoui-vue/button'
import { Drawer } from '@mcistudio/unoui-vue/drawer'
import { Form, FormItem, type FormRules } from '@mcistudio/unoui-vue/form'
import { Input } from '@mcistudio/unoui-vue/input'
import { message } from '@mcistudio/unoui-vue/message'
import { Pagination } from '@mcistudio/unoui-vue/pagination'
import { Select } from '@mcistudio/unoui-vue/select'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Switch } from '@mcistudio/unoui-vue/switch'
import { Table, type TableColumn } from '@mcistudio/unoui-vue/table'
import { Tag } from '@mcistudio/unoui-vue/tag'
import type { SaveTenant, Tenant, TenantId } from '@unoadmin/core'
import { computed, onMounted, reactive, ref } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { tenants: tenantService } = useAdminBaseServices()

const items = ref<Tenant[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const appliedKeyword = ref('')
const loading = ref(false)
const ready = ref(false)
const error = ref('')

const drawerOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const formRef = ref<InstanceType<typeof Form>>()
const editing = ref<Tenant | null>(null)
const form = reactive<SaveTenant>({ name: '', code: '', mode: 'group', plan: 'standard', contact: '', status: 'active' })

const isPlatform = computed(() => Boolean(session.state.account?.platformAdmin))
const canCreate = computed(() => session.can('tenant:create'))
const canUpdate = computed(() => session.can('tenant:update'))

const rules: FormRules = {
  name: [{ required: true, message: '请输入租户名称' }],
  code: [{ required: true, message: '请输入租户编码' }],
  mode: [{ required: true, message: '请选择租户形态' }]
}

const columns = computed<TableColumn<Tenant>[]>(() => [
  { key: 'name', title: '租户', minWidth: 200 },
  { key: 'mode', title: '形态', width: 100 },
  { key: 'plan', title: '套餐', width: 110 },
  { key: 'entities', title: '经营主体', minWidth: 240 },
  { key: 'accountCount', title: '账户', width: 80, align: 'right' },
  { key: 'contact', title: '联系人', minWidth: 120 },
  { key: 'status', title: '状态', width: 90 },
  ...(canUpdate.value ? [{ key: 'actions', title: '操作', width: 90, align: 'center' as const, fixed: 'right' as const }] : [])
])

/** 加载租户列表；非平台账号由后端收敛到自身租户。 */
async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await tenantService.list({
      page: page.value,
      pageSize: pageSize.value,
      search: appliedKeyword.value || undefined
    })
    items.value = result.items
    total.value = result.total
    ready.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '租户加载失败'
  } finally {
    loading.value = false
  }
}

/** 应用搜索词并从第一页重新查询。 */
function search() {
  appliedKeyword.value = keyword.value.trim()
  page.value = 1
  void load()
}

/** 打开新建或编辑抽屉，并重置表单状态。 */
function open(record?: Tenant) {
  editing.value = record ?? null
  Object.assign(form, {
    name: record?.name ?? '',
    code: record?.code ?? '',
    mode: record?.mode ?? 'group',
    plan: record?.plan ?? 'standard',
    contact: record?.contact ?? '',
    status: record?.status ?? 'active'
  })
  formError.value = ''
  drawerOpen.value = true
}

/** 校验后创建或更新租户，并刷新会话中的租户列表。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  formError.value = ''
  saving.value = true
  try {
    if (editing.value) await tenantService.update(editing.value.id as TenantId, { ...form })
    else await tenantService.create({ ...form })
    message.success('租户已保存')
    drawerOpen.value = false
    await Promise.allSettled([load(), session.refresh()])
  } catch (cause) {
    formError.value = cause instanceof Error ? cause.message : '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>租户管理</template>
      <template #status>共 {{ total }} 个租户</template>
      <template #description>租户是采购与数据隔离的最外层；集团客户一个租户多个主体，单体客户一个租户一个主体。</template>
      <template #actions>
        <form class="flex w-full shrink-0 items-center justify-end gap-2" @submit.prevent="search">
          <Input v-model="keyword" prefix-icon="i-lucide:search" clearable placeholder="搜索租户名称或编码" class="w-56" @clear="search" />
          <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
          <Button v-if="canCreate && isPlatform" icon="i-lucide:plus" @click="open()">新建租户</Button>
        </form>
      </template>
    </PageHeader>

    <Alert v-if="!isPlatform" type="info" show-icon class="mb-4" title="当前账号只能查看自己所属的租户；跨租户管理由平台运营管理员完成。" />
    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <Skeleton v-if="!ready && loading" variant="table" :rows="6" :columns="7" />
    <template v-else>
      <Table :rows="items" :columns="columns" row-key="id" min-width="1080px" empty-text="暂无租户" :aria-busy="loading">
        <template #cell-name="{ row }">
          <div class="flex min-w-0 flex-col">
            <span class="truncate font-medium">{{ row.name }}</span>
            <span class="text-xs text-tertiary">{{ row.code }}</span>
          </div>
        </template>
        <template #cell-mode="{ row }">
          <Tag :color="row.mode === 'group' ? 'blue' : 'gray'">{{ row.mode === 'group' ? '集团' : '单体' }}</Tag>
        </template>
        <template #cell-plan="{ row }">
          <span class="text-sm text-secondary">{{ row.plan }}</span>
        </template>
        <template #cell-entities="{ row }">
          <div class="flex flex-wrap items-center gap-1">
            <Tag v-for="entity in (row.legalEntities ?? []).slice(0, 2)" :key="entity.id" :color="entity.type === 'GROUP' ? 'purple' : 'cyan'" class="max-w-40">
              <span class="block truncate">{{ entity.name }}</span>
            </Tag>
            <Tag v-if="(row.legalEntities?.length ?? 0) > 2" color="gray">+{{ (row.legalEntities?.length ?? 0) - 2 }}</Tag>
            <span v-if="!row.legalEntities?.length" class="text-xs text-tertiary">尚未创建主体</span>
          </div>
        </template>
        <template #cell-accountCount="{ row }">
          <span class="tabular-nums">{{ row.accountCount ?? 0 }}</span>
        </template>
        <template #cell-contact="{ row }">{{ row.contact || '-' }}</template>
        <template #cell-status="{ row }">
          <Tag :color="row.status === 'active' ? 'green' : 'gray'">{{ row.status === 'active' ? '正常' : '暂停' }}</Tag>
        </template>
        <template #cell-actions="{ row }">
          <Button variant="link" size="sm" icon="i-lucide:pencil" @click="open(row)">编辑</Button>
        </template>
      </Table>

      <div v-if="total > pageSize" class="mt-4 flex justify-end">
        <Pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50]" layout="prev, pager, next, sizes" size="sm" @change="load" />
      </div>
    </template>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑租户' : '新建租户'" size="min(34rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Alert
          type="info"
          show-icon
          :title="form.mode === 'group' ? '集团租户：可在租户下创建多个经营主体，集团层面统一查看。' : '单体租户：保存后自动生成唯一经营主体，后续不可再新增主体。'"
        />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <div class="grid gap-x-4 sm:grid-cols-2">
            <FormItem label="租户名称" prop="name" required>
              <Input v-model="form.name" :maxlength="60" placeholder="例如：汉唐控股集团" />
            </FormItem>
            <FormItem label="租户编码" prop="code" required>
              <Input v-model="form.code" :maxlength="32" placeholder="例如：HANTANG" />
            </FormItem>
            <FormItem label="客户形态" prop="mode" required>
              <Select v-model="form.mode" :options="[{ label: '集团统一采购', value: 'group' }, { label: '单体公司采购', value: 'standalone' }]" />
            </FormItem>
            <FormItem label="套餐">
              <Select v-model="form.plan" :options="[{ label: '标准版', value: 'standard' }, { label: '企业版', value: 'enterprise' }, { label: '旗舰版', value: 'flagship' }]" />
            </FormItem>
            <FormItem label="联系人" class="sm:col-span-2">
              <Input v-model="form.contact" :maxlength="60" placeholder="例如：集团信息中心 / 张三" />
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
  "title": "租户管理",
  "permission": "tenant:view"
}
</router>
