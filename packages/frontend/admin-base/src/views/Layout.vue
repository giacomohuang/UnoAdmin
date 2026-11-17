<script setup lang="ts">
/**
 * 管理端主布局：顶部上下文栏 + 100px 图标导航 + 180px 二级菜单 + 内容区。
 *
 * 结构与 UnoMap 管理端一致；菜单由模块声明并按权限过滤，租户与经营主体
 * 切换集中在顶部，避免页面各自维护上下文。
 */
import type { NavigationItem } from '@unoadmin/core'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'

import ContextSwitcher from '../components/ContextSwitcher.vue'
import NavMenu from '../components/NavMenu.vue'
import SessionActions from '../components/SessionActions.vue'
import { useAdminBaseContext } from '../composables/useAdminBaseContext'

const { branding, getNavigation, resolveNavigation } = useAdminBaseContext()
const route = useRoute()
const router = useRouter()

/** 当前视口是否仍适合常驻侧边栏。 */
const wideViewport = ref(true)
const menuVisible = ref(true)
const selectedIndex = ref(-1)

const navigation = computed<NavigationItem[]>(() => getNavigation())
const routeIndex = computed(() =>
  navigation.value.findIndex((item) => isItemActive(item) || item.children?.some((child) => isItemActive(child)))
)
const currentIndex = computed(() => (selectedIndex.value === -1 ? routeIndex.value : selectedIndex.value))
const currentItem = computed(() => (currentIndex.value < 0 ? undefined : navigation.value[currentIndex.value]))
const submenu = computed(() => currentItem.value?.children ?? [])
const hasSubmenu = computed(() => submenu.value.length > 0)
const showRail = computed(() => menuVisible.value && wideViewport.value)
const breadcrumb = computed(() => resolveNavigation(route.path))
const pageTitle = computed(() => (typeof route.meta.title === 'string' ? route.meta.title : ''))

/** 命中子路径也算激活，详情页不会丢失高亮。 */
function isItemActive(item: NavigationItem) {
  if (!item.path) return false
  return route.path === item.path || route.path.startsWith(`${item.path}/`)
}

/** 选中一级入口：叶子直接导航，分组只切换二级菜单。 */
function selectNavigation(item: NavigationItem, index: number) {
  selectedIndex.value = index
  if (!item.children?.length && item.path) void router.push(item.path)
  if (!wideViewport.value) menuVisible.value = false
}

/** 切换侧边栏显隐，窄屏下作为抽屉使用。 */
function toggleMenu() {
  menuVisible.value = !menuVisible.value
}

/** 记录视口宽度，窄屏默认收起侧边栏。 */
function syncViewport() {
  const wide = window.innerWidth >= 900
  if (wide !== wideViewport.value) {
    wideViewport.value = wide
    menuVisible.value = wide
  }
}

watch(
  () => route.path,
  () => {
    selectedIndex.value = -1
    if (!wideViewport.value) menuVisible.value = false
  }
)

onMounted(() => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
})

onBeforeUnmount(() => window.removeEventListener('resize', syncViewport))
</script>

<template>
  <div class="flex h-screen w-screen flex-col overflow-hidden bg-secondary text-primary">
    <header class="z-30 flex h-14 shrink-0 items-center gap-3 border-b border-light bg-primary px-3">
      <button
        type="button"
        class="grid h-8 w-8 shrink-0 place-items-center rounded-md border-0 bg-transparent text-secondary transition-colors hover:(bg-secondary text-primary)"
        :aria-label="menuVisible ? '收起菜单' : '展开菜单'"
        :title="menuVisible ? '收起菜单' : '展开菜单'"
        @click="toggleMenu"
      >
        <span :class="menuVisible ? 'i-lucide:panel-left-close' : 'i-lucide:panel-left-open'" class="h-5 w-5" aria-hidden="true" />
      </button>

      <div class="flex shrink-0 items-center gap-2 border-l border-light pl-3">
        <span class="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-700 text-white">{{ branding.productMark }}</span>
        <div class="hidden flex-col sm:flex">
          <strong class="text-sm leading-4">{{ branding.productName }}</strong>
          <span class="text-[11px] leading-3 text-tertiary">多租户 SaaS 管理框架</span>
        </div>
      </div>

      <nav class="ml-2 hidden min-w-0 items-center gap-1.5 text-sm text-tertiary md:flex" aria-label="面包屑">
        <span v-if="breadcrumb.parent" class="shrink-0">{{ breadcrumb.parent.title }}</span>
        <span v-if="breadcrumb.parent" class="i-lucide:chevron-right h-3.5 w-3.5 shrink-0 text-quaternary" aria-hidden="true" />
        <span class="truncate text-secondary">{{ breadcrumb.current?.title || pageTitle || '工作台' }}</span>
      </nav>

      <div class="ml-auto flex min-w-0 items-center gap-2">
        <ContextSwitcher />
        <SessionActions />
      </div>
    </header>

    <div class="relative flex min-h-0 flex-1">
      <!-- 一级图标导航：100px，保持与 UnoMap 相同的视觉节奏。 -->
      <aside
        v-show="showRail"
        class="z-20 w-25 shrink-0 overflow-y-auto border-r border-light bg-primary pt-2.5"
        aria-label="一级导航"
      >
        <NavMenu :items="navigation" :active-key="currentItem?.id" :label="branding.productName" :level="1" @select="selectNavigation" />
      </aside>

      <!-- 二级菜单：仅当一级项包含子项时出现。 -->
      <aside
        v-show="showRail && hasSubmenu"
        class="z-10 w-45 shrink-0 overflow-y-auto border-r border-light bg-primary py-2.5"
        aria-label="二级导航"
      >
        <div class="mb-1 px-3 text-[11px] font-600 uppercase tracking-wide text-tertiary">{{ currentItem?.title }}</div>
        <NavMenu :items="submenu" :label="currentItem?.title || '二级导航'" :level="2" @navigate="selectedIndex = -1" />
      </aside>

      <!-- 窄屏抽屉的遮罩，点击后收起侧边栏。 -->
      <div v-if="!wideViewport && menuVisible" class="absolute inset-0 z-10 bg-slate-900/30" aria-hidden="true" @click="menuVisible = false" />

      <section class="min-w-0 flex-1 overflow-auto">
        <RouterView />
      </section>
    </div>
  </div>
</template>
