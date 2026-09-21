<template>
  <div>
    <div class="section-heading">
      <h2 class="section-heading-title">{{ t('dashboard.recentUsage') }}</h2>
      <span class="section-heading-meta">{{ t('dashboard.last7Days') }}</span>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-12">
      <LoadingSpinner size="lg" />
    </div>
    <div v-else-if="data.length === 0" class="py-8">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>
    <template v-else>
      <div class="ledger mt-2">
        <div class="ledger-head">
          <span>{{ t('dashboard.model') }} · {{ t('usage.time') }}</span>
          <span class="ledger-num">{{ t('dashboard.tokens') }}</span>
          <span class="ledger-num-strong">{{ t('usage.cost') }}</span>
        </div>

        <div v-for="log in data" :key="log.id" class="ledger-row ledger-row-hover">
          <div class="min-w-0">
            <p class="truncate font-medium text-gray-900 dark:text-white" :title="log.model">{{ log.model }}</p>
            <p class="metric-note">{{ formatDateTime(log.created_at) }}</p>
          </div>
          <span class="ledger-num text-gray-600 dark:text-dark-400">
            {{ (log.input_tokens + log.output_tokens).toLocaleString() }}
          </span>
          <span class="ledger-num-strong">
            <span class="metric-accent" :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</span>
            <span class="font-normal text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / ${{ formatCost(log.total_cost) }}</span>
          </span>
        </div>
      </div>

      <router-link to="/usage" class="mt-3 flex items-center justify-center gap-2 py-3 text-caption font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
        {{ t('dashboard.viewAllUsage') }}
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)
</script>
