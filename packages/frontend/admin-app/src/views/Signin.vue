<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useAppSession } from '../app-session'

const router = useRouter()
const session = useAppSession()

/** Restore the local demo session before navigating past the auth guard. */
async function enterWorkspace() {
  await session.signIn()
  await router.push('/dashboard')
}
</script>

<template>
  <main class="signin-page">
    <section class="signin-card">
      <div class="signin-mark">U</div>
      <span class="signin-kicker">UnoAdmin / DEMO</span>
      <h1>进入管理工作台</h1>
      <p>这是一个由基础管理 package 和示例业务 package 组成的 SPA 骨架。</p>
      <button class="base-button signin-submit" type="button" @click="enterWorkspace">进入演示</button>
      <small>当前使用本地 mock adapter，无需后端服务。</small>
    </section>
  </main>
</template>
<router lang="json">
{
  "layout": false
}
</router>
