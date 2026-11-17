<template>
  <header class="mb-4 flex flex-wrap items-end justify-between gap-4">
    <div class="min-w-0 flex-1">
      <div class="flex min-w-0 items-end gap-3">
        <h1 class="min-w-0 max-w-full truncate text-2xl font-500 text-primary"><slot name="title" /></h1>
        <span v-if="$slots.status" class="inline-flex min-h-5 items-center whitespace-nowrap text-sm font-600 text-tertiary" role="status" :aria-busy="loading">
          <span v-if="loading" class="i-lucide:loader-circle h-4 w-4 animate-spin text-brand" aria-label="加载中" />
          <slot v-else name="status" />
        </span>
      </div>
      <p v-if="$slots.description" class="mb-0 mt-1 text-xs leading-5 text-tertiary"><slot name="description" /></p>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center justify-end gap-2"><slot name="actions" /></div>
  </header>
</template>

<script setup lang="ts">
/** 页面标题栏的加载态契约；状态文案由调用方在请求成功后提供。 */
interface PageHeaderProps {
  loading?: boolean
}

withDefaults(defineProps<PageHeaderProps>(), {
  loading: false
})
</script>
