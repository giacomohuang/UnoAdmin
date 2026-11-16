<script setup lang="ts">
import { onMounted, ref } from 'vue'

import type { User } from '@unoadmin/core'

import { useAdminBaseServices } from '../../composables/useAdminBaseServices'

const services = useAdminBaseServices()
const users = ref<User[]>([])
const search = ref('')
const loading = ref(false)
const errorMessage = ref('')

/** Load a bounded first page for the base package demonstration. */
async function loadUsers() {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await services.users.list({ page: 1, pageSize: 20, search: search.value })
    users.value = result.items
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '用户加载失败'
  } finally {
    loading.value = false
  }
}

/** Toggle status through the injected service so the page has a real write path. */
async function toggleStatus(user: User) {
  const nextStatus = user.status === 'active' ? 'disabled' : 'active'
  await services.users.updateStatus(user.id, nextStatus)
  user.status = nextStatus
}

onMounted(loadUsers)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>用户管理</h1>
        <p>管理登录账户、状态和已分配角色。</p>
      </div>
    </header>

    <div class="base-panel">
      <div class="base-panel__toolbar">
        <h2>账户列表</h2>
        <form class="list-filter" @submit.prevent="loadUsers">
          <input v-model="search" class="base-input" type="search" placeholder="搜索用户名或邮箱" aria-label="搜索用户">
          <button class="base-button" type="submit">查询</button>
        </form>
      </div>
      <p v-if="errorMessage" class="list-error" role="alert">{{ errorMessage }}</p>
      <div v-if="loading" class="base-empty">正在加载用户...</div>
      <div v-else-if="!users.length" class="base-empty">暂无用户</div>
      <table v-else class="base-table">
        <thead>
          <tr>
            <th>用户</th>
            <th>邮箱</th>
            <th>角色</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td><strong>{{ user.displayName }}</strong><br><span class="secondary-text">{{ user.username }}</span></td>
            <td>{{ user.email }}</td>
            <td>{{ user.roleNames.join('、') }}</td>
            <td><span class="base-status" :class="{ 'base-status--muted': user.status !== 'active' }">{{ user.status === 'active' ? '启用' : '停用' }}</span></td>
            <td><button class="base-button base-button--quiet" type="button" @click="toggleStatus(user)">{{ user.status === 'active' ? '停用' : '启用' }}</button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.list-filter {
  display: flex;
  gap: 8px;
}

.secondary-text {
  color: #789095;
  font-size: 12px;
}

.list-error {
  margin: 0;
  padding: 12px 20px;
  color: #b42318;
  background: #fff5f3;
  font-size: 13px;
}

@media (max-width: 560px) {
  .list-filter {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>

<router lang="json">
{
  "title": "用户管理",
  "permission": "user:read"
}
</router>
