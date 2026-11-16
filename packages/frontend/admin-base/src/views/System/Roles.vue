<script setup lang="ts">
import { onMounted, ref } from 'vue'

import type { Role } from '@unoadmin/core'

import { useAdminRuntime } from '../../runtime'

const runtime = useAdminRuntime()
const roles = ref<Role[]>([])
const loading = ref(false)

/** Read roles from the adapter to validate a package-owned list page. */
async function loadRoles() {
  loading.value = true
  roles.value = (await runtime.services.base.roles.list({ page: 1, pageSize: 50 })).items
  loading.value = false
}

onMounted(loadRoles)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>角色管理</h1>
        <p>通过角色聚合权限，控制不同岗位的管理范围。</p>
      </div>
      <button class="base-button" type="button">新建角色</button>
    </header>
    <div class="base-panel">
      <div class="base-panel__toolbar"><h2>角色列表</h2><span class="secondary-text">{{ roles.length }} 个角色</span></div>
      <div v-if="loading" class="base-empty">正在加载角色...</div>
      <table v-else class="base-table">
        <thead><tr><th>角色</th><th>说明</th><th>权限数</th><th>成员数</th><th>状态</th></tr></thead>
        <tbody>
          <tr v-for="role in roles" :key="role.id">
            <td><strong>{{ role.name }}</strong></td>
            <td>{{ role.description }}</td>
            <td>{{ role.permissionCodes.length }}</td>
            <td>{{ role.userCount }}</td>
            <td><span class="base-status">正常</span></td>
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
  "title": "角色管理",
  "permission": "role:read"
}
</router>
