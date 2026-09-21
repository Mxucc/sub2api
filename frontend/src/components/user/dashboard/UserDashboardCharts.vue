<template>
  <!-- 图表区：去卡片化，工具栏 + 发丝线分层 -->
  <section class="chart-frame">
    <!-- 工具栏：日期范围靠左，粒度 / 刷新靠右 -->
    <div class="chart-hair">
      <DateRangePicker
        :start-date="startDate"
        :end-date="endDate"
        @update:startDate="$emit('update:startDate', $event)"
        @update:endDate="$emit('update:endDate', $event)"
        @change="$emit('dateRangeChange', $event)"
      />
      <div class="flex flex-wrap items-center gap-2">
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
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="$emit('refresh')">
          <Icon name="refresh" size="sm" />
          {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <!-- 通栏趋势图 -->
    <div class="mt-4">
      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>

    <!-- 模型分布：标题行 + 圆环 / 台账表 -->
    <div class="section-heading mt-6">
      <h3 class="section-heading-title">{{ t('dashboard.modelDistribution') }}</h3>
      <span class="section-heading-meta">{{ dateRangeMeta }}</span>
    </div>
    <div class="relative mt-3 rounded border border-gray-200 p-3 dark:border-dark-700">
      <div
        v-if="loading"
        class="absolute inset-0 z-10 flex items-center justify-center rounded bg-white/50 backdrop-blur-sm dark:bg-dark-800/50"
      >
        <LoadingSpinner size="md" />
      </div>
      <div class="flex flex-col items-center gap-5 md:flex-row md:items-start md:gap-6">
        <div class="h-44 w-44 shrink-0 md:h-[220px] md:w-[220px]">
          <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
          <div v-else class="flex h-full items-center justify-center text-caption text-gray-500 dark:text-dark-400">{{ t('dashboard.noDataAvailable') }}</div>
        </div>
        <div class="min-w-0 w-full flex-1">
          <table class="w-full text-2xs">
            <thead>
              <tr class="border-b border-gray-200/70 text-gray-400 dark:border-dark-700 dark:text-dark-500">
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

// 模型分布标题旁的元信息：当前展示的数据区间
const dateRangeMeta = computed(() => (
  props.startDate === props.endDate ? props.startDate : `${props.startDate} — ${props.endDate}`
))

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

<style scoped>
/* 通栏趋势图借用 TokenUsageTrend 自带的画布，这里把它抬到 .chart-canvas-wide 的高度 */
:deep(.chart-canvas) {
  height: 320px;
}

@media (max-width: 639px) {
  :deep(.chart-canvas) {
    height: 220px;
  }
}
</style>
