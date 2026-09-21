<template>
  <!--
    企业控制台页面头部（腾讯云 / TDesign 控制台样式）
    - 标题 + 描述 → 右侧操作区，底部 1px 分隔线
    - 面包屑由 AppLayout 的全局面包屑条统一提供（来自路由 meta.breadcrumbs），
      因此这里默认不渲染面包屑；仅在显式传入 `breadcrumbs` 时才在页头内联展示
      （适合抽屉/二级工作台这类没有全局面包屑条的页面）。
  -->
  <div class="page-header-bar">
    <div class="page-header-main">
      <nav v-if="items.length" class="breadcrumb" :aria-label="t('common.breadcrumb')">
        <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
          <RouterLink v-if="item.to && index < items.length - 1" :to="item.to" class="breadcrumb-link">
            {{ resolveLabel(item.label) }}
          </RouterLink>
          <span v-else class="breadcrumb-current" aria-current="page">{{ resolveLabel(item.label) }}</span>
          <span v-if="index < items.length - 1" class="breadcrumb-separator" aria-hidden="true">/</span>
        </template>
      </nav>

      <div class="page-header-row">
        <div class="min-w-0">
          <span v-if="index" class="page-header-index">{{ index }}</span>
          <h1 class="page-title">{{ title }}</h1>
          <p v-if="description" class="page-header-copy">{{ description }}</p>
        </div>
        <div v-if="$slots.actions" class="page-header-actions">
          <slot name="actions" />
        </div>
      </div>
    </div>

    <div v-if="$slots.tabs" class="page-header-tabs">
      <slot name="tabs" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'

type BreadcrumbItem = { label: string; to?: string }

const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    /** editorial 编号（如 "01"），页面可显式传入，缺省不渲染 */
    index?: string
    /** 页头内联面包屑（可选）；labels 可以是 i18n key，也可以是已翻译文本 */
    breadcrumbs?: BreadcrumbItem[]
  }>(),
  {
    description: '',
    index: '',
    breadcrumbs: undefined
  }
)

const { t, te } = useI18n()

const items = computed<BreadcrumbItem[]>(() => props.breadcrumbs ?? [])

function resolveLabel(label: string): string {
  return te(label) ? t(label) : label
}
</script>
