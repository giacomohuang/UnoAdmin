<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { useDemoServices } from '../composables/useDemoServices'
import type { DemoRecord } from '@unoadmin/domain-demo'

const services = useDemoServices()
const records = ref<DemoRecord[]>([])
const loading = ref(false)

/** Load the demo record list through its package-owned service contract. */
async function loadRecords() {
  loading.value = true
  records.value = (await services.listRecords({ page: 1, pageSize: 20 })).items
  loading.value = false
}

onMounted(loadRecords)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>项目记录</h1>
        <p>用于验证独立业务 package 的列表页、服务和自动生成路由。</p>
      </div>
      <button class="base-button" type="button">新建项目</button>
    </header>
    <div class="base-panel">
      <div class="base-panel__toolbar"><h2>项目列表</h2><span class="secondary-text">{{ records.length }} 条记录</span></div>
      <div v-if="loading" class="base-empty">正在加载项目...</div>
      <table v-else class="base-table">
        <thead><tr><th>项目名称</th><th>负责人</th><th>状态</th><th>最近更新</th></tr></thead>
        <tbody>
          <tr v-for="record in records" :key="record.id">
            <td><strong>{{ record.name }}</strong></td>
            <td>{{ record.owner }}</td>
            <td><span class="base-status" :class="{ 'base-status--muted': record.status !== 'active' }">{{ record.status === 'active' ? '进行中' : '已暂停' }}</span></td>
            <td>{{ record.updatedAt }}</td>
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
  "title": "项目记录",
  "permission": "demo:read"
}
</router>
