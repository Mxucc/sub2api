<template>
  <!--
    公开页导航标签：首页 / 控制台 / 模型广场 / 关于
    单一来源：公开页外壳（PublicLayout）与控制台外壳（AppHeader / AppSidebar 移动抽屉）共用同一份定义，
    保证任何页面（含 /dashboard、/admin/*）头部标签都一致。
  -->
  <nav
    :class="vertical ? 'flex flex-col' : 'flex items-center gap-1'"
    :aria-label="t('home.nav.label')"
  >
    <RouterLink
      v-for="link in links"
      :key="link.path"
      :to="link.to"
      class="font-medium transition-colors"
      :class="[
        vertical ? 'flex h-10 items-center px-3 text-control' : 'flex h-8 items-center px-3 text-[13px]',
        isActive(link.match)
          ? 'text-primary-600 dark:text-primary-300'
          : 'text-gray-600 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'
      ]"
    >
      {{ t(link.label) }}
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore } from '@/stores'

const props = withDefaults(
  defineProps<{
    /** 竖排（移动端抽屉用） */
    vertical?: boolean
    /** 控制台内打开模型广场时带上 ?embedded=1，保持控制台框架不跳出 */
    embeddedPlaza?: boolean
  }>(),
  {
    vertical: false,
    embeddedPlaza: false
  }
)

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

/** 模型广场受站点开关与「是否需要登录」约束 */
const plazaVisible = computed(() => {
  const settings = appStore.cachedPublicSettings
  if (settings?.model_plaza_enabled !== true) return false
  return authStore.isAuthenticated || settings?.model_plaza_require_auth !== true
})

const consolePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))

interface NavLink {
  to: string | { path: string; query: Record<string, string> }
  path: string
  label: string
  /** 命中前缀，用于高亮 */
  match: string[]
}

const links = computed<NavLink[]>(() => {
  const list: NavLink[] = [
    { to: '/home', path: '/home', label: 'home.nav.home', match: ['/home', '/'] },
    {
      to: consolePath.value,
      path: consolePath.value,
      label: 'home.nav.console',
      match: ['/dashboard', '/admin']
    }
  ]

  if (plazaVisible.value) {
    list.push({
      to: props.embeddedPlaza
        ? { path: '/model-plaza', query: { embedded: '1' } }
        : '/model-plaza',
      path: '/model-plaza',
      label: 'home.nav.plaza',
      match: ['/model-plaza']
    })
  }

  list.push({ to: '/about', path: '/about', label: 'home.nav.about', match: ['/about'] })
  return list
})

function isActive(prefixes: string[]): boolean {
  return prefixes.some((p) => route.path === p || route.path.startsWith(`${p}/`))
}
</script>
