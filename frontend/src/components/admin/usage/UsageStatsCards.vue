<template>
  <!--
    用量概览指标区（editorial）：
      英雄行三格：总请求 / 总 Token（页脚拆输入·输出·缓存） / 总消费（全页唯一品牌焦点卡）
      行内指标：平均时长，不占卡片位
  -->
  <div class="metric-hero-row">
    <!-- ① 总请求 -->
    <div class="metric-card">
      <div class="flex items-center gap-2">
        <Icon name="document" size="sm" class="metric-accent" :stroke-width="2" />
        <span class="metric-label">{{ t('usage.totalRequests') }}</span>
      </div>
      <div class="metric-value">{{ stats?.total_requests?.toLocaleString() || '0' }}</div>
      <div class="metric-note">{{ t('usage.inSelectedRange') }}</div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('usage.avgDuration') }}</span>
          <span class="stat-pair-value">{{ formatDuration(stats?.average_duration_ms || 0) }}</span>
        </div>
      </div>
    </div>

    <!-- ② 总 Token：页脚拆输入 / 输出 / 缓存（缓存键值带明细气泡） -->
    <div class="metric-card">
      <div class="flex items-center gap-2">
        <Icon name="cube" size="sm" class="metric-accent" :stroke-width="2" />
        <span class="metric-label">{{ t('usage.totalTokens') }}</span>
      </div>
      <div class="metric-value">
        {{ formatTokens(stats?.total_tokens || 0) }}
        <span class="metric-unit">tokens</span>
      </div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('usage.in') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.total_input_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('usage.out') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.total_output_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ cacheLabel() }}</span>
          <!--
            气泡锚在触发元素上：挂在 .stat-pair-value（truncate = overflow:hidden）里会被裁掉。
            .metric-card 自身也是 overflow-hidden，因此气泡向上展开并右对齐，避免被卡片裁切。
          -->
          <span class="group relative inline-flex cursor-help items-center gap-0.5" tabindex="0">
            <span class="stat-pair-value">{{ formatTokens(stats?.total_cache_tokens || 0) }}</span>
            <svg
              class="h-3.5 w-3.5 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <span
              class="pointer-events-none absolute bottom-full right-0 z-30 mb-2 hidden w-56 rounded-lg border border-gray-200 bg-white p-3 text-left text-xs text-gray-700 shadow-lg group-hover:block group-focus:block dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200"
            >
              <span class="mb-2 block font-medium text-gray-900 dark:text-white">
                {{ cacheDetailLabel() }}
              </span>
              <span class="flex items-center justify-between gap-3">
                <span>{{ t('usage.cacheCreationTokensLabel') }}</span>
                <span class="tabular-nums">
                  {{ formatTokens(stats?.total_cache_creation_tokens || 0) }}
                </span>
              </span>
              <span class="mt-1 flex items-center justify-between gap-3">
                <span>{{ t('usage.cacheReadTokensLabel') }}</span>
                <span class="tabular-nums">
                  {{ formatTokens(stats?.total_cache_read_tokens || 0) }}
                </span>
              </span>
            </span>
          </span>
        </div>
      </div>
    </div>

    <!-- ③ 总消费：全页唯一焦点卡 -->
    <div class="metric-card metric-card-hero">
      <div class="flex items-center gap-2">
        <Icon name="dollar" size="sm" class="metric-accent" :stroke-width="2" />
        <span class="metric-label">{{ t('usage.totalCost') }}</span>
      </div>
      <div class="metric-value">${{ (stats?.total_actual_cost || 0).toFixed(4) }}</div>
      <div class="stat-pairs metric-foot">
        <div v-if="showAccountCost && totalAccountCost != null" class="stat-pair">
          <span class="stat-pair-label">{{ t('usage.accountCost') }}</span>
          <span class="stat-pair-value text-amber-600 dark:text-amber-400">
            ${{ totalAccountCost.toFixed(4) }}
          </span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('usage.standardCost') }}</span>
          <span
            class="stat-pair-value text-gray-500 dark:text-dark-400"
            :class="{ 'line-through': strikeStandardCost }"
          >
            ${{ (stats?.total_cost || 0).toFixed(4) }}
          </span>
        </div>
      </div>
    </div>
  </div>

  <!-- 行内指标：平均时长，不占卡片位 -->
  <div class="stat-line">
    <span class="stat-line-item">
      {{ t('usage.avgDuration') }}
      <strong>{{ formatDuration(stats?.average_duration_ms || 0) }}</strong>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'
import type { UsageStatsResponse } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  stats: (AdminUsageStatsResponse | UsageStatsResponse) | null
  showAccountCost?: boolean
  strikeStandardCost?: boolean
}>(), {
  showAccountCost: true,
  strikeStandardCost: false,
})

const { t } = useI18n()

const totalAccountCost = computed(() => {
  const stats = props.stats as (AdminUsageStatsResponse & { total_account_cost?: number }) | null
  return stats?.total_account_cost ?? null
})
const showAccountCost = computed(() => props.showAccountCost)
const strikeStandardCost = computed(() => props.strikeStandardCost)

const formatDuration = (ms: number) =>
  ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`

const formatTokens = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + 'B'
  if (value >= 1e6) return (value / 1e6).toFixed(2) + 'M'
  if (value >= 1e3) return (value / 1e3).toFixed(2) + 'K'
  return value.toLocaleString()
}

const cacheLabel = () => t('usage.cacheTotal')
const cacheDetailLabel = () => t('usage.cacheBreakdown')
</script>
