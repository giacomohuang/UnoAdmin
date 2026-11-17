/**
 * 管理端 UnoCSS 配置。
 *
 * presetUnoUI 同时提供 Wind4 预设、语义色（bg-primary/text-secondary 等）、
 * Iconify 图标与 UnoUI 组件所需的自定义规则，因此这里不再重复声明主题。
 */
import { readFileSync, readdirSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'

import { presetUnoUI } from '@mcistudio/unoui-vue/uno'
import { defineConfig, transformerDirectives, transformerVariantGroup } from 'unocss'

const require = createRequire(import.meta.url)
const unoUIDirectory = resolve(dirname(require.resolve('@mcistudio/unoui-vue/package.json')), 'dist')

/** 读取 UnoUI 产物，保证组件内部的动态类名同样被扫描到。 */
const unoUIContent = readdirSync(unoUIDirectory, { encoding: 'utf8', recursive: true })
  .filter((file) => file.endsWith('.js'))
  .map((file) => readFileSync(resolve(unoUIDirectory, file), 'utf8'))
  .join('\n')

export default defineConfig({
  presets: [presetUnoUI()],
  transformers: [transformerDirectives(), transformerVariantGroup()],
  content: {
    inline: [unoUIContent],
    filesystem: ['src/**/*.{vue,ts}', '../admin-base/src/**/*.{vue,ts}'],
    pipeline: {
      include: [/\.(vue|svelte|[jt]sx|mdx?|html)($|\?)/, 'src/**/*.{js,ts}', '../admin-base/src/**/*.{vue,ts}'],
      exclude: ['uno.config.ts']
    }
  }
})
