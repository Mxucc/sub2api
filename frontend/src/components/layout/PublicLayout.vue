<template>
  <!--
    公开页外壳（首页 / 模型广场 等对外页面共用）
    - 顶部导航：站点 logo + 站名 + 导航（首页 / 模型广场 / 用量查询 / 文档）+ 语言 / 主题 / 登录·控制台
    - 内容区：默认 max-w-6xl；wide 用于模型广场这类需要更宽表格的页面（max-w-7xl）
    - 底部：站名 + 版权 + 版本
    控制台内部不使用本组件（控制台是 AppLayout：通栏顶栏 + 模块侧栏）。
  -->
  <div class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <header
      class="sticky top-0 z-40 border-b border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
    >
      <div
        class="mx-auto flex h-16 items-center justify-between gap-4 px-4 sm:px-6"
        :class="wide ? 'max-w-7xl' : 'max-w-6xl'"
      >
        <!-- 品牌 -->
        <RouterLink to="/" class="flex min-w-0 items-center gap-2.5" :title="siteName">
          <span class="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden">
            <img
              v-if="settingsLoaded"
              :src="siteLogo || '/logo.svg'"
              alt="Logo"
              class="h-full w-full object-contain"
            />
          </span>
          <span class="truncate text-[15px] font-medium text-gray-900 dark:text-gray-100">
            {{ siteName }}
          </span>
        </RouterLink>

        <!-- 桌面端导航（与 AppHeader 共用 PublicNavTabs，保证任何页面头部标签一致） -->
        <PublicNavTabs v-if="showNav" class="hidden lg:flex" />

        <!-- 右侧操作 -->
        <div class="flex shrink-0 items-center gap-1 sm:gap-2">
          <LocaleSwitcher />
          <button
            type="button"
            class="icon-btn"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>

          <RouterLink
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="btn btn-primary btn-sm hidden sm:inline-flex"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </RouterLink>

          <!-- 移动端菜单 -->
          <button
            v-if="showNav"
            type="button"
            class="icon-btn lg:hidden"
            :aria-label="t('common.toggleMenu')"
            :aria-expanded="mobileOpen"
            @click="mobileOpen = !mobileOpen"
          >
            <Icon :name="mobileOpen ? 'x' : 'menu'" size="sm" />
          </button>
        </div>
      </div>

      <!-- 移动端下拉导航 -->
      <div
        v-if="showNav && mobileOpen"
        class="border-t border-gray-200 bg-white pb-2 lg:hidden dark:border-dark-700 dark:bg-dark-900"
      >
        <PublicNavTabs vertical class="px-4 pt-1" />
        <div class="px-4">
          <RouterLink
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="btn btn-primary btn-sm mt-1 w-full"
            @click="mobileOpen = false"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </RouterLink>
        </div>
      </div>
    </header>

    <!-- 内容区 -->
    <main class="flex-1">
      <slot />
    </main>

    <footer v-if="showFooter" class="border-t border-gray-200 dark:border-dark-700">
      <div
        class="mx-auto flex flex-col items-center gap-2 px-4 py-6 text-caption text-gray-500 sm:flex-row sm:justify-between sm:px-6 dark:text-dark-400"
        :class="wide ? 'max-w-7xl' : 'max-w-6xl'"
      >
        <span class="[overflow-wrap:anywhere]">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </span>
        <span class="flex items-center gap-3">
          <RouterLink
            to="/key-usage"
            class="transition-colors hover:text-gray-900 dark:hover:text-white"
          >
            {{ t('home.nav.keyUsage') }}
          </RouterLink>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="transition-colors hover:text-gray-900 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <span v-if="version" class="text-gray-400 dark:text-dark-500">v{{ version }}</span>
        </span>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import PublicNavTabs from '@/components/layout/PublicNavTabs.vue'
import { sanitizeUrl } from '@/utils/url'
import { useTheme } from '@/composables/useTheme'
import { useAppStore, useAuthStore } from '@/stores'

withDefaults(
  defineProps<{
    /** 更宽的内容容器（模型广场等宽表格页面） */
    wide?: boolean
    /** 是否渲染底部版权条 */
    showFooter?: boolean
    /** 是否渲染顶部导航链接（登录/注册等页面可关闭） */
    showNav?: boolean
  }>(),
  {
    wide: false,
    showFooter: true,
    showNav: true
  }
)

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { isDark, toggleTheme } = useTheme()

const mobileOpen = ref(false)

const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || 'Sub2API')
const siteLogo = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.site_logo || '', {
    allowRelative: true,
    allowDataUrl: true
  })
)
const docUrl = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
)
const version = computed(() => appStore.cachedPublicSettings?.version || '')
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())


</script>
