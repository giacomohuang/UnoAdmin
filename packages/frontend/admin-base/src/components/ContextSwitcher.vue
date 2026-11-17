<script setup lang="ts">
/**
 * 顶部上下文切换器：租户决定采购与数据隔离边界，经营主体决定当前操作的公司。
 *
 * 切换会重新签发令牌，因此完成后整页刷新，避免各页面残留上一个上下文的数据。
 */
import { Select, type SelectModelValue } from '@mcistudio/unoui-vue/select'
import type { LegalEntityId, TenantId } from '@unoadmin/core'
import { computed, ref } from 'vue'

import { useAdminBaseContext } from '../composables/useAdminBaseContext'

const { session } = useAdminBaseContext()
const switching = ref(false)

const tenants = computed(() => session.state.tenants)
const entities = computed(() => session.state.legalEntities)
const currentTenant = computed(() => session.state.currentTenant)
const currentEntity = computed(() => session.state.currentLegalEntity)
const tenantModeLabel = computed(() => (currentTenant.value?.mode === 'group' ? '集团' : '单体'))

/** 租户/主体选项直接使用服务端下发的可切换范围。 */
const tenantOptions = computed(() => tenants.value.map((tenant) => ({
  label: `${tenant.name}（${tenant.mode === 'group' ? '集团' : '单体'}）`,
  value: tenant.id
})))
const entityOptions = computed(() => entities.value.map((entity) => ({
  label: `${entity.name}${entity.type === 'GROUP' ? ' · 总部' : ''}`,
  value: entity.id
})))

/** 切换租户后整页刷新，让所有列表重新按新边界查询。 */
async function changeTenant(value: SelectModelValue | undefined) {
  if (!value || switching.value) return
  switching.value = true
  try {
    await session.switchTenant(String(value) as TenantId)
    window.location.reload()
  } finally {
    switching.value = false
  }
}

/** 在租户内切换经营主体。 */
async function changeEntity(value: SelectModelValue | undefined) {
  if (!value || switching.value) return
  switching.value = true
  try {
    await session.switchLegalEntity(String(value) as LegalEntityId)
    window.location.reload()
  } finally {
    switching.value = false
  }
}
</script>

<template>
  <div class="flex min-w-0 items-center gap-1.5">
    <div class="hidden items-center gap-1.5 sm:flex">
      <span class="i-lucide:briefcase-business h-3.5 w-3.5 text-tertiary" aria-hidden="true" />
      <Select
        v-if="tenants.length > 1"
        :model-value="currentTenant?.id"
        :options="tenantOptions"
        size="sm"
        width="190px"
        :disabled="switching"
        aria-label="切换租户"
        @change="changeTenant"
      />
      <span v-else class="max-w-44 truncate text-sm font-500" :title="currentTenant?.name">{{ currentTenant?.name || '未选择租户' }}</span>
      <span class="rounded border border-light px-1 text-[11px] leading-4 text-tertiary">{{ tenantModeLabel }}</span>
    </div>

    <span class="mx-1 hidden h-4 w-px bg-tertiary/60 sm:block" aria-hidden="true" />

    <div class="flex min-w-0 items-center gap-1.5">
      <span class="i-lucide:building-2 h-3.5 w-3.5 shrink-0 text-tertiary" aria-hidden="true" />
      <Select
        v-if="entities.length > 1"
        :model-value="currentEntity?.id"
        :options="entityOptions"
        size="sm"
        width="200px"
        :disabled="switching"
        aria-label="切换经营主体"
        @change="changeEntity"
      />
      <span v-else class="max-w-44 truncate text-sm text-secondary" :title="currentEntity?.name">{{ currentEntity?.name || '未选择主体' }}</span>
    </div>
  </div>
</template>
