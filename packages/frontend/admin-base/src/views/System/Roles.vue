<script setup lang="ts">
/**
 * 角色管理：把权限码与数据范围打包成可分配的角色。
 *
 * 角色的数据范围会与账户范围取并集，因此这里只决定「最小可见范围」，
 * 具体能看哪些经营主体仍由账户授权列表约束。
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
import type { Permission, Role, RoleId, SaveRole } from '@unoadmin/core'
import { computed, onMounted, reactive, ref } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import PermissionTree from '../../components/PermissionTree.vue'
import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const { roles: roleService, permissions: permissionService } = useAdminBaseServices()

const items = ref<Role[]>([])
const permissions = ref<Permission[]>([])
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
const editing = ref<Role | null>(null)
const formRef = ref<InstanceType<typeof Form>>()
const form = reactive<SaveRole>({
  code: '',
  name: '',
  description: '',
  status: 'active',
  dataScope: 'entity',
  legalEntityId: '',
  permissionCodes: []
})

const canCreate = computed(() => session.can('role:create'))
const canUpdate = computed(() => session.can('role:update'))
const isStandalone = computed(() => session.state.currentTenant?.mode === 'standalone')
const entityOptions = computed(() => [
  { label: '租户级（适用于全部主体）', value: '' },
  ...session.state.legalEntities.map((entity) => ({ label: entity.name, value: entity.id }))
])
const scopeOptions = [
  { label: '租户级（全部经营主体）', value: 'tenant' },
  { label: '主体级（授权主体）', value: 'entity' },
  { label: '本人组织及下级', value: 'self' }
]

const rules: FormRules = {
  name: [{ required: true, message: '请输入角色名称' }],
  code: [{ required: true, message: '请输入角色编码' }]
}

const columns = computed<TableColumn<Role>[]>(() => [
  { key: 'name', title: '角色', minWidth: 220 },
  { key: 'dataScope', title: '数据范围', width: 110 },
  { key: 'permissionCodes', title: '权限', width: 100 },
  { key: 'userCount', title: '成员', width: 80, align: 'right' },
  { key: 'description', title: '说明', minWidth: 220 },
  { key: 'status', title: '状态', width: 90 },
  ...(canUpdate.value ? [{ key: 'actions', title: '操作', width: 90, align: 'center' as const, fixed: 'right' as const }] : [])
])

/** 数据范围标签。 */
function scopeLabel(scope: Role['dataScope']) {
  if (scope === 'tenant') return { label: '租户级', color: 'blue' as const }
  if (scope === 'self') return { label: '本人组织', color: 'orange' as const }
  return { label: '主体级', color: 'cyan' as const }
}

/** 加载角色分页。 */
async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await roleService.list({
      page: page.value,
      pageSize: pageSize.value,
      search: appliedKeyword.value || undefined
    })
    items.value = result.items
    total.value = result.total
    ready.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '角色加载失败'
  } finally {
    loading.value = false
  }
}

/** 应用搜索词并重新查询。 */
function search() {
  appliedKeyword.value = keyword.value.trim()
  page.value = 1
  void load()
}

/** 打开抽屉并加载权限目录。 */
async function open(record?: Role) {
  editing.value = record ?? null
  Object.assign(form, {
    code: record?.code ?? '',
    name: record?.name ?? '',
    description: record?.description ?? '',
    status: record?.status ?? 'active',
    dataScope: record?.dataScope ?? 'entity',
    legalEntityId: record?.legalEntityId ?? '',
    permissionCodes: [...(record?.permissionCodes ?? [])]
  })
  formError.value = ''
  drawerOpen.value = true
  if (!permissions.value.length) {
    try {
      permissions.value = await permissionService.list()
    } catch (cause) {
      formError.value = cause instanceof Error ? cause.message : '权限目录加载失败'
    }
  }
}

