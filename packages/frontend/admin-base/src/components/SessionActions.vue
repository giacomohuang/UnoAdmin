<script setup lang="ts">
/** 顶部账号区：展示当前账户与数据范围，并提供退出登录入口。 */
import { Dropdown } from '@mcistudio/unoui-vue/dropdown'
import { message } from '@mcistudio/unoui-vue/message'
import { Icon } from '@iconify/vue'
import { computed } from 'vue'
import { useRouter } from 'vue-router'

import { useAdminBaseContext } from '../composables/useAdminBaseContext'

interface SessionMenuItem {
  key: 'profile' | 'logout'
  label: string
  icon: string
  danger?: boolean
}

const { session } = useAdminBaseContext()
const router = useRouter()

const account = computed(() => session.state.account)
const initial = computed(() => account.value?.displayName?.slice(0, 1) ?? '?')
const scopeLabel = computed(() => {
  switch (session.state.dataScope) {
    case 'tenant':
      return account.value?.platformAdmin ? '平台全量数据' : '租户全量数据'
    case 'self':
      return '本人组织数据'
    default:
      return '授权主体数据'
  }
})
const roleLabel = computed(() => (account.value?.platformAdmin ? '平台运营管理员' : account.value?.roleNames.join('、') || '未分配角色'))

const items: SessionMenuItem[] = [
  { key: 'profile', label: '账户信息', icon: 'lucide:user-round' },
  { key: 'logout', label: '退出登录', icon: 'lucide:log-out', danger: true }
]

/** 退出前清空本地会话，再回到登录页。 */
async function handleSelect(item: unknown) {
  const selected = item as SessionMenuItem
  if (selected.key !== 'logout') return
  await session.signOut()
  message.success('已退出登录')
  await router.push('/signin')
}
</script>

<template>
  <Dropdown :items="items" align="right" :min-width="'15rem'" @select="handleSelect">
    <template #trigger="{ open }">
      <button
        type="button"
        class="flex h-9 items-center gap-2 rounded-lg border border-light bg-primary px-2 text-left transition-colors hover:bg-secondary"
        :aria-expanded="open"
        aria-label="账号菜单"
      >
        <span class="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-brand text-xs font-600 text-white">{{ initial }}</span>
        <span class="hidden min-w-0 flex-col sm:flex">
          <span class="max-w-32 truncate text-xs font-500 leading-4">{{ account?.displayName }}</span>
          <span class="max-w-32 truncate text-[11px] leading-3 text-tertiary">{{ scopeLabel }}</span>
        </span>
        <span class="i-lucide:chevron-down h-3.5 w-3.5 shrink-0 text-tertiary" aria-hidden="true" />
      </button>
    </template>

    <template #header>
      <div class="flex items-start gap-2.5 px-3 py-3">
        <span class="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-brand text-sm font-600 text-white">{{ initial }}</span>
        <div class="min-w-0">
          <strong class="block truncate text-sm">{{ account?.displayName }}</strong>
          <span class="block truncate text-xs text-tertiary">{{ account?.accountName }} · {{ account?.email || '未填写邮箱' }}</span>
          <span class="mt-0.5 block truncate text-[11px] text-tertiary">{{ roleLabel }}</span>
        </div>
      </div>
    </template>

    <template #item="{ item }">
      <div class="flex items-center gap-2 px-3 py-2" :class="(item as SessionMenuItem).danger ? 'text-red-600' : ''">
        <Icon :icon="(item as SessionMenuItem).icon" class="h-4 w-4" />
        <span class="text-sm">{{ (item as SessionMenuItem).label }}</span>
      </div>
    </template>
  </Dropdown>
</template>
