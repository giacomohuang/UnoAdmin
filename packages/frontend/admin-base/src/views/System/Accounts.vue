<script setup lang="ts">
/**
 * 账户管理：管理端登录身份与数据授权范围。
 *
 * 账户同时绑定经营主体（数据隔离单元）、组织（本人组织范围）与角色（权限码），
 * 三者共同决定登录后能看到什么、能操作什么。
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
import { Tooltip } from '@mcistudio/unoui-vue/tooltip'
import type { Account, AccountId, OrgUnit, Role, SaveAccount } from '@unoadmin/core'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { accounts: accountService, roles: roleService, orgs: orgService } = useAdminBaseServices()

const items = ref<Account[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const appliedKeyword = ref('')
const loading = ref(false)
const ready = ref(false)
const error = ref('')

const roles = ref<Role[]>([])
const orgOptions = ref<Array<{ label: string; value: string }>>([])
const choicesLoading = ref(false)

const drawerOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const editing = ref<Account | null>(null)
const formRef = ref<InstanceType<typeof Form>>()
const form = reactive<SaveAccount>({
  username: '',
  displayName: '',
  password: '',
  email: '',
  phone: '',
  status: 'active',
  dataScope: 'entity',
  legalEntityIds: [],
  orgIds: [],
  roleIds: []
})

const canCreate = computed(() => session.can('account:create'))
const canUpdate = computed(() => session.can('account:update'))
const entityOptions = computed(() => session.state.legalEntities.map((entity) => ({ label: entity.name, value: entity.id })))
const roleOptions = computed(() => roles.value.map((role) => ({
  label: `${role.name}（${role.dataScope === 'tenant' ? '租户级' : role.dataScope === 'self' ? '本人组织' : '主体级'}）`,
  value: role.id
})))

const rules: FormRules = {
  username: [{ required: true, message: '请输入登录账号' }],
  displayName: [{ required: true, message: '请输入姓名' }]
}

const scopeOptions = [
  { label: '租户级（全部经营主体）', value: 'tenant' },
  { label: '主体级（授权主体）', value: 'entity' },
  { label: '本人组织及下级', value: 'self' }
]

const columns = computed<TableColumn<Account>[]>(() => [
  { key: 'username', title: '账户', minWidth: 190 },
  { key: 'legalEntityNames', title: '经营主体', minWidth: 220 },
  { key: 'dataScope', title: '数据范围', width: 120 },
  { key: 'roleNames', title: '角色', minWidth: 170 },
  { key: 'status', title: '状态', width: 90 },
  ...(canUpdate.value ? [{ key: 'actions', title: '操作', width: 90, align: 'center' as const, fixed: 'right' as const }] : [])
])

/** 数据范围的中文标签与配色。 */
function scopeTag(scope: Account['dataScope']) {
  if (scope === 'tenant') return { label: '租户级', color: 'blue' as const }
  if (scope === 'self') return { label: '本人组织', color: 'orange' as const }
  return { label: '主体级', color: 'cyan' as const }
}

/** 加载账户分页。 */
async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await accountService.list({
      page: page.value,
      pageSize: pageSize.value,
      search: appliedKeyword.value || undefined
    })
    items.value = result.items
    total.value = result.total
    ready.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '账户加载失败'
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

/** 扁平化组织树，并带上主体名称作为前缀。 */
function flattenOrgs(nodes: OrgUnit[], entityName: string, depth = 0): Array<{ label: string; value: string }> {
  return nodes.flatMap((node) => [
    { label: `${entityName} / ${'— '.repeat(depth)}${node.name}`, value: node.id },
    ...flattenOrgs(node.children ?? [], entityName, depth + 1)
  ])
}

/** 按已选主体并行加载组织选项，任一失败时保留其它结果。 */
async function loadOrgOptions() {
  if (!form.legalEntityIds.length) {
    orgOptions.value = []
    return
  }
  const results = await Promise.allSettled(form.legalEntityIds.map(async (id) => {
    const entityName = session.state.legalEntities.find((entity) => entity.id === id)?.name ?? id
    return flattenOrgs(await orgService.tree(id), entityName)
  }))
  orgOptions.value = results.flatMap((result) => (result.status === 'fulfilled' ? result.value : []))
  // 主体变化后清理已失效的组织授权。
  const valid = new Set(orgOptions.value.map((option) => option.value))
  form.orgIds = form.orgIds.filter((id) => valid.has(id))
}

