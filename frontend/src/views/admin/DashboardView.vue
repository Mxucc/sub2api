<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <div class="stagger space-y-4">
          <!-- Editorial masthead -->
          <header class="page-header-bar">
            <div class="page-header-main">
              <div class="page-header-row flex-wrap">
                <div class="min-w-0">
                  <span class="page-header-index">ADMIN / OVERVIEW</span>
                  <h1 class="page-title">{{ t('admin.dashboard.title') }}</h1>
                  <p class="page-header-copy">{{ t('admin.dashboard.description') }}</p>
                  <div class="page-header-meta">
                    <span>{{ startDate }} → {{ endDate }}</span>
                    <span aria-hidden="true">·</span>
                    <span>{{
                      granularity === 'day' ? t('admin.dashboard.day') : t('admin.dashboard.hour')
                    }}</span>
                    <template v-if="stats.stats_updated_at">
                      <span aria-hidden="true">·</span>
                      <span>{{ t('common.updatedAt', { date: stats.stats_updated_at }) }}</span>
                    </template>
                  </div>
                </div>
                <div class="page-header-actions flex-wrap justify-end">
                  <DateRangePicker
                    v-model:start-date="startDate"
                    v-model:end-date="endDate"
                    @change="onDateRangeChange"
                  />
                  <div
                    class="segmented"
                    role="group"
                    :aria-label="t('admin.dashboard.granularity')"
                  >
                    <button
                      v-for="option in granularityOptions"
                      :key="option.value"
                      type="button"
                      class="segmented-item"
                      :class="{ 'is-active': granularity === option.value }"
                      :aria-pressed="granularity === option.value"
                      @click="selectGranularity(option.value)"
                    >
                      {{ option.label }}
                    </button>
                  </div>
                  <button
                    type="button"
                    class="btn btn-secondary"
                    :disabled="chartsLoading"
                    @click="loadDashboardStats"
                  >
                    {{ t('common.refresh') }}
                  </button>
                </div>
              </div>
            </div>
          </header>

          <!-- Metrics: headline row (the page's single inverted hero card) -->
          <div class="metric-grid-lead stagger">
            <!-- Today Tokens -->
            <div class="metric-card metric-lead metric-card-hero">
              <div class="flex items-center gap-2">
                <Icon name="cube" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.todayTokens') }}</span>
              </div>
              <div class="metric-value">{{ formatTokens(stats.today_tokens) }}</div>
              <p class="metric-note">
                <span :title="t('admin.dashboard.actual')">${{ formatCost(stats.today_actual_cost) }}</span>
                <span class="opacity-60"> / </span>
                <span :title="t('admin.dashboard.accountCost')"
                  >${{ formatCost(stats.today_account_cost) }}</span
                >
                <span class="opacity-60"> / </span>
                <span :title="t('admin.dashboard.standard')">${{ formatCost(stats.today_cost) }}</span>
              </p>
              <div class="metric-foot">
                <span>{{ t('admin.dashboard.todayRequests') }}</span>
                <span
                  ><strong>{{ stats.today_requests }}</strong></span
                >
              </div>
            </div>

            <!-- Today Requests -->
            <div class="metric-card metric-lead">
              <div class="flex items-center gap-2">
                <Icon name="chart" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.todayRequests') }}</span>
              </div>
              <div class="metric-value">{{ stats.today_requests }}</div>
              <p class="metric-note">
                {{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}
              </p>
              <div class="metric-foot">
                <span>{{ t('admin.dashboard.todayTokens') }}</span>
                <span
                  ><strong>{{ formatTokens(stats.today_tokens) }}</strong></span
                >
              </div>
            </div>

            <!-- Total Tokens -->
            <div class="metric-card metric-lead">
              <div class="flex items-center gap-2">
                <Icon name="database" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.totalTokens') }}</span>
              </div>
              <div class="metric-value">{{ formatTokens(stats.total_tokens) }}</div>
              <p class="metric-note">
                <span class="metric-accent" :title="t('admin.dashboard.actual')"
                  >${{ formatCost(stats.total_actual_cost) }}</span
                >
                <span class="text-gray-400 dark:text-dark-500"> / </span>
                <span class="text-amber-500 dark:text-amber-400" :title="t('admin.dashboard.accountCost')"
                  >${{ formatCost(stats.total_account_cost) }}</span
                >
                <span class="text-gray-400 dark:text-dark-500"> / </span>
                <span class="text-gray-400 dark:text-dark-500" :title="t('admin.dashboard.standard')"
                  >${{ formatCost(stats.total_cost) }}</span
                >
              </p>
              <div class="metric-foot">
                <span>{{ t('admin.dashboard.totalRequests') }}</span>
                <span
                  ><strong>{{ formatNumber(stats.total_requests) }}</strong></span
                >
              </div>
            </div>
          </div>

          <!-- Metrics: inventory and throughput -->
          <div class="metric-grid stagger">
            <!-- Total API Keys -->
            <div class="metric-card">
              <div class="flex items-center gap-2">
                <Icon name="key" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.apiKeys') }}</span>
              </div>
              <div class="metric-value">{{ stats.total_api_keys }}</div>
              <div class="metric-foot">
                <span>{{ t('common.active') }}</span>
                <span
                  ><strong>{{ stats.active_api_keys }}</strong></span
                >
              </div>
            </div>

            <!-- Service Accounts -->
            <div class="metric-card">
              <div class="flex items-center gap-2">
                <Icon name="server" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.accounts') }}</span>
              </div>
              <div class="metric-value">{{ stats.total_accounts }}</div>
              <p v-if="stats.error_accounts > 0" class="metric-note text-red-500">
                {{ stats.error_accounts }} {{ t('common.error') }}
              </p>
              <div class="metric-foot">
                <span>{{ t('common.active') }}</span>
                <span
                  ><strong>{{ stats.normal_accounts }}</strong></span
                >
              </div>
            </div>

            <!-- Users -->
            <div class="metric-card">
              <div class="flex items-center gap-2">
                <Icon name="userPlus" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.users') }}</span>
              </div>
              <div class="metric-value">
                <span class="metric-accent">+{{ stats.today_new_users }}</span>
              </div>
              <div class="metric-foot">
                <span>{{ t('common.total') }}</span>
                <span
                  ><strong>{{ formatNumber(stats.total_users) }}</strong></span
                >
              </div>
            </div>

            <!-- Performance (RPM / TPM) -->
            <div class="metric-card">
              <div class="flex items-center gap-2">
                <Icon name="bolt" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.performance') }}</span>
              </div>
              <div class="metric-value">
                {{ formatTokens(stats.rpm) }}
                <span class="metric-unit">RPM</span>
              </div>
              <p class="metric-note">
                {{ formatTokens(stats.tpm) }}
                <span class="metric-unit">TPM</span>
              </p>
            </div>

            <!-- Avg Response Time -->
            <div class="metric-card">
              <div class="flex items-center gap-2">
                <Icon name="clock" size="sm" class="metric-accent" :stroke-width="2" />
                <span class="metric-label">{{ t('admin.dashboard.avgResponse') }}</span>
              </div>
              <div class="metric-value">{{ formatDuration(stats.average_duration_ms) }}</div>
              <p class="metric-note">
                {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
              </p>
            </div>
          </div>

          <!-- Quick Actions -->
          <section>
            <div class="section-heading">
              <h2 class="section-heading-title">{{ t('admin.dashboard.quickActions') }}</h2>
            </div>
            <div class="grid grid-cols-1 gap-x-8 md:grid-cols-2">
              <button
                v-if="canUseBatchImage"
                type="button"
                class="group flex w-full items-center gap-3 border-b border-gray-200 py-3.5 text-left transition-colors dark:border-dark-700"
                @click="router.push('/batch-image')"
              >
                <Icon
                  name="sparkles"
                  size="sm"
                  class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-600 dark:group-hover:text-primary-300"
                  :stroke-width="2"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-control text-gray-900 dark:text-white">
                    {{ t('admin.dashboard.batchImage') }}
                  </span>
                  <span class="block text-2xs text-gray-500 dark:text-dark-400">
                    {{ t('admin.dashboard.batchImageDesc') }}
                  </span>
                </span>
                <Icon
                  name="chevronRight"
                  size="sm"
                  class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-600 dark:group-hover:text-primary-300"
                />
              </button>
              <button
                type="button"
                class="group flex w-full items-center gap-3 border-b border-gray-200 py-3.5 text-left transition-colors dark:border-dark-700"
                @click="router.push('/admin/groups')"
              >
                <Icon
                  name="grid"
                  size="sm"
                  class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-600 dark:group-hover:text-primary-300"
                  :stroke-width="2"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-control text-gray-900 dark:text-white">
                    {{ t('admin.dashboard.groupPricing') }}
                  </span>
                  <span class="block text-2xs text-gray-500 dark:text-dark-400">
                    {{ t('admin.dashboard.groupPricingDesc') }}
                  </span>
                </span>
                <Icon
                  name="chevronRight"
                  size="sm"
                  class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-600 dark:group-hover:text-primary-300"
                />
              </button>
            </div>
          </section>

          <!-- Charts -->
          <section class="space-y-4">
            <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <div class="chart-frame">
                <ModelDistributionChart
                  :model-stats="modelStats"
                  :enable-ranking-view="true"
                  :ranking-items="rankingItems"
                  :ranking-total-actual-cost="rankingTotalActualCost"
                  :ranking-total-requests="rankingTotalRequests"
                  :ranking-total-tokens="rankingTotalTokens"
                  :loading="chartsLoading"
                  :ranking-loading="rankingLoading"
                  :ranking-error="rankingError"
                  :start-date="startDate"
                  :end-date="endDate"
                  @ranking-click="goToUserUsage"
                />
              </div>
              <div class="chart-frame">
                <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
              </div>
            </div>

            <!-- User Usage Trend (full width) -->
            <div class="chart-frame">
              <div class="chart-hair">
                <div>
                  <h3 class="section-heading-title">
                    {{ t('admin.dashboard.recentUsage') }} (Top 12)
                  </h3>
                  <span class="section-heading-meta">
                    {{
                      granularity === 'day' ? t('admin.dashboard.day') : t('admin.dashboard.hour')
                    }}
                  </span>
                </div>
              </div>
              <div class="chart-canvas mt-3">
                <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                  <LoadingSpinner size="md" />
                </div>
                <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
                <div
                  v-else
                  class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
                >
                  {{ t('admin.dashboard.noDataAvailable') }}
                </div>
              </div>
            </div>
          </section>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity switch options for the segmented control
const granularityOptions = computed<{ value: 'day' | 'hour'; label: string }[]>(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#0052d9',
    '#366ef4',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#003cab',
    '#f97316',
    '#6366f1',
    '#22d3ee',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const selectGranularity = (value: 'day' | 'hour') => {
  if (granularity.value === value) return
  granularity.value = value
  void loadChartData()
}

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
})
</script>

<style scoped>
</style>
