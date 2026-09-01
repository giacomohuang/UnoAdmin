<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import type { Permission } from '@unoadmin/core'
import { useAdminRuntime } from '@unoadmin/runtime'

const runtime = useAdminRuntime()
const permissions = ref<Permission[]>([])
const loading = ref(false)

interface FlatPermission extends Permission {
  depth: number
}

/** Flatten the adapter tree only for presentation; the source contract remains hierarchical. */
const rows = computed<FlatPermission[]>(() => {
  const result: FlatPermission[] = []
  const visit = (items: Permission[], depth: number) => {
    for (const item of items) {
      result.push({ ...item, depth })
      if (item.children) visit(item.children, depth + 1)
    }
  }
  visit(permissions.value, 0)
  return result
})

/** Load the full permission tree for the tree-shaped page. */
async function loadPermissions() {
  loading.value = true
  permissions.value = await runtime.services.base.permissions.list()
  loading.value = false
}

onMounted(loadPermissions)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>权限管理</h1>
        <p>查看模块、页面和操作权限的层级结构。</p>
      </div>
    </header>
    <div class="base-panel">
      <div class="base-panel__toolbar"><h2>权限目录</h2><span class="secondary-text">{{ rows.length }} 项权限</span></div>
      <div v-if="loading" class="base-empty">正在加载权限...</div>
      <div v-else class="base-tree">
        <div class="base-tree__row base-tree__row--header"><strong>名称</strong><strong>权限码</strong><strong>类型</strong></div>
        <div v-for="item in rows" :key="item.id" class="base-tree__row">
          <strong :style="{ paddingLeft: `${item.depth * 20}px` }">{{ item.name }}</strong>
          <span>{{ item.code }}</span>
          <span class="base-status base-status--muted">{{ item.type }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.secondary-text {
  color: #789095;
  font-size: 12px;
}

.base-tree__row--header {
  min-height: 42px;
  color: #71878b;
  background: #fbfdfd;
  font-size: 12px;
}
</style>

<router lang="json">
{
  "title": "权限管理",
  "permission": "permission:read"
}
</router>
