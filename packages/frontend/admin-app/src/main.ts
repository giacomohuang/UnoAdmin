import { configureUnoUI } from '@mcistudio/unoui-vue/config'
import { createPinia } from 'pinia'
import { createApp } from 'vue'

import {
  ADMIN_BRANDING_KEY,
  ADMIN_MODULES_KEY,
  ADMIN_SERVICES_KEY,
  ADMIN_SESSION_KEY
} from '@unoadmin/core'

import './styles.css'
// UnoUI 组件依赖 UnoCSS 原子类，virtual:uno.css 必须在本文件内引入一次。
import 'virtual:uno.css'
import '@mcistudio/unoui-vue/style.css'

import App from './App.vue'
import router from './router'
import { createApiClient } from './adapters/api-client'
import { createHttpAuth } from './adapters/http-auth'
import { createHttpServices } from './adapters/http-services'
import { tokenStore } from './adapters/token-store'
import { modules } from './modules'
import { createAdminSession } from './app-session'
import { installRouterGuards } from './router/guards'

// 组件库只在本项目内使用简体中文，无需接入完整 i18n 运行时。
configureUnoUI({ locale: 'zh-CN' })

/** 所有 REST 请求共用一个客户端，令牌与租户/主体上下文随请求自动注入。 */
const apiClient = createApiClient({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1/',
  getAccessToken: () => tokenStore.getAccess(),
  getRefreshToken: () => tokenStore.getRefresh(),
  getTenantId: () => tokenStore.getTenant(),
  getLegalEntityId: () => tokenStore.getLegalEntity(),
  onTokens: (access, refresh) => tokenStore.setTokens(access, refresh),
  onAuthFailed: () => tokenStore.clear()
})
const authAdapter = createHttpAuth(apiClient, tokenStore)
const appServices = createHttpServices(apiClient)
const session = createAdminSession({ auth: authAdapter })
const branding = {
  productName: 'UnoAdmin',
  productMark: 'U'
}

const app = createApp(App)
app.use(createPinia())
app.provide(ADMIN_SESSION_KEY, session)
app.provide(ADMIN_BRANDING_KEY, branding)
app.provide(ADMIN_MODULES_KEY, modules)
app.provide(ADMIN_SERVICES_KEY, appServices)
app.use(router)

await session.initialize()
installRouterGuards(router, session)
app.mount('#app')
