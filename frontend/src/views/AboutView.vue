<template>
  <!-- 关于页：与首页 / 模型广场 / 登录共用同一个公开页外壳（PublicLayout） -->
  <PublicLayout>
    <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:py-10">
      <!-- editorial 页头 -->
      <div class="page-header-bar">
        <div class="page-header-main">
          <div class="page-header-row">
            <div class="min-w-0">
              <span class="page-header-index">{{ siteName }} / ABOUT</span>
              <h1 class="page-title">{{ t('home.about.title') }}</h1>
              <p class="page-header-copy">{{ lead }}</p>
              <div class="page-header-meta">
                <span v-if="version">{{ t('home.about.versionLabel') }} v{{ version }}</span>
                <span v-if="siteSubtitle">{{ siteSubtitle }}</span>
              </div>
            </div>
            <div class="page-header-actions">
              <RouterLink :to="consolePath" class="btn btn-primary btn-sm">
                {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
              </RouterLink>
              <RouterLink to="/home" class="btn btn-secondary btn-sm">
                {{ t('home.about.backHome') }}
              </RouterLink>
            </div>
          </div>
        </div>
      </div>

      <!-- 三段信息卡 -->
      <div class="grid grid-cols-1 gap-3.5 lg:grid-cols-3">
        <div class="metric-card">
          <span class="metric-label">{{ t('home.about.accessTitle') }}</span>
          <p class="metric-note">{{ t('home.about.accessFlow') }}</p>
          <div class="metric-foot">
            <span>{{ t('home.about.accessApiBase') }}</span>
            <strong class="truncate font-mono">{{ apiBaseUrl || t('home.about.notConfigured') }}</strong>
          </div>
        </div>

        <div class="metric-card">
          <span class="metric-label">{{ t('home.about.contactTitle') }}</span>
          <p class="metric-note">{{ siteSubtitle || siteName }}</p>
          <div class="metric-foot">
            <span>{{ t('home.about.contactTitle') }}</span>
            <strong class="truncate">
              <a
                v-if="contactUrl"
                :href="contactUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary-600 transition-colors hover:underline dark:text-primary-300"
              >
                {{ contactText }}
              </a>
              <span v-else>{{ contactText || t('home.about.notConfigured') }}</span>
            </strong>
          </div>
        </div>

        <div class="metric-card">
          <span class="metric-label">{{ t('home.about.docsTitle') }}</span>
          <p class="metric-note">
            {{ t('home.nav.plaza') }} · {{ t('home.nav.keyUsage') }}
          </p>
          <div class="metric-foot">
            <span>{{ t('home.about.consoleTitle') }}</span>
            <strong>
              <RouterLink
                :to="consolePath"
                class="text-primary-600 transition-colors hover:underline dark:text-primary-300"
              >
                {{ t('home.goToDashboard') }}
              </RouterLink>
            </strong>
          </div>
        </div>
      </div>

      <!-- 快速入口（发丝线行） -->
      <div class="mt-6">
        <div class="section-heading">
          <h2 class="section-heading-title">{{ t('home.about.docsTitle') }}</h2>
          <span v-if="docUrl" class="section-heading-meta">
            <a :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-primary-600">
              {{ t('home.docs') }} →
            </a>
          </span>
        </div>
        <div class="ledger">
          <RouterLink
            v-if="plazaVisible"
            to="/model-plaza"
            class="ledger-row ledger-row-hover"
          >
            <span class="text-gray-900 dark:text-gray-100">{{ t('home.nav.plaza') }}</span>
            <span class="ledger-num text-gray-500 dark:text-dark-400">/model-plaza</span>
            <span class="ledger-num-strong text-primary-600 dark:text-primary-300">→</span>
          </RouterLink>
          <RouterLink to="/key-usage" class="ledger-row ledger-row-hover">
            <span class="text-gray-900 dark:text-gray-100">{{ t('home.nav.keyUsage') }}</span>
            <span class="ledger-num text-gray-500 dark:text-dark-400">/key-usage</span>
            <span class="ledger-num-strong text-primary-600 dark:text-primary-300">→</span>
          </RouterLink>
          <RouterLink :to="consolePath" class="ledger-row ledger-row-hover">
            <span class="text-gray-900 dark:text-gray-100">{{ t('home.nav.console') }}</span>
            <span class="ledger-num text-gray-500 dark:text-dark-400">{{ consolePath }}</span>
            <span class="ledger-num-strong text-primary-600 dark:text-primary-300">→</span>
          </RouterLink>
        </div>
      </div>
    </div>
  </PublicLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PublicLayout from '@/components/layout/PublicLayout.vue'

import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || 'Sub2API')
const siteSubtitle = computed(() => settings.value?.site_subtitle || '')
const version = computed(() => settings.value?.version || '')
const apiBaseUrl = computed(() => settings.value?.api_base_url || '')

const lead = computed(() => t('home.about.lead', { site: siteName.value }))

const contactRaw = computed(() => (settings.value?.contact_info || '').trim())
const contactUrl = computed(() => {
  if (!contactRaw.value) return ''
  return sanitizeUrl(contactRaw.value, { allowRelative: false, allowDataUrl: false }) || ''
})
const contactText = computed(() => {
  if (!contactRaw.value) return ''
  return contactUrl.value ? contactUrl.value.replace(/^https?:\/\//, '') : contactRaw.value
})

const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const consolePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const plazaVisible = computed(() => {
  const s = settings.value
  if (s?.model_plaza_enabled !== true) return false
  return isAuthenticated.value || s?.model_plaza_require_auth !== true
})

onMounted(() => {
  void appStore.fetchPublicSettings()
})
</script>
