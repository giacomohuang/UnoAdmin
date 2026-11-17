<script setup lang="ts">
/** 渲染一级图标导航或二级菜单，交互与 UnoMap 管理端保持一致。 */
import { Icon } from '@iconify/vue'
import type { NavigationItem } from '@unoadmin/core'
import { useRoute } from 'vue-router'

const props = defineProps<{
  level: 1 | 2
  items: NavigationItem[]
  activeKey?: string
  label: string
}>()

const emit = defineEmits<{
  navigate: []
  select: [item: NavigationItem, index: number]
}>()

const route = useRoute()

/** 子路径也视为当前入口激活，避免详情页丢失导航反馈。 */
function isActive(item: NavigationItem) {
  if (!item.path) return false
  return route.path === item.path || route.path.startsWith(`${item.path}/`)
}
</script>

<template>
  <nav v-if="props.level === 1" class="flex flex-col gap-1 px-1" :aria-label="props.label">
    <button
      v-for="(item, index) in props.items"
      :key="item.id"
      type="button"
      class="group flex w-full cursor-pointer flex-col items-center rounded-lg border-0 bg-transparent px-1 py-2 text-secondary transition-colors duration-200 hover:text-primary focus-visible:outline-2 focus-visible:outline-offset--2 focus-visible:outline-brand"
      :class="{ 'text-primary': props.activeKey === item.id }"
      :title="item.title"
      :aria-current="props.activeKey === item.id ? 'page' : undefined"
      :data-navigation-index="index"
      :data-navigation-key="item.id"
      @click="emit('select', item, index)"
    >
      <span
        class="mb-1 flex rounded-md p-2.5 transition-colors duration-150"
        :class="props.activeKey === item.id ? 'bg-brand text-white' : 'group-hover:bg-brand-400 group-hover:text-white'"
        aria-hidden="true"
      >
        <Icon :icon="item.icon || 'lucide:circle'" class="h-5 w-5" />
      </span>
      <span class="max-w-full truncate text-center text-xs leading-tight">{{ item.title }}</span>
    </button>
  </nav>

  <nav v-else class="flex flex-col gap-1 px-2" :aria-label="props.label">
    <RouterLink
      v-for="item in props.items"
      :key="item.id"
      :to="item.path || '/'"
      class="group flex h-9 items-center gap-2 rounded-md px-2 text-sm text-secondary no-underline transition-colors hover:(bg-brand-50 text-primary) focus-visible:outline-2 focus-visible:outline-offset--2 focus-visible:outline-brand"
      :class="isActive(item) ? 'bg-brand-50 font-500 text-brand-700' : ''"
      @click="emit('navigate')"
    >
      <span class="flex h-4 w-4 shrink-0 items-center justify-center" aria-hidden="true">
        <span
          class="h-1.5 w-1.5 rounded-full bg-quaternary transition-all group-hover:(h-2 w-2 bg-brand)"
          :class="isActive(item) ? 'h-2 w-2 bg-brand' : ''"
        />
      </span>
      <span class="min-w-0 flex-1 truncate">{{ item.title }}</span>
    </RouterLink>
  </nav>
</template>
