import { fileURLToPath, URL } from 'node:url'

import { routesGeneratorPlugin } from '@unoadmin/vite-plugins/routes-generator'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import UnoCSS from 'unocss/vite'
import { defineConfig } from 'vite'

export const routeExcludePatterns = ['**/components', 'Layout.vue'] as const

const workspaceRoot = fileURLToPath(new URL('../../..', import.meta.url))
const adminBaseViews = fileURLToPath(new URL('../admin-base/src/views', import.meta.url))
const appViews = fileURLToPath(new URL('./src/views', import.meta.url))

/** 配置单一 SPA 入口，并把各启用 package 的页面合并进同一张路由表。 */
export default defineConfig({
  // 独立缓存目录避免与旧版本依赖预构建产物混用，配置调整后可干净重建。
  cacheDir: 'node_modules/.vite-unoadmin-v2',
  optimizeDeps: {
    /**
     * 页面大量使用异步路由组件，默认只扫描 index.html 会让 UnoUI 子路径在首次进入
     * 某个页面时才被发现，触发依赖重优化与整页刷新。把全部源码作为扫描入口，
     * 可以在冷启动时一次性覆盖所有懒加载链路。
     */
    entries: ['index.html', 'src/**/*.{vue,ts}', '../admin-base/src/**/*.{vue,ts}'],
    include: [
      '@mcistudio/unoui-vue/alert',
      '@mcistudio/unoui-vue/button',
      '@mcistudio/unoui-vue/checkbox',
      '@mcistudio/unoui-vue/config',
      '@mcistudio/unoui-vue/drawer',
      '@mcistudio/unoui-vue/dropdown',
      '@mcistudio/unoui-vue/form',
      '@mcistudio/unoui-vue/input',
      '@mcistudio/unoui-vue/message',
      '@mcistudio/unoui-vue/millercolumns',
      '@mcistudio/unoui-vue/pagination',
      '@mcistudio/unoui-vue/popconfirm',
      '@mcistudio/unoui-vue/select',
      '@mcistudio/unoui-vue/skeleton',
      '@mcistudio/unoui-vue/switch',
      '@mcistudio/unoui-vue/table',
      '@mcistudio/unoui-vue/tag',
      '@mcistudio/unoui-vue/tooltip',
      '@mcistudio/unoui-vue/uno'
    ]
  },
  plugins: [
    // UnoCSS 必须先于 Vue 插件，保证 SFC 内的原子类在编译阶段即可收集。
    UnoCSS(),
    vue(),
    vueJsx(),
    routesGeneratorPlugin({
      layoutDir: fileURLToPath(new URL('../admin-base/src/views/Layout.vue', import.meta.url)),
      // 保持构建期生成结果与路由契约测试一致。
      exclude: routeExcludePatterns,
      routeRoots: [
        { directory: adminBaseViews, basePath: '' },
        { namePrefix: 'app', directory: appViews, basePath: '' }
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
      // UnoUI 与 admin-base 位于工作区中，需要放行仓库根目录的 /@fs 读取。
      allow: [workspaceRoot]
    }
  }
})
