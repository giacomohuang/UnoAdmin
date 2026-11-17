<script setup lang="ts">
/**
 * 权限授权树：按父子级联维护勾选状态。
 *
 * 勾选父节点会同时勾选全部后代；勾选子节点会补齐祖先，
 * 避免出现「有子权限但父菜单不可见」的无效授权。
 */
import { Checkbox } from '@mcistudio/unoui-vue/checkbox'
import type { Permission } from '@unoadmin/core'
import { computed, ref } from 'vue'

defineOptions({ name: 'PermissionTree' })

const props = withDefaults(
  defineProps<{
    nodes: Permission[]
    modelValue: string[]
    disabled?: boolean
  }>(),
  { disabled: false }
)

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

interface FlatNode extends Permission {
  depth: number
  parentId: string
}

const expanded = ref<Set<string>>(new Set(['permission-dashboard', 'permission-system']))

/** 扁平化权限树，供祖先/后代计算与渲染使用。 */
const flat = computed(() => {
  const result: FlatNode[] = []
  const visit = (items: Permission[], depth: number, parentId: string) => {
    for (const item of items) {
      result.push({ ...item, depth, parentId })
      if (item.children?.length) visit(item.children, depth + 1, item.id)
    }
  }
  visit(props.nodes, 0, '')
  return result
})

const selected = computed(() => {
  // 角色存储的是权限码而不是节点 ID，这里统一换算成 ID 再参与级联计算。
  const codes = new Set(props.modelValue)
  return new Set(flat.value.filter((node) => codes.has(node.code)).map((node) => node.id))
})

/** 收集某个节点的全部后代 ID。 */
function descendants(id: string): string[] {
  return flat.value.filter((node) => node.parentId === id).flatMap((node) => [node.id, ...descendants(node.id)])
}

/** 收集某个节点的全部祖先 ID。 */
function ancestors(id: string): string[] {
  const result: string[] = []
  let current = flat.value.find((node) => node.id === id)
  while (current?.parentId) {
    result.push(current.parentId)
    current = flat.value.find((node) => node.id === current?.parentId)
  }
  return result
}

/** 判断节点是否处于半选状态。 */
function indeterminate(node: Permission) {
  const ids = [node.id, ...descendants(node.id)]
  const count = ids.filter((id) => selected.value.has(id)).length
  return count > 0 && count < ids.length
}

/** 按父子规则切换授权，并把 ID 集合还原为权限码。 */
function toggle(node: Permission, checked: boolean) {
  const next = new Set(selected.value)
  const targets = [node.id, ...descendants(node.id)]
  if (checked) {
    targets.forEach((id) => next.add(id))
    ancestors(node.id).forEach((id) => next.add(id))
  } else {
    targets.forEach((id) => next.delete(id))
  }
  const codes = flat.value.filter((item) => next.has(item.id)).map((item) => item.code)
  emit('update:modelValue', codes)
}

/** 切换节点展开状态。 */
function toggleExpand(node: Permission) {
  const next = new Set(expanded.value)
  if (next.has(node.id)) next.delete(node.id)
  else next.add(node.id)
  expanded.value = next
}

const typeLabel: Record<Permission['type'], string> = { menu: '菜单', page: '页面', action: '操作' }
</script>

<template>
  <div class="rounded-lg border border-light">
    <ul class="m-0 max-h-80 list-none overflow-auto p-2">
      <li v-for="node in props.nodes" :key="node.id">
        <div class="flex min-h-8 items-center gap-1.5 rounded px-1 hover:bg-secondary">
          <button
            v-if="node.children?.length"
            type="button"
            class="grid h-5 w-5 shrink-0 place-items-center rounded border-0 bg-transparent text-tertiary"
            :aria-label="expanded.has(node.id) ? '收起' : '展开'"
            @click="toggleExpand(node)"
          >
            <span :class="expanded.has(node.id) ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'" class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
          <span v-else class="h-5 w-5 shrink-0" aria-hidden="true" />

          <Checkbox
            :checked="selected.has(node.id)"
            :indeterminate="indeterminate(node)"
            :disabled="props.disabled"
            @change="toggle(node, $event)"
          />
          <span class="min-w-0 flex-1 truncate text-sm">{{ node.name }}</span>
          <span class="shrink-0 font-mono text-[11px] text-tertiary">{{ node.code }}</span>
          <span class="shrink-0 rounded bg-secondary px-1 text-[11px] text-tertiary">{{ typeLabel[node.type] }}</span>
        </div>

        <PermissionTree
          v-if="expanded.has(node.id) && node.children?.length"
          class="ml-5 border-l border-light pl-1"
          :nodes="node.children"
          :model-value="props.modelValue"
          :disabled="props.disabled"
          @update:model-value="emit('update:modelValue', $event)"
        />
      </li>
    </ul>
  </div>
</template>