/** 校验并保存角色。 */
async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate()
  if (!valid) return
  formError.value = ''
  saving.value = true
  try {
    const payload: SaveRole = { ...form, legalEntityId: form.dataScope === 'tenant' ? '' : form.legalEntityId }
    if (editing.value) await roleService.update(editing.value.id as RoleId, payload)
    else await roleService.create(payload)
    message.success('角色已保存')
    drawerOpen.value = false
    await load()
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
      <template #title>角色管理</template>
      <template #status>共 {{ total }} 个角色</template>
      <template #description>角色聚合权限码与数据范围，账户通过角色获得菜单、路由与按钮权限。</template>
      <template #actions>
        <form class="flex w-full shrink-0 items-center justify-end gap-2" @submit.prevent="search">
          <Input v-model="keyword" prefix-icon="i-lucide:search" clearable placeholder="搜索角色名称、编码或权限" class="w-56" @clear="search" />
          <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
          <Button v-if="canCreate" icon="i-lucide:plus" @click="open()">新建角色</Button>
        </form>
      </template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <Skeleton v-if="!ready && loading" variant="table" :rows="6" :columns="6" />
    <template v-else>
      <Table :rows="items" :columns="columns" row-key="id" min-width="1000px" empty-text="暂无角色" :aria-busy="loading">
        <template #cell-name="{ row }">
          <div class="flex min-w-0 flex-col">
            <span class="flex items-center gap-1 truncate font-medium">
              {{ row.name }}
              <Tag v-if="row.builtIn" color="gray" size="sm">内置</Tag>
            </span>
            <span class="font-mono text-xs text-tertiary">{{ row.code }}</span>
          </div>
        </template>
        <template #cell-dataScope="{ row }">
          <Tag :color="scopeLabel(row.dataScope).color">{{ scopeLabel(row.dataScope).label }}</Tag>
        </template>
        <template #cell-permissionCodes="{ row }">
          <Tag v-if="row.permissionCodes.includes('*')" color="purple">全部权限</Tag>
          <span v-else class="text-sm tabular-nums">{{ row.permissionCodes.length }}</span>
        </template>
        <template #cell-userCount="{ row }"><span class="tabular-nums">{{ row.userCount }}</span></template>
        <template #cell-description="{ row }"><span class="line-clamp-2 text-sm text-secondary">{{ row.description || '-' }}</span></template>
        <template #cell-status="{ row }">
          <Tag :color="row.status === 'active' ? 'green' : 'gray'">{{ row.status === 'active' ? '正常' : '停用' }}</Tag>
        </template>
        <template #cell-actions="{ row }">
          <Button variant="link" size="sm" icon="i-lucide:shield-check" @click="open(row)">授权</Button>
        </template>
      </Table>

      <div v-if="total > pageSize" class="mt-4 flex justify-end">
        <Pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50]" layout="prev, pager, next, sizes" size="sm" @change="load" />
      </div>
    </template>

    <Drawer v-model:visible="drawerOpen" :title="editing ? '编辑角色与授权' : '新建角色'" size="min(44rem, 100vw)" :show-close="!saving" :close-on-click-modal="!saving">
      <div class="min-w-0 space-y-4 p-4">
        <Alert v-if="formError" :title="formError" type="error" show-icon />
        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="sm" item-gap="12px" :disabled="saving">
          <div class="grid gap-x-4 sm:grid-cols-2">
            <FormItem label="角色名称" prop="name" required>
              <Input v-model="form.name" :maxlength="40" placeholder="例如：集团管理员" />
            </FormItem>
            <FormItem label="角色编码" prop="code" required>
              <Input v-model="form.code" :maxlength="40" placeholder="例如：GROUP_ADMIN" />
            </FormItem>
            <FormItem label="数据范围" class="sm:col-span-2">
              <Select v-model="form.dataScope" :options="scopeOptions" />
            </FormItem>
            <FormItem v-if="form.dataScope !== 'tenant'" label="专属经营主体" class="sm:col-span-2">
              <Select v-model="form.legalEntityId" :options="entityOptions" :disabled="isStandalone" placeholder="租户级角色可跨主体授予" />
            </FormItem>
            <FormItem label="说明" class="sm:col-span-2">
              <Input v-model="form.description" :maxlength="120" placeholder="选填，说明角色适用岗位" />
            </FormItem>
            <FormItem label="状态" class="sm:col-span-2">
              <Switch v-model="form.status" active-value="active" inactive-value="suspended" active-text="正常" inactive-text="停用" />
            </FormItem>
          </div>
        </Form>

        <section class="border-t border-light pt-4">
          <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
            <h3 class="m-0 text-sm font-600">权限授权</h3>
            <div class="flex items-center gap-2 text-xs text-tertiary">
              <span>已选 {{ form.permissionCodes.length }} 项</span>
              <Button variant="link" size="sm" :disabled="saving" @click="form.permissionCodes = []">清空</Button>
            </div>
          </div>
          <Alert v-if="!permissions.length" type="info" show-icon title="权限目录为空，请确认后端种子数据已执行。" />
          <PermissionTree v-else v-model="form.permissionCodes" :nodes="permissions" :disabled="saving" />
        </section>
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
  "title": "角色管理",
  "permission": "role:view"
}
</router>
