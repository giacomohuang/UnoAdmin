<script setup lang="ts">
/**
 * 角色授权用的权限选择器：MillerColumns 逐级浏览 + 行内 Checkbox。
 *
 * 勾选父级会连带选中全部子权限；取消父级会取消整个子树；
 * 勾选子级前必须先勾选父级，避免出现「有子权限但父菜单不可见」的无效授权。
 */
import { Checkbox } from '@mcistudio/unoui-vue/checkbox'
import { MillerColumns, type MillerColumnsId } from '@mcistudio/unoui-vue/millercolumns'
import { Icon } from '@iconify/vue'
import type { Permission } from '@unoadmin/core'
import { computed, ref } from 'vue'

import {
  descendantIds,
  flattenPermissions,
  millerColumnsCheckboxClass,
  permissionIcon,
  toggleGrant,
  type PermissionNode
} from './tree'

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

const selection = ref<MillerColumnsId[]>([])
const flat = computed(() => flattenPermissions(props.nodes))

/** 角色存的是权限码，这里换算成节点 ID 参与父子级联判断。 */
const selectedIds = computed(() => {
  const codes = new Set(props.modelValue)
  return new Set(flat.value.filter((node) => codes.has(node.code)).map((node) => node.id))
})

/** 判断未选父节点下是否存在已选后代，用于显示半选状态。 */
function partial(node: PermissionNode) {
  const ids = [node.id, ...descendantIds(node.id, flat.value)]
  const count = ids.filter((id) => selectedIds.value.has(id)).length
  return count > 0 && count < ids.length
}

/** 缺少已选父级时禁用子节点，保证授权始终自根向下有效。 */
function disabledNode(node: PermissionNode) {
  return props.disabled || Boolean(node.parentId && !selectedIds.value.has(node.parentId))
}

/** 按父子级联规则切换授权，默认连带全部子权限。 */
function toggle(node: PermissionNode) {
  if (disabledNode(node)) return
  emit('update:modelValue', toggleGrant(node.id, props.modelValue, flat.value))
}
</script>

<template>
  <MillerColumns
    v-model="selection"
    :data-source="flat"
    id-key="id"
    parent-id-key="parentId"
    order-key="sort"
    height="20rem"
    min-height="20rem"
    width="100%"
    column-width="15rem"
    :show-info-panel="false"
    empty-text="该权限没有下级"
    no-data-text="暂无可授权权限"
    aria-label="权限授权"
  >
    <template #col-title="{ colIndex }">第 {{ colIndex + 1 }} 级</template>
    <template #item-left="{ item, active }">
      <span class="flex shrink-0 items-center" @click.stop @keydown.stop>
        <Checkbox
          :class="millerColumnsCheckboxClass(active)"
          :checked="selectedIds.has(item.id)"
          :indeterminate="partial(item)"
          :disabled="disabledNode(item)"
          :aria-label="item.name"
          @change="toggle(item)"
        />
      </span>
      <Icon
        :icon="permissionIcon(item)"
        class="h-4 w-4 shrink-0"
        :class="active ? 'text-white' : item.type === 'action' ? 'text-amber-600' : 'text-emerald-600'"
      />
      <span class="min-w-0 truncate" :title="item.name">{{ item.name }}</span>
    </template>
  </MillerColumns>
</template>