/** 打开抽屉时并行加载角色与组织选项。 */
async function loadChoices() {
  choicesLoading.value = true
  const results = await Promise.allSettled([roleService.list({ page: 1, pageSize: 100 })])
  const [roleResult] = results
  if (roleResult.status === 'fulfilled') roles.value = roleResult.value.items
  await loadOrgOptions()
  choicesLoading.value = false
}

/** 打开新建或编辑抽屉。 */
async function open(record?: Account) {
  editing.value = record ?? null
  Object.assign(form, {
    username: record?.username ?? '',
    displayName: record?.displayName ?? '',
    password: '',
    email: record?.email ?? '',
    phone: record?.phone ?? '',
    status: record?.status ?? 'active',
    dataScope: record?.dataScope ?? 'entity',
    legalEntityIds: [...(record?.legalEntityIds ?? [])],
    orgIds: [...(record?.orgIds ?? [])],
    roleIds: [...(record?.roleIds ?? [])]
  })
  formError.value = ''
  drawerOpen.value = true
  await loadChoices()
}

/** 校验并保存账户。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  if (!editing.value && (form.password ?? '').length < 8) {
    formError.value = '新建账户必须设置至少 8 位密码'
    return
  }
  if (form.dataScope !== 'tenant' && !form.legalEntityIds.length) {
    formError.value = '主体级或本人组织范围至少需要选择一个经营主体'
    return
  }
  formError.value = ''
  saving.value = true
  try {
    const payload: SaveAccount = { ...form, password: form.password || undefined }
    if (editing.value) await accountService.update(editing.value.id as AccountId, payload)
    else await accountService.create(payload)
    message.success('账户已保存')
    drawerOpen.value = false
    await load()
  } catch (cause) {
    formError.value = cause instanceof Error ? cause.message : '保存失败'
  } finally {
    saving.value = false
  }
}

/** 主体选择变化后重新加载组织选项。 */
watch(
  () => [...form.legalEntityIds],
  () => {
    if (drawerOpen.value && !choicesLoading.value) void loadOrgOptions()
  }
)

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>账户管理</template>
      <template #status>共 {{ total }} 个账户</template>
      <template #description>账户按经营主体与组织授权；角色提供权限码与数据范围。</template>
      <template #actions>
        <form class="flex w-full shrink-0 items-center justify-end gap-2" @submit.prevent="search">
          <Input v-model="keyword" prefix-icon="i-lucide:search" clearable placeholder="搜索账号、姓名或邮箱" class="w-56" @clear="search" />
          <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
          <Button v-if="canCreate" icon="i-lucide:user-plus" @click="open()">新建账户</Button>
        </form>
      </template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <Skeleton v-if="!ready && loading" variant="table" :rows="8" :columns="6" />
    <template v-else>
      <Table :rows="items" :columns="columns" row-key="id" min-width="1040px" empty-text="暂无账户" :aria-busy="loading">
        <template #cell-username="{ row }">
          <div class="flex min-w-0 items-center gap-2">
            <span class="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-secondary text-xs font-600">{{ row.displayName.slice(0, 1) }}</span>
            <div class="flex min-w-0 flex-col">
              <span class="flex items-center gap-1 truncate font-medium">
                {{ row.displayName }}
                <Tag v-if="row.platformAdmin" color="purple" size="sm">平台</Tag>
              </span>
              <span class="truncate text-xs text-tertiary">{{ row.username }}<template v-if="row.email"> · {{ row.email }}</template></span>
            </div>
          </div>
        </template>
        <template #cell-legalEntityNames="{ row }">
          <div class="flex flex-wrap items-center gap-1">
            <Tag v-for="name in row.legalEntityNames.slice(0, 2)" :key="name" color="cyan" class="max-w-36">
              <span class="block truncate">{{ name }}</span>
            </Tag>
            <Tooltip v-if="row.legalEntityNames.length > 2" :title="row.legalEntityNames.slice(2).join('、')">
              <Tag color="gray">+{{ row.legalEntityNames.length - 2 }}</Tag>
            </Tooltip>
            <span v-if="!row.legalEntityNames.length" class="text-xs text-tertiary">全部（租户级）</span>
          </div>
        </template>
        <template #cell-dataScope="{ row }">
          <Tag :color="scopeTag(row.dataScope).color">{{ scopeTag(row.dataScope).label }}</Tag>
        </template>
        <template #cell-roleNames="{ row }">
          <div class="flex flex-wrap items-center gap-1">
            <Tag v-for="name in row.roleNames" :key="name" color="blue" class="max-w-36">
              <span class="block truncate">{{ name }}</span>
            </Tag>
            <span v-if="!row.roleNames.length" class="text-xs text-tertiary">未分配</span>
          </div>
        </template>
        <template #cell-status="{ row }">
          <Tag :color="row.status === 'active' ? 'green' : 'gray'">{{ row.status === 'active' ? '启用' : '停用' }}</Tag>
        </template>
        <template #cell-actions="{ row }">
          <Button variant="link" size="sm" icon="i-lucide:pencil" @click="open(row)">编辑</Button>
        </template>
      </Table>

      <div v-if="total > pageSize" class="mt-4 flex justify-end">
        <Pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100]" layout="prev, pager, next, sizes" size="sm" @change="load" />
      </div>
    </template>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑账户' : '新建账户'" size="min(44rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <section>
            <h3 class="mb-3 mt-0 text-sm font-600">账号信息</h3>
            <div class="grid gap-x-4 sm:grid-cols-2">
              <FormItem label="登录账号" prop="username" required>
                <Input v-model="form.username" :disabled="Boolean(editing)" :maxlength="64" autocomplete="off" placeholder="例如：chen.yu" />
              </FormItem>
              <FormItem label="姓名" prop="displayName" required>
                <Input v-model="form.displayName" :maxlength="50" autocomplete="off" placeholder="例如：陈宇" />
              </FormItem>
              <FormItem :label="editing ? '重置密码（留空不修改）' : '初始密码'" :required="!editing">
                <Input v-model="form.password" type="password" show-password autocomplete="new-password" placeholder="至少 8 位" />
              </FormItem>
              <FormItem label="手机号">
                <Input v-model="form.phone" :maxlength="25" placeholder="选填" />
              </FormItem>
              <FormItem label="邮箱" class="sm:col-span-2">
                <Input v-model="form.email" type="email" :maxlength="120" placeholder="选填" />
              </FormItem>
            </div>
          </section>

          <section class="border-t border-light pt-4">
            <h3 class="mb-3 mt-0 text-sm font-600">数据授权</h3>
            <div class="grid gap-x-4 sm:grid-cols-2">
              <FormItem label="数据范围" class="sm:col-span-2">
                <Select v-model="form.dataScope" :options="scopeOptions" />
              </FormItem>
              <FormItem label="经营主体" class="sm:col-span-2" :required="form.dataScope !== 'tenant'">
                <Select
                  v-model="form.legalEntityIds"
                  :options="entityOptions"
                  multiple
                  filterable
                  clearable
                  collapse-tags
                  :max-collapse-tags="2"
                  :disabled="form.dataScope === 'tenant'"
                  placeholder="选择账户可访问的经营主体"
                />
              </FormItem>
              <FormItem label="组织（本人组织范围）" class="sm:col-span-2">
                <Select
                  v-model="form.orgIds"
                  :options="orgOptions"
                  multiple
                  filterable
                  clearable
                  collapse-tags
                  :max-collapse-tags="2"
                  :loading="choicesLoading"
                  :disabled="!form.legalEntityIds.length"
                  placeholder="选择账户所属组织，可多选"
                />
              </FormItem>
              <FormItem label="角色" class="sm:col-span-2">
                <Select v-model="form.roleIds" :options="roleOptions" multiple filterable clearable collapse-tags :max-collapse-tags="2" placeholder="选择角色以授予权限码" />
              </FormItem>
              <FormItem label="状态" class="sm:col-span-2">
                <Switch v-model="form.status" active-value="active" inactive-value="disabled" active-text="启用" inactive-text="停用" />
              </FormItem>
            </div>
          </section>
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
  "title": "账户管理",
  "permission": "account:view"
}
</router>
