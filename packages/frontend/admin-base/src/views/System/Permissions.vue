<script setup lang="ts">
/**
 * 权限管理：展示平台内置的权限目录。
 *
 * 权限由代码声明并按角色授予，页面只做只读展示，
 * 便于产品与实施人员核对「菜单 → 页面 → 操作」的授权粒度。
 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Button } from '@mcistudio/unoui-vue/button'
import { Input } from '@mcistudio/unoui-vue/input'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Table, type TableColumn } from '@mcistudio/unoui-vue/table'
import { Tag } from '@mcistudio/unoui-vue/tag'
import { Icon } from '@iconify/vue'
import type { Permission } from '@unoadmin/core'
import { computed, onMounted, ref } from 'vue'

import PageHeader from '../../components/PageHeader.vue'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { permissions: permissionService } = useAdminBaseServices()

const nodes = ref<Permission[]>([])
const loading = ref(false)
const ready = ref(false)
const error = ref('')
const keyword = ref('')

interface FlatPermission extends Permission {
  depth: number
}

/** 扁平化权限树，仅用于表格展示，源数据仍保持层级。 */
function flatten(items: Permission[], depth = 0): FlatPermission[] {
  return items.flatMap((item) => [{ ...item, depth }, ...flatten(item.children ?? [], depth + 1)])
}

const flat = computed(() => flatten(nodes.value))
const rows = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return flat.value
  return flat.value.filter((item) => `${item.name}${item.code}`.toLowerCase().includes(query))
})
const actionCount = computed(() => flat.value.filter((item) => item.type === 'action').length)
const pageCount = computed(() => flat.value.filter((item) => item.type === 'page').length)

const columns: TableColumn<FlatPermission>[] = [
  { key: 'name', title: '权限名称', minWidth: 240 },
  { key: 'code', title: '权限码', minWidth: 180 },
  { key: 'type', title: '类型', width: 90 },
  { key: 'scope', title: '生效层级', width: 110 }
]

const typeLabel: Record<Permission['type'], string> = { menu: '菜单', page: '页面', action: '操作' }
const scopeLabel: Record<Permission['scope'], { label: string; color: 'purple' | 'blue' | 'cyan' }> = {
  platform: { label: '平台', color: 'purple' },
  tenant: { label: '租户', color: 'blue' },
  entity: { label: '主体', color: 'cyan' }
}

/** 兼容旧数据中缺失 scope/type 的权限节点，避免渲染中断。 */
function scopeStyle(scope: Permission['scope'] | undefined) {
  return scopeLabel[scope as Permission['scope']] ?? { label: '租户', color: 'blue' as const }
}

/** 权限类型的中文文案，未知类型原样展示。 */
function typeText(type: Permission['type'] | undefined) {
  return type ? typeLabel[type] ?? type : '页面'
}

/** 权限类型对应的图标名，供 Iconify 组件动态渲染。 */
function typeIcon(type: Permission['type'] | undefined) {
  if (type === 'action') return 'lucide:mouse-pointer-click'
  if (type === 'menu') return 'lucide:folder-tree'
  return 'lucide:file-text'
}

/** 加载后端权限树。 */
async function load() {
  loading.value = true
  error.value = ''
  try {
    nodes.value = await permissionService.list()
    ready.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '权限目录加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>权限管理</template>
      <template #status>{{ flat.length }} 项权限 · {{ pageCount }} 个页面 · {{ actionCount }} 个操作</template>
      <template #description>权限由代码声明，角色负责组合授权；菜单、路由与按钮共用同一批权限码。</template>
      <template #actions>
        <div class="flex shrink-0 items-center gap-2">
          <Input v-model="keyword" prefix-icon="i-lucide:search" clearable placeholder="搜索权限名称或编码" class="w-56" />
          <Button variant="outline" size="icon-md" icon="i-lucide:refresh-cw" title="刷新" :loading="loading" @click="load" />
        </div>
      </template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <div class="mb-4 flex flex-wrap items-center gap-2 text-xs text-tertiary">
      <span>生效层级：</span>
      <Tag color="purple">平台</Tag><span>跨租户运营</span>
      <Tag color="blue">租户</Tag><span>租户内统管</span>
      <Tag color="cyan">主体</Tag><span>单个经营主体</span>
    </div>

    <Skeleton v-if="!ready && loading" variant="table" :rows="10" :columns="4" />
    <Table v-else :rows="rows" :columns="columns" row-key="id" min-width="760px" empty-text="没有匹配的权限" :aria-busy="loading">
      <template #cell-name="{ row }">
        <span class="flex min-w-0 items-center gap-1.5" :style="{ paddingLeft: `${row.depth * 18}px` }">
          <span v-if="row.depth > 0" class="h-3.5 w-3.5 shrink-0 border-l border-b border-dashed border-medium" aria-hidden="true" />
          <Icon :icon="typeIcon(row.type)" class="h-3.5 w-3.5 shrink-0 text-tertiary" />
          <span class="truncate font-medium">{{ row.name }}</span>
        </span>
      </template>
      <template #cell-code="{ row }"><span class="font-mono text-xs">{{ row.code }}</span></template>
      <template #cell-type="{ row }">
        <span class="text-sm text-secondary">{{ typeText(row.type) }}</span>
      </template>
      <template #cell-scope="{ row }">
        <Tag :color="scopeStyle(row.scope).color">{{ scopeStyle(row.scope).label }}</Tag>
      </template>
    </Table>
  </div>
</template>

<router lang="json">
{
  "title": "权限管理",
  "permission": "permission:view"
}
</router>
