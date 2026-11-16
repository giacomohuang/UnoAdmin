import { fileURLToPath, URL } from 'node:url'

import { routesGeneratorPlugin } from '@unoadmin/vite-plugins/routes-generator'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import { defineConfig } from 'vite'

// import { routeExcludePatterns } from './src/router/route-generator.config.ts'

// export { routeExcludePatterns } from './src/router/route-generator.config.ts'

export const routeExcludePatterns = ['**/components', 'Layout.vue'] as const

const workspaceRoot = fileURLToPath(new URL('../../..', import.meta.url))
const adminBaseViews = fileURLToPath(new URL('../admin-base/src/views', import.meta.url))
const demoViews = fileURLToPath(new URL('../domain-demo/src/views', import.meta.url))
const appViews = fileURLToPath(new URL('./src/views', import.meta.url))

/** Configure one SPA entry and merge route pages from every enabled package. */
export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    routesGeneratorPlugin({
      layoutDir: fileURLToPath(new URL('../admin-base/src/views/Layout.vue', import.meta.url)),
      // Keep build-time generation aligned with the route contract tests.
      exclude: routeExcludePatterns,
      routeRoots: [
        { moduleId: 'admin-base', directory: adminBaseViews, basePath: '' },
        { moduleId: 'demo', directory: demoViews, basePath: '/demo' },
        { moduleId: 'app', directory: appViews, basePath: '' }
      ]
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    fs: {
      allow: [workspaceRoot]
    }
  }
})
