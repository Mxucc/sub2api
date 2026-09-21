<template>
  <!-- 图表区：去卡片化，工具栏 + 发丝线分层 -->
  <section class="chart-frame">
    <div class="chart-hair">
      <div class="ml-auto flex flex-wrap items-center gap-2">
        <DateRangePicker
          :start-date="startDate"
          :end-date="endDate"
          @update:startDate="$emit('update:startDate', $event)"
          @update:endDate="$emit('update:endDate', $event)"
          @change="$emit('dateRangeChange', $event)"
        />
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="$emit('refresh')">
          <Icon name="refresh" size="sm" />
          {{ t('common.refresh') }}
        </button>
        <div class="segmented" role="group" :aria-label="t('dashboard.granularity')">
          <button
            v-for="option in granularityOptions"
            :key="option.value"
            type="button"
            class="segmented-item"
            :aria-pressed="granularity === option.value"
            @click="selectGranularity(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="mt-4 grid grid-cols-1 gap-8 lg:grid-cols-2">
      <!-- 模型分布 -->
      <div class="relative">
        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/50 backdrop-blur-sm dark:bg-dark-800/50">
          <LoadingSpinner size="md" />
        </div>
        <h3 class="section-heading-title">{{ t('dashboard.modelDistribution') }}</h3>
        <div class="chart-canvas mt-3 flex h-auto flex-col items-center gap-4 sm:h-[260px] sm:flex-row sm:gap-6">
          <div class="h-48 w-48 shrink-0 sm:h-[260px] sm:w-[260px]">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="flex h-full items-center justify-center text-caption text-gray-500 dark:text-dark-400">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto sm:max-h-full">
            <table class="w-full text-2xs">
              <thead>
                <tr class="text-gray-500 dark:text-dark-400">
                  <th class="pb-2 text-left font-normal">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right font-normal">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right font-normal">{{ t('dashboard.tokens') }}</th>
                  <th class="pb-2 text-right font-normal">{{ t('dashboard.actual') }}</th>
                  <th class="pb-2 text-right font-normal">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model" class="border-t border-gray-200/70 dark:border-dark-700">
                  <td class="max-w-[100px] truncate py-1.5 font-medium text-gray-900 dark:text-white" :title="model.model">{{ model.model }}</td>
                  <td class="py-1.5 text-right tabular-nums text-gray-600 dark:text-dark-400">{{ formatNumber(model.requests) }}</td>
                  <td class="py-1.5 text-right tabular-nums text-gray-600 dark:text-dark-400">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="py-1.5 text-right tabular-nums metric-accent">${{ formatCost(model.actual_cost) }}</td>
                  <td class="py-1.5 text-right tabular-nums text-gray-400 dark:text-dark-500">${{ formatCost(model.cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Token 使用趋势 -->
      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
const emit = defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()

// 粒度选项以分段控件呈现，保持原有 v-model:granularity + granularityChange 事件契约
const granularityOptions = computed(() => [
  { value: 'day', label: t('dashboard.day') },
  { value: 'hour', label: t('dashboard.hour') },
])

function selectGranularity(value: string) {
  if (value === props.granularity) return
  emit('update:granularity', value)
  emit('granularityChange')
}

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#0052d9', '#366ef4', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#22d3ee']
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>
