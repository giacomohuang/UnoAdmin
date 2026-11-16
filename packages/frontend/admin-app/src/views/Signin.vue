<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import { useAppSession } from '../app-session'

const router = useRouter()
const session = useAppSession()

/** 默认填入种子账号，保留“进入演示”的一键体验，但登录请求走真实后端。 */
const username = ref('lin.wei')
const password = ref('admin123')
const error = ref('')
const loading = ref(false)

/** 使用账号密码建立真实会话，再进入受路由守卫保护的工作台。 */
async function enterWorkspace() {
  error.value = ''
  loading.value = true
  try {
    await session.signIn({ username: username.value.trim(), password: password.value })
    await router.push('/dashboard')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '登录失败，请检查后端服务与账号密码'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="signin-page">
    <section class="signin-card">
      <div class="signin-mark">U</div>
      <span class="signin-kicker">UnoAdmin / API</span>
      <h1>进入管理工作台</h1>
      <p>使用 Go 后端账户登录，令牌与权限由服务实时下发。</p>

      <label class="signin-field">
        <span>用户名</span>
        <input v-model="username" type="text" autocomplete="username" />
      </label>
      <label class="signin-field">
        <span>密码</span>
        <input v-model="password" type="password" autocomplete="current-password" @keyup.enter="enterWorkspace" />
      </label>

      <p v-if="error" class="signin-error" role="alert">{{ error }}</p>
      <button class="base-button signin-submit" type="button" :disabled="loading" @click="enterWorkspace">
        {{ loading ? '登录中…' : '进入演示' }}
      </button>
      <small>默认种子账号 lin.wei / admin123，可在 .env 中调整后端地址。</small>
    </section>
  </main>
</template>
<router lang="json">
{
  "layout": false
}
</router>
