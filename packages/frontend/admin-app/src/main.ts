import { createApp } from 'vue'
import { createPinia } from 'pinia'

import {
  ADMIN_BRANDING_KEY,
  ADMIN_MODULES_KEY,
  ADMIN_SERVICES_KEY,
  ADMIN_SESSION_KEY,
} from '@unoadmin/core'
import '@unoadmin/ui/styles.css'
import '@unoadmin/admin-base/styles.css'
import './styles.css'

import App from './App.vue'
import router from './router'
import { authAdapter, appServices } from './adapters/mock'
import { modules } from './modules'
import { createAdminSession } from './app-session'
import { installRouterGuards } from './router/guards'

const session = createAdminSession({ auth: authAdapter })
const branding = {
  productName: 'UnoAdmin',
  productMark: 'U',
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
