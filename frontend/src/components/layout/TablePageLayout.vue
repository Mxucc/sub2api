<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile }">
    <!--
      固定区域：页头
      标题/描述（#header）与操作按钮（#actions）同一行，右侧对齐；
      页头底部 1px 分隔线由本组件统一提供，因此 TablePageLayout 内不再出现
      「页头 → 操作行 → 筛选行」三段各自成行的散乱结构。
    -->
    <div v-if="$slots.header || $slots.actions" class="layout-section-fixed layout-page-head">
      <div v-if="$slots.header" class="layout-page-head-main">
        <slot name="header" />
      </div>
      <div v-if="$slots.actions" class="layout-page-head-actions">
        <slot name="actions" />
      </div>
    </div>

    <!-- 滚动区域：筛选 + 表格 + 分页收在同一张卡片内（筛选=卡片头，分页=卡片尾） -->
    <div class="layout-section-scrollable">
      <div class="surface table-scroll-container">
        <div v-if="$slots.filters" class="table-filter-slot">
          <slot name="filters" />
        </div>

        <slot name="table" />

        <div v-if="$slots.pagination" class="card-footer table-footer-slot">
          <slot name="pagination" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex flex-col gap-3;
  height: calc(100vh - 56px - 2.5rem); /* 减去 h-14 通栏顶栏 + lg:py-5 的上下 padding */
}

.layout-section-fixed {
  @apply flex-shrink-0;
}

/* ===== 页头行：左侧标题/描述，右侧操作按钮 ===== */
.layout-page-head {
  @apply flex items-start justify-between gap-4;
  @apply border-b border-gray-200 pb-3 dark:border-dark-700;
}

.layout-page-head-main {
  @apply min-w-0 flex-1;
}

.layout-page-head-actions {
  @apply flex shrink-0 items-center gap-2;
}

/* PageHeader 自带 mb-4/pb-4/border-b，在表格页由外层行统一提供间距与分隔线 */
.layout-page-head :deep(.page-header-bar) {
  @apply mb-0 border-b-0 pb-0;
}

.layout-section-scrollable {
  @apply flex-1 min-h-0 flex flex-col;
}

/* 表格卡片容器 - 增强版表体滚动方案（外观由 .surface 提供） */
.table-scroll-container {
  /* 可见溢出：筛选行的下拉/气泡不被卡片裁剪（全站直角，无需裁剪圆角） */
  @apply flex h-full flex-col overflow-visible;
}

/* 卡片头：筛选栏（内部 .filter-bar 自带 px-4 py-3 + 底部 1px 分隔线） */
.table-filter-slot {
  @apply flex-shrink-0;
}

/* 卡片尾：分页（.card-footer 提供分隔线与底色） */
.table-footer-slot {
  @apply flex-shrink-0;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 min-h-0 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  @apply bg-gray-50 dark:bg-dark-800;
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply px-3 py-2.5 text-left text-[13px] font-medium text-gray-700 dark:text-dark-400 border-b border-gray-200 dark:border-dark-700;
  @apply bg-gray-50 dark:bg-dark-800;
}

.table-scroll-container :deep(td) {
  @apply px-3 py-2.5 text-[13px] text-gray-700 dark:text-dark-300 border-b border-gray-200/70 dark:border-dark-700;
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto overflow-visible border-none shadow-none bg-transparent;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

/* 移动端卡片外观被移除，筛选栏与分页各自成块，避免出现悬空的 1px 线 */
.table-page-layout.mobile-mode .table-filter-slot,
.table-page-layout.mobile-mode .table-footer-slot {
  @apply border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900;
}

.table-page-layout.mobile-mode .table-footer-slot {
  @apply border-t-0;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}
</style>
