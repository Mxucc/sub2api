<template>
  <!--
    指标网格：余额焦点卡（品牌浅底 + 顶线）与其余指标同一张网格。
    格子紧凑（内边距 13px / 数值 22px），一屏尽量多列，避免整页看不到关键信息。
  -->
  <div class="metric-grid">
    <div v-if="!isSimple" class="metric-card metric-card-hero">
      <div class="metric-label">{{ t('dashboard.balance') }}</div>
      <div class="metric-value">
        ${{ formatBalance(balance) }}
        <span class="metric-unit">{{ t('common.available') }}</span>
      </div>
      <p class="metric-note">
        {{ t('usage.totalCost') }} ·
        <span class="tabular-nums" :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
        <span class="tabular-nums" :title="t('dashboard.standard')">/ ${{ formatCost(stats?.total_cost || 0) }}</span>
      </p>
      <div class="metric-foot">
        <span>{{ t('dashboard.todayCost') }}</span>
        <strong>
          <span :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
          <span :title="t('dashboard.standard')">/ ${{ formatCost(stats?.today_cost || 0) }}</span>
        </strong>
      </div>
    </div>

    <!-- API 密钥 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.apiKeys') }}</div>
      <div class="metric-value">{{ stats?.total_api_keys || 0 }}</div>
      <p class="metric-note">
        <span class="metric-accent tabular-nums">{{ stats?.active_api_keys || 0 }}</span> {{ t('common.active') }}
      </p>
    </div>

    <!-- 今日请求 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayRequests') }}</div>
      <div class="metric-value">{{ stats?.today_requests || 0 }}</div>
      <p class="metric-note">
        {{ t('common.total') }}: <span class="tabular-nums">{{ formatNumber(stats?.total_requests || 0) }}</span>
      </p>
    </div>

    <!-- 今日消费 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayCost') }}</div>
      <div class="metric-value">
        <span :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
        <span class="metric-unit" :title="t('dashboard.standard')">/ ${{ formatCost(stats?.today_cost || 0) }}</span>
      </div>
      <p class="metric-note">
        {{ t('common.total') }}:
        <span class="tabular-nums" :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
        <span class="tabular-nums" :title="t('dashboard.standard')">/ ${{ formatCost(stats?.total_cost || 0) }}</span>
      </p>
    </div>

    <!-- 今日 Token -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayTokens') }}</div>
      <div class="metric-value">{{ formatTokens(stats?.today_tokens || 0) }}</div>
      <p class="metric-note">
        {{ t('dashboard.input') }}: <span class="tabular-nums">{{ formatTokens(stats?.today_input_tokens || 0) }}</span>
        / {{ t('dashboard.output') }}: <span class="tabular-nums">{{ formatTokens(stats?.today_output_tokens || 0) }}</span>
        / {{ t('dashboard.cache') }}: <span class="tabular-nums">{{ formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0)) }}</span>
      </p>
    </div>

    <!-- 累计 Token -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.totalTokens') }}</div>
      <div class="metric-value">{{ formatTokens(stats?.total_tokens || 0) }}</div>
      <p class="metric-note">
        {{ t('dashboard.input') }}: <span class="tabular-nums">{{ formatTokens(stats?.total_input_tokens || 0) }}</span>
        / {{ t('dashboard.output') }}: <span class="tabular-nums">{{ formatTokens(stats?.total_output_tokens || 0) }}</span>
        / {{ t('dashboard.cache') }}: <span class="tabular-nums">{{ formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0)) }}</span>
      </p>
    </div>

    <!-- 性能指标 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.performance') }}</div>
      <div class="metric-value">
        {{ formatTokens(stats?.rpm || 0) }}
        <span class="metric-unit">RPM</span>
      </div>
      <p class="metric-note">
        <span class="metric-accent tabular-nums">{{ formatTokens(stats?.tpm || 0) }}</span> TPM
      </p>
    </div>

    <!-- 平均响应 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.avgResponse') }}</div>
      <div class="metric-value">{{ formatDuration(stats?.average_duration_ms || 0) }}</div>
      <p class="metric-note">{{ t('dashboard.averageTime') }}</p>
    </div>
  </div>

  <!-- 平台台账：平台用量 + 配额窗口 -->
  <template v-if="!isSimple && platformCards.length > 0">
    <div class="section-heading">
      <h2 class="section-heading-title">{{ t('dashboard.platformBreakdown') }}</h2>
      <span class="section-heading-meta">{{ t('dashboard.platformCount', { count: platformCount }) }}</span>
    </div>

    <div class="metric-grid">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        data-testid="platform-card"
        :data-platform="item.platform"
        class="metric-card"
        :class="item.isOther ? 'border-dashed' : ''"
      >
        <div class="metric-label">
          {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
        </div>
        <div class="metric-value" :title="t('dashboard.actual')">
          ${{ formatCost(item.total_actual_cost) }}
        </div>
        <p class="metric-note">
          {{ t('dashboard.requests') }}:
          <span class="tabular-nums">{{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}</span>
          · {{ t('dashboard.tokens') }}:
          <span class="tabular-nums">{{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}</span>
        </p>

        <!-- 配额窗口：仅当 quota 配置存在、非 __other__ 且至少有一个窗口配了 limit 时显示 -->
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-4 space-y-3">
          <div class="metric-label">{{ t('dashboard.platformQuota.title') }}</div>
          <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
            <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="space-y-1.5">
              <div class="flex items-baseline justify-between gap-2">
                <span class="metric-note mt-0">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                <!-- limit=0：完全禁用 -->
                <span v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0" class="text-2xs font-medium text-red-600 dark:text-red-400">
                  {{ t('dashboard.platformQuota.disabled') }}
                </span>
                <!-- limit>0：已用 / 限额 -->
                <span v-else class="text-2xs tabular-nums text-gray-700 dark:text-gray-200">
                  ${{ formatUsd((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0) }} / ${{ formatUsd(quotaVal(item.quota, `${w}_limit_usd`) as number) }}
                </span>
              </div>
              <div class="h-1.5 w-full overflow-hidden rounded bg-gray-200 dark:bg-dark-700">
                <div
                  v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0"
                  class="h-full w-full rounded bg-red-500"
                />
                <div
                  v-else
                  class="h-full rounded transition-all"
                  :class="quotaBarClass(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                  :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                />
              </div>
              <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="metric-note">
                {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
              </p>
            </div>
          </template>
        </div>

        <div class="metric-foot">
          <span>{{ t('dashboard.todayCost') }}</span>
          <strong>${{ formatCost(item.today_actual_cost) }}</strong>
        </div>
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlatformDashboardStats, UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map。三档全空的记录不产生卡片，挂到卡片上也不渲染配额区。
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // 卡片集合 = 有用量的平台 ∪ 至少配置了一档限额的平台。
  // 三档全空的限额记录等价于不限额，不单独产生卡片。
  // 后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>(byPlat.keys())
  for (const [platform, q] of byQuota) {
    if (hasAnyLimit(q)) platforms.add(platform)
  }

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  // 排序：按 PLATFORM_ORDER，未知平台按名称排序
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  // __other__ 补差逻辑：只对 by_platform 有 usage 数据的总和计算
  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

// 标题右侧的平台计数 = 实际渲染的平台卡片数，不含"其他"差额卡。
const platformCount = computed(() => platformCards.value.filter((c) => !c.isOther).length)

// Quota helpers

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaBarClass(p: number): string {
  if (p >= 95) return 'bg-red-500'
  if (p >= 75) return 'bg-amber-500'
  return 'bg-primary-500'
}

// 与 formatBalance 一致使用 Intl.NumberFormat 做半偶舍入，避免 toFixed 在不同 JS 引擎
// 下偶发截断而非四舍五入（与后端展示精度不一致）。
const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return usdFormatter.format(n)
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>
