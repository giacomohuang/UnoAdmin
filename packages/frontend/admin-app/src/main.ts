import { createApp } from 'vue'
import { createPinia } from 'pinia'

import '@unoadmin/ui/styles.css'
import '@unoadmin/admin-base/styles.css'
import './styles.css'

import App from './App.vue'
import router from './router'
import { authAdapter, appServices } from './adapters/mock'
import { modules } from './modules'
import { createAdminRuntime, provideAdminRuntime } from '@unoadmin/runtime'
import { installRouterGuards } from './router/guards'

const runtime = createAdminRuntime({
  auth: authAdapter,
  branding: {
    productName: 'UnoAdmin',
    productMark: 'U',
  },
  modules,
  services: appServices,
})

const app = createApp(App)
app.use(createPinia())
provideAdminRuntime(app, runtime)
app.use(router)

await runtime.initialize()
installRouterGuards(router, runtime)
app.mount('#app')
