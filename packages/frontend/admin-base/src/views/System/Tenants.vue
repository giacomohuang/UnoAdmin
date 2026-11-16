<script setup lang="ts">
import { onMounted, ref } from 'vue'

import type { Tenant } from '@unoadmin/core'

import { useAdminBaseContext } from '../../composables/useAdminBaseContext'
import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const { session } = useAdminBaseContext()
const services = useAdminBaseServices()
const tenants = ref<Tenant[]>([])
const loading = ref(false)

/** Load tenant records from the same service port used by the Layout selector. */
async function loadTenants() {
  loading.value = true
  tenants.value = (await services.tenants.list({ page: 1, pageSize: 50 })).items
  loading.value = false
}

onMounted(loadTenants)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>租户管理</h1>
        <p>查看租户状态，并验证当前租户上下文的切换能力。</p>
      </div>
      <button class="base-button" type="button">新建租户</button>
    </header>
    <div class="base-panel">
      <div class="base-panel__toolbar"><h2>租户列表</h2><span class="secondary-text">平台范围</span></div>
      <div v-if="loading" class="base-empty">正在加载租户...</div>
      <table v-else class="base-table">
        <thead><tr><th>租户名称</th><th>编码</th><th>当前状态</th><th>当前上下文</th></tr></thead>
        <tbody>
          <tr v-for="tenant in tenants" :key="tenant.id">
            <td><strong>{{ tenant.name }}</strong></td>
            <td>{{ tenant.code }}</td>
            <td><span class="base-status" :class="{ 'base-status--muted': tenant.status !== 'active' }">{{ tenant.status === 'active' ? '正常' : '暂停' }}</span></td>
            <td>{{ session.state.currentTenant?.id === tenant.id ? '当前租户' : '可切换' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.secondary-text {
  color: #789095;
  font-size: 12px;
}
</style>

<router lang="json">
{
  "title": "租户管理",
  "permission": "tenant:read"
}
</router>
