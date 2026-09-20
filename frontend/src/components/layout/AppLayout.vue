<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- 通栏顶栏（fixed inset-x-0 top-0 h-14 z-40） -->
    <AppHeader />

    <!-- 顶栏之下的模块导航侧栏（fixed left-0 top-14 bottom-0） -->
    <AppSidebar />

    <!-- 内容区：顶部让出 56px 顶栏，左侧让出侧栏宽度（240px / 折叠 64px） -->
    <main
      class="min-h-screen pt-14 transition-[padding] duration-200 ease-out"
      :class="[sidebarCollapsed ? 'lg:pl-16' : 'lg:pl-60']"
    >
      <!-- 面包屑条：仅当路由声明了 meta.breadcrumbs 时渲染（全站唯一面包屑来源） -->
      <div v-if="breadcrumbs.length" class="crumb-strip">
        <nav class="breadcrumb" :aria-label="t('common.breadcrumb')">
          <template v-for="(item, index) in breadcrumbs" :key="`${item.label}-${index}`">
            <router-link
              v-if="item.to && index < breadcrumbs.length - 1"
              :to="item.to"
              class="breadcrumb-link"
            >
              {{ resolveLabel(item.label) }}
            </router-link>
            <span v-else class="breadcrumb-current" aria-current="page">
              {{ resolveLabel(item.label) }}
            </span>
            <span v-if="index < breadcrumbs.length - 1" class="breadcrumb-separator" aria-hidden="true">/</span>
          </template>
        </nav>
      </div>

      <!-- 页面内容（各页面自带 PageHeader + 卡片/表格） -->
      <div class="px-5 py-4 lg:px-6 lg:py-5">
        <slot />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const route = useRoute()
const { t, te } = useI18n()

type BreadcrumbItem = { label: string; to?: string }


// 控制台面包屑来自路由 meta，缺失时整条不渲染。
// meta.breadcrumbs 里存的是 i18n key，因此统一用 resolveLabel 解析（非 key 原样输出）。
const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const items = route.meta?.breadcrumbs
  return Array.isArray(items) ? items : []
})

function resolveLabel(label: string): string {
  return te(label) ? t(label) : label
}

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
