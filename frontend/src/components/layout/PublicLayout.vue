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

        <!-- 桌面端导航 -->
        <nav v-if="showNav" class="hidden items-center gap-1 lg:flex" :aria-label="siteName">
          <RouterLink
            v-for="link in internalLinks"
            :key="link.path"
            :to="link.path"
            class="flex h-8 items-center px-3 text-[13px] font-medium transition-colors"
            :class="
              isActive(link.path)
                ? 'text-primary-600 dark:text-primary-300'
                : 'text-gray-600 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'
            "
          >
            {{ t(link.label) }}
          </RouterLink>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-8 items-center gap-1.5 px-3 text-[13px] font-medium text-gray-600 transition-colors hover:text-gray-900 dark:text-dark-400 dark:hover:text-white"
          >
            <Icon name="book" size="sm" />
            {{ t('home.docs') }}
          </a>
        </nav>

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
      <nav
        v-if="showNav && mobileOpen"
        class="border-t border-gray-200 bg-white px-4 py-2 lg:hidden dark:border-dark-700 dark:bg-dark-900"
      >
        <RouterLink
          v-for="link in internalLinks"
          :key="`m-${link.path}`"
          :to="link.path"
          class="flex h-10 items-center text-control font-medium"
          :class="
            isActive(link.path)
              ? 'text-primary-600 dark:text-primary-300'
              : 'text-gray-600 dark:text-dark-400'
          "
          @click="mobileOpen = false"
        >
          {{ t(link.label) }}
        </RouterLink>
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="flex h-10 items-center gap-1.5 text-control font-medium text-gray-600 dark:text-dark-400"
        >
          <Icon name="book" size="sm" />
          {{ t('home.docs') }}
        </a>
        <RouterLink
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="btn btn-primary btn-sm my-2 w-full sm:hidden"
          @click="mobileOpen = false"
        >
          {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
        </RouterLink>
      </nav>
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
import { computed, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
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

const route = useRoute()
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

/** 模型广场入口受开关与「是否需要登录」约束 */
const modelPlazaVisible = computed(() => {
  const settings = appStore.cachedPublicSettings
  const enabled = settings?.model_plaza_enabled === true
  if (!enabled) return false
  return isAuthenticated.value || settings?.model_plaza_require_auth !== true
})

/** 公开页导航：首页 / 控制台 / 模型广场 / 关于（参考 new-api 的统一头部框架） */
const internalLinks = computed(() => {
  const links: Array<{ path: string; label: string }> = [
    { path: '/home', label: 'home.nav.home' },
    { path: dashboardPath.value, label: 'home.nav.console' },
    { path: '/about', label: 'home.nav.about' }
  ]
  // 模型广场受站点开关与「是否要求登录」约束，关闭时不显示该标签
  if (modelPlazaVisible.value) {
    links.splice(2, 0, { path: '/model-plaza', label: 'home.nav.plaza' })
  }
  return links
})

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(`${path}/`)
}
</script>
