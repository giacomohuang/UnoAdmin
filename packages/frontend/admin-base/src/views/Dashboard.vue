<script setup lang="ts">
/** 工作台：汇总当前上下文规模，并说明多租户隔离边界。 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Skeleton } from '@mcistudio/unoui-vue/skeleton'
import { Tag } from '@mcistudio/unoui-vue/tag'
import { Icon } from '@iconify/vue'
import type { AdminStats } from '@unoadmin/core'
import { computed, onMounted, ref } from 'vue'

import PageHeader from '../components/PageHeader.vue'
import { useAdminBaseContext } from '../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../composables/useAdminBaseServices'

const { branding, session } = useAdminBaseContext()
const { stats: statsService } = useAdminBaseServices()

const stats = ref<AdminStats | null>(null)
const loading = ref(false)
const error = ref('')

const account = computed(() => session.state.account)
const tenant = computed(() => session.state.currentTenant)
const entity = computed(() => session.state.currentLegalEntity)
const isPlatform = computed(() => Boolean(account.value?.platformAdmin))
const scopeText = computed(() => {
  if (isPlatform.value) return '可跨租户查看与排障'
  if (session.state.dataScope === 'tenant') return '可查看租户下全部经营主体'
  if (session.state.dataScope === 'self') return '仅可查看本人所属组织'
  return '仅可查看被授权经营主体'
})

/** 拉取工作台指标；非平台账号只统计自身租户。 */
async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    stats.value = await statsService.get()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '指标加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)

/** 指标卡片统一渲染，避免四份重复模板。 */
const cards = computed(() => {
  const value = stats.value
  return [
    { key: 'tenants', label: '租户', value: value?.tenants ?? 0, icon: 'lucide:briefcase-business', tone: 'bg-brand-50 text-brand-700', visible: isPlatform.value },
    { key: 'legalEntities', label: '经营主体', value: value?.legalEntities ?? 0, icon: 'lucide:building-2', tone: 'bg-emerald-50 text-emerald-700', visible: true },
    { key: 'accounts', label: '账户', value: value?.accounts ?? 0, icon: 'lucide:users', tone: 'bg-amber-50 text-amber-700', visible: true },
    { key: 'roles', label: '角色', value: value?.roles ?? 0, icon: 'lucide:shield-check', tone: 'bg-violet-50 text-violet-700', visible: true }
  ].filter((card) => card.visible)
})
</script>

<template>
  <div class="min-w-0 p-4">
    <PageHeader :loading="loading">
      <template #title>工作台</template>
      <template #status>{{ tenant?.name || '未选择租户' }}</template>
      <template #description>当前登录 {{ account?.displayName }} · {{ scopeText }}</template>
    </PageHeader>

    <Alert v-if="error" :title="error" type="error" show-icon class="mb-4" />

    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <template v-if="!stats && loading">
        <Skeleton v-for="index in 4" :key="index" variant="grid" height="7rem" :columns="1" />
      </template>
      <article v-for="card in cards" v-else :key="card.key" class="flex items-start justify-between gap-3 rounded-xl border border-light bg-primary p-4">
        <div class="min-w-0">
          <span class="block text-xs text-tertiary">{{ card.label }}</span>
          <strong class="mt-1 block text-2xl font-600 leading-8">{{ card.value }}</strong>
        </div>
        <span class="grid h-9 w-9 shrink-0 place-items-center rounded-lg" :class="card.tone">
          <Icon :icon="card.icon" class="h-5 w-5" />
        </span>
      </article>
    </div>

    <div class="mt-4 grid gap-4 lg:grid-cols-[1.15fr_1fr]">
      <section class="rounded-xl border border-light bg-primary p-5">
        <h2 class="m-0 text-base font-600">当前上下文</h2>
        <p class="mb-4 mt-1 text-sm text-tertiary">租户决定采购与数据隔离边界，经营主体决定当前操作的公司。</p>
        <dl class="m-0 grid gap-3 sm:grid-cols-2">
          <div class="rounded-lg bg-secondary p-3">
            <dt class="text-xs text-tertiary">租户</dt>
            <dd class="mb-0 mt-1 flex items-center gap-2 text-sm font-500">
              <span class="truncate">{{ tenant?.name || '未选择' }}</span>
              <Tag :color="tenant?.mode === 'group' ? 'blue' : 'gray'" size="sm">{{ tenant?.mode === 'group' ? '集团' : '单体' }}</Tag>
            </dd>
            <dd class="mb-0 mt-0.5 text-xs text-tertiary">{{ tenant?.code }} · {{ tenant?.plan }}</dd>
          </div>
          <div class="rounded-lg bg-secondary p-3">
            <dt class="text-xs text-tertiary">经营主体</dt>
            <dd class="mb-0 mt-1 truncate text-sm font-500">{{ entity?.name || '未选择' }}</dd>
            <dd class="mb-0 mt-0.5 text-xs text-tertiary">{{ entity?.code }} · {{ entity?.type === 'GROUP' ? '集团总部' : '经营公司' }}</dd>
          </div>
          <div class="rounded-lg bg-secondary p-3">
            <dt class="text-xs text-tertiary">数据范围</dt>
            <dd class="mb-0 mt-1 text-sm font-500">{{ session.state.dataScope === 'tenant' ? '租户级' : session.state.dataScope === 'self' ? '本人组织' : '主体级' }}</dd>
            <dd class="mb-0 mt-0.5 text-xs text-tertiary">{{ scopeText }}</dd>
          </div>
          <div class="rounded-lg bg-secondary p-3">
            <dt class="text-xs text-tertiary">可切换主体</dt>
            <dd class="mb-0 mt-1 text-sm font-500">{{ session.state.legalEntities.length }} 个</dd>
            <dd class="mb-0 mt-0.5 text-xs text-tertiary">由账户授权范围与角色数据范围共同决定</dd>
          </div>
        </dl>
      </section>

      <section class="rounded-xl border border-light bg-primary p-5">
        <h2 class="m-0 text-base font-600">{{ branding.productName }} 能力边界</h2>
        <p class="mb-4 mt-1 text-sm text-tertiary">同一套代码同时服务集团统一采购与单体公司采购。</p>
        <ul class="m-0 grid list-none gap-3 p-0 text-sm">
          <li class="flex gap-2">
            <span class="i-lucide:layers mt-0.5 h-4 w-4 shrink-0 text-brand" aria-hidden="true" />
            <span><strong class="font-500">租户</strong>是计费与数据隔离的最外层；集团客户一个租户，单体客户也是一个租户。</span>
          </li>
          <li class="flex gap-2">
            <span class="i-lucide:building-2 mt-0.5 h-4 w-4 shrink-0 text-emerald-600" aria-hidden="true" />
            <span><strong class="font-500">经营主体</strong>是租户内的隔离单元；集团租户可有多个，单体租户只有一个。</span>
          </li>
          <li class="flex gap-2">
            <span class="i-lucide:network mt-0.5 h-4 w-4 shrink-0 text-amber-600" aria-hidden="true" />
            <span><strong class="font-500">组织架构</strong>挂在主体之下，账户按主体与组织授权，部门负责人只看本部门。</span>
          </li>
          <li class="flex gap-2">
            <span class="i-lucide:shield-check mt-0.5 h-4 w-4 shrink-0 text-violet-600" aria-hidden="true" />
            <span><strong class="font-500">角色</strong>携带数据范围与权限码，菜单、路由与按钮使用同一批权限判断。</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<router lang="json">
{
  "title": "工作台",
  "permission": "dashboard:view"
}
</router>
