<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-dark-950">
    <div class="w-full max-w-[400px]">
      <!-- 单卡片：顶部 4px 品牌条 + 品牌区 + 表单 -->
      <div class="card overflow-hidden shadow-card">
        <div class="h-1 bg-primary-500 dark:bg-primary-400"></div>

        <div class="p-8">
          <!-- 品牌区：40px 方形 logo + 18px 产品名 + 12px muted 副标题 -->
          <div class="mb-6 flex items-center gap-3">
            <template v-if="settingsLoaded">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
                <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
              </div>
              <div class="min-w-0">
                <h1 class="truncate text-lg font-medium text-gray-900 dark:text-gray-100">
                  {{ siteName }}
                </h1>
                <p class="truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ siteSubtitle }}
                </p>
              </div>
            </template>
          </div>

          <slot />
        </div>
      </div>

      <!-- Footer Links -->
      <div class="mt-4 text-center text-[13px]">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-6 text-center text-xs text-gray-500 dark:text-dark-400">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
