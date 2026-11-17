<script setup lang="ts">
/** 登录页：账号密码登录，并按租户形态展示演示账号。 */
import { Alert } from '@mcistudio/unoui-vue/alert'
import { Button } from '@mcistudio/unoui-vue/button'
import { Form, FormItem, type FormRules } from '@mcistudio/unoui-vue/form'
import { Input } from '@mcistudio/unoui-vue/input'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import { useAppSession } from '../app-session'

const router = useRouter()
const session = useAppSession()

const form = ref({ username: 'chen.yu', password: 'admin123' })
const formRef = ref<InstanceType<typeof Form>>()
const error = ref('')
const loading = ref(false)
const rules: FormRules = {
  username: [{ required: true, message: '请输入账号' }],
  password: [{ required: true, message: '请输入密码' }]
}

/** 演示账号按客户形态区分，便于体验集团与单体两种租户。 */
const demoAccounts = [
  { username: 'lin.wei', label: '平台运营管理员', note: '可跨租户切换' },
  { username: 'chen.yu', label: '集团管理员', note: '汉唐控股 · 全部主体' },
  { username: 'zhao.ming', label: '主体管理员', note: '仅华东公司' },
  { username: 'li.na', label: '单体企业管理员', note: '明远科技' }
]

/** 使用账号密码建立真实会话，再进入受路由守卫保护的工作台。 */
async function enterWorkspace() {
  error.value = ''
  const valid = await formRef.value?.validate()
  if (!valid) return
  loading.value = true
  try {
    await session.signIn({ username: form.value.username.trim(), password: form.value.password })
    await router.push('/dashboard')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '登录失败，请检查后端服务与账号密码'
  } finally {
    loading.value = false
  }
}

/** 点击演示账号时填入表单，仍需用户主动提交。 */
function useDemo(username: string) {
  form.value.username = username
  form.value.password = 'admin123'
}
</script>

<template>
  <main class="grid min-h-screen w-full place-items-center bg-secondary p-4">
    <section class="grid w-full max-w-4xl overflow-hidden rounded-2xl border border-light bg-primary shadow-[0_24px_60px_rgb(15_23_42_/_0.14)] md:grid-cols-[1.1fr_1fr]">
      <!-- 左侧品牌区说明产品定位，避免登录页只有一张孤立表单。 -->
      <div class="hidden flex-col justify-between gap-10 bg-brand-700 p-10 text-white md:flex">
        <div class="flex items-center gap-3">
          <span class="grid h-10 w-10 place-items-center rounded-xl bg-white/15 text-lg font-700">U</span>
          <div>
            <strong class="block text-lg leading-tight">UnoAdmin</strong>
            <span class="text-xs text-white/70">多租户 SaaS 管理框架</span>
          </div>
        </div>
        <div class="space-y-5">
          <h1 class="m-0 text-2xl font-600 leading-9">一套框架，覆盖集团与单体两种客户形态</h1>
          <ul class="m-0 list-none space-y-3 p-0 text-sm text-white/80">
            <li class="flex items-start gap-2"><span class="i-lucide:building-2 mt-0.5 h-4 w-4 shrink-0" />集团统一采购：一个租户下多个经营主体，数据相互隔离</li>
            <li class="flex items-start gap-2"><span class="i-lucide:layers mt-0.5 h-4 w-4 shrink-0" />集团层面可统一查看，主体内独立管理</li>
            <li class="flex items-start gap-2"><span class="i-lucide:shield-check mt-0.5 h-4 w-4 shrink-0" />租户、主体、组织三级数据边界</li>
          </ul>
        </div>
        <p class="m-0 text-xs text-white/60">UnoUI · Vue 3 · Go</p>
      </div>

      <div class="flex flex-col gap-5 p-8 sm:p-10">
        <header class="space-y-1">
          <span class="text-xs font-600 uppercase tracking-widest text-brand">UnoAdmin</span>
          <h2 class="m-0 text-2xl font-600">进入管理工作台</h2>
          <p class="m-0 text-sm text-tertiary">使用后端账户登录，令牌与权限由服务实时下发。</p>
        </header>

        <Form ref="formRef" :model="form" :rules="rules" label-position="top" size="md" item-gap="14px" @submit.prevent="enterWorkspace">
          <FormItem label="账号" prop="username">
            <Input v-model="form.username" prefix-icon="i-lucide:user" autocomplete="username" placeholder="请输入账号" />
          </FormItem>
          <FormItem label="密码" prop="password">
            <Input v-model="form.password" type="password" show-password prefix-icon="i-lucide:lock" autocomplete="current-password" placeholder="请输入密码" @keyup.enter="enterWorkspace" />
          </FormItem>
        </Form>

        <Alert v-if="error" :title="error" type="error" show-icon />

        <Button size="lg" icon="i-lucide:log-in" :loading="loading" class="w-full" @click="enterWorkspace">
          {{ loading ? '登录中…' : '登录' }}
        </Button>

        <div class="space-y-2 border-t border-light pt-4">
          <span class="text-xs text-tertiary">演示账号（密码 admin123）</span>
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              v-for="item in demoAccounts"
              :key="item.username"
              type="button"
              class="flex flex-col items-start gap-0.5 rounded-lg border border-light px-3 py-2 text-left transition-colors hover:(border-brand bg-brand-50/60)"
              @click="useDemo(item.username)"
            >
              <span class="text-sm font-500">{{ item.label }}</span>
              <span class="text-xs text-tertiary">{{ item.username }} · {{ item.note }}</span>
            </button>
          </div>
        </div>
      </div>
    </section>
  </main>
</template>

<router lang="json">
{
  "layout": false
}
</router>
