<template>
  <div>
    <div class="section-heading">
      <h2 class="section-heading-title">{{ t('dashboard.quickActions') }}</h2>
    </div>

    <div class="mt-2 divide-y divide-gray-200/70 dark:divide-dark-700">
      <button
        @click="router.push('/keys')"
        class="group flex w-full items-center gap-3 px-1 py-3 text-left transition-colors duration-150 hover:bg-gray-50 dark:hover:bg-dark-800/60"
      >
        <Icon name="key" size="md" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
        <div class="min-w-0 flex-1">
          <p class="text-control font-medium text-gray-900 dark:text-white">{{ t('dashboard.createApiKey') }}</p>
          <span class="chip mt-1.5 whitespace-normal text-left">{{ t('dashboard.generateNewKey') }}</span>
        </div>
        <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
      </button>

      <button
        @click="router.push('/usage')"
        class="group flex w-full items-center gap-3 px-1 py-3 text-left transition-colors duration-150 hover:bg-gray-50 dark:hover:bg-dark-800/60"
      >
        <Icon name="chart" size="md" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
        <div class="min-w-0 flex-1">
          <p class="text-control font-medium text-gray-900 dark:text-white">{{ t('dashboard.viewUsage') }}</p>
          <span class="chip mt-1.5 whitespace-normal text-left">{{ t('dashboard.checkDetailedLogs') }}</span>
        </div>
        <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
      </button>

      <button
        v-if="canUseBatchImage"
        @click="router.push('/batch-image')"
        class="group flex w-full items-center gap-3 px-1 py-3 text-left transition-colors duration-150 hover:bg-gray-50 dark:hover:bg-dark-800/60"
      >
        <Icon name="sparkles" size="md" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
        <div class="min-w-0 flex-1">
          <p class="text-control font-medium text-gray-900 dark:text-white">{{ t('dashboard.batchImageAgent') }}</p>
          <span class="chip mt-1.5 whitespace-normal text-left">{{ t('dashboard.batchImageAgentDesc') }}</span>
        </div>
        <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
      </button>

      <button
        @click="router.push('/redeem')"
        class="group flex w-full items-center gap-3 px-1 py-3 text-left transition-colors duration-150 hover:bg-gray-50 dark:hover:bg-dark-800/60"
      >
        <Icon name="gift" size="md" class="shrink-0 text-gray-400 transition-colors group-hover:text-amber-500 dark:text-dark-500" />
        <div class="min-w-0 flex-1">
          <p class="text-control font-medium text-gray-900 dark:text-white">{{ t('dashboard.redeemCode') }}</p>
          <span class="chip mt-1.5 whitespace-normal text-left">{{ t('dashboard.addBalanceWithCode') }}</span>
        </div>
        <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-amber-500 dark:text-dark-500" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
