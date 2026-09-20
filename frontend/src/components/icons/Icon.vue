<template>
  <component :is="iconComponent" :class="sizeClass" :stroke-width="strokeWidth" />
</template>

<script setup lang="ts">
/*
 * 图标组件：Lucide 线性图标（round cap / 1.5px 圆润线性）
 * - 对外 API 不变：name / size / strokeWidth，因此全站调用方无需改动
 * - 名称 → Lucide 组件映射见 ./iconMap.ts，新增图标时在该表补一行（类型会强制补全）
 * - 模板根节点必须是单个元素（不能有根级注释），否则 Vue 会退化为多根组件，
 *   调用方传入的 class 等属性无法自动透传到 svg 上
 */
import { computed } from 'vue'
import { fallbackIcon, iconMap, type IconName } from './iconMap'

const props = withDefaults(
  defineProps<{
    name: IconName
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
    strokeWidth?: number
  }>(),
  {
    size: 'md',
    strokeWidth: 1.5
  }
)

const iconComponent = computed(() => iconMap[props.name] ?? fallbackIcon)

const sizeClass = computed(
  () =>
    ({
      xs: 'h-3 w-3',
      sm: 'h-4 w-4',
      md: 'h-5 w-5',
      lg: 'h-6 w-6',
      xl: 'h-8 w-8'
    })[props.size]
)
</script>
