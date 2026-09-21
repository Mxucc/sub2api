<template>
  <!--
    用量概览（参考 new-api 的排版，只取布局）：
      首行三格：余额（品牌焦点卡） / 近期累计 Token（页脚拆输入·输出·缓存） / 快捷操作
      第二行：紧凑指标条（今日 Token · 今日消费 · 今日请求 · API 密钥）
      第三行：行内性能指标（RPM / TPM · 平均响应 · 累计 Token），不占卡片位
  -->
  <div class="metric-hero-row">
    <!-- ① 余额：全页唯一焦点卡 -->
    <div v-if="!isSimple" class="metric-card metric-card-hero">
      <div class="metric-label">{{ t('dashboard.balance') }}</div>
      <div class="metric-value">
        ${{ formatBalance(balance) }}
        <span class="metric-unit">{{ t('common.available') }}</span>
      </div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.actual') }}</span>
          <span class="stat-pair-value">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.standard') }}</span>
          <span class="stat-pair-value">${{ formatCost(stats?.total_cost || 0) }}</span>
        </div>
      </div>
    </div>

    <!-- ② 近期累计 Token：页脚拆输入 / 输出 / 缓存 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.totalTokens') }}</div>
      <div class="metric-value">
        {{ formatTokens(stats?.total_tokens || 0) }}
        <span class="metric-unit">tokens</span>
      </div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.input') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.total_input_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.output') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.total_output_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.cache') }}</span>
          <span class="stat-pair-value">{{ formatTokens(totalCacheTokens) }}</span>
        </div>
      </div>
    </div>

    <!-- ③ 快捷操作：两列动作格 -->
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.quickActions') }}</div>
      <div class="action-grid mt-1">
        <button class="action-item" @click="router.push('/keys')">
          <Icon name="key" size="sm" class="shrink-0 text-gray-400 dark:text-dark-500" />
          <span class="action-item-label">{{ t('dashboard.createApiKey') }}</span>
          <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-300 dark:text-dark-600" />
        </button>
        <button class="action-item" @click="router.push('/usage')">
          <Icon name="chart" size="sm" class="shrink-0 text-gray-400 dark:text-dark-500" />
          <span class="action-item-label">{{ t('dashboard.viewUsage') }}</span>
          <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-300 dark:text-dark-600" />
        </button>
        <button v-if="canUseBatchImage" class="action-item" @click="router.push('/batch-image')">
          <Icon name="sparkles" size="sm" class="shrink-0 text-gray-400 dark:text-dark-500" />
          <span class="action-item-label">{{ t('dashboard.batchImageAgent') }}</span>
          <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-300 dark:text-dark-600" />
        </button>
        <button class="action-item" @click="router.push('/redeem')">
          <Icon name="gift" size="sm" class="shrink-0 text-gray-400 dark:text-dark-500" />
          <span class="action-item-label">{{ t('dashboard.redeemCode') }}</span>
          <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-300 dark:text-dark-600" />
        </button>
      </div>
    </div>
  </div>

  <!-- 紧凑指标条：今日数据 -->
  <div class="metric-strip">
    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayTokens') }}</div>
      <div class="metric-value">{{ formatTokens(stats?.today_tokens || 0) }}</div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.input') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.today_input_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.output') }}</span>
          <span class="stat-pair-value">{{ formatTokens(stats?.today_output_tokens || 0) }}</span>
        </div>
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.cache') }}</span>
          <span class="stat-pair-value">{{ formatTokens(todayCacheTokens) }}</span>
        </div>
      </div>
    </div>

    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayCost') }}</div>
      <div class="metric-value metric-accent">${{ formatCost(stats?.today_actual_cost || 0) }}</div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('dashboard.standard') }}</span>
          <span class="stat-pair-value">${{ formatCost(stats?.today_cost || 0) }}</span>
        </div>
      </div>
    </div>

    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.todayRequests') }}</div>
      <div class="metric-value">{{ stats?.today_requests || 0 }}</div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('common.total') }}</span>
          <span class="stat-pair-value">{{ formatNumber(stats?.total_requests || 0) }}</span>
        </div>
      </div>
    </div>

    <div class="metric-card">
      <div class="metric-label">{{ t('dashboard.apiKeys') }}</div>
      <div class="metric-value">{{ stats?.total_api_keys || 0 }}</div>
      <div class="stat-pairs metric-foot">
        <div class="stat-pair">
          <span class="stat-pair-label">{{ t('common.active') }}</span>
          <span class="stat-pair-value metric-accent">{{ stats?.active_api_keys || 0 }}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- 行内性能指标：不占卡片位 -->
  <div class="stat-line justify-end sm:justify-start">
    <span class="stat-line-item">
      {{ t('dashboard.performance') }}
      <strong>{{ formatTokens(stats?.rpm || 0) }} RPM</strong>
      <span class="text-gray-300 dark:text-dark-600">/</span>
      <strong>{{ formatTokens(stats?.tpm || 0) }} TPM</strong>
    </span>
    <span class="stat-line-item">
      {{ t('dashboard.avgResponse') }}
      <strong>{{ formatDuration(stats?.average_duration_ms || 0) }}</strong>
    </span>
  </div>

  <!-- 平台拆分：折叠头 + 平台卡片（每张卡内是三行键值，参考用量概览的排版） -->
  <template v-if="!isSimple && platformCards.length > 0">
    <div>
      <button
        class="collapse-head"
        :aria-expanded="platformOpen"
        @click="platformOpen = !platformOpen"
      >
        <span class="collapse-head-title">
          <Icon name="grid" size="sm" class="text-gray-400 dark:text-dark-500" />
          {{ t('dashboard.platformBreakdown') }}
        </span>
        <span class="flex items-center gap-2">
          <span class="section-heading-meta">
            {{ t('dashboard.platformCount', { count: platformCount }) }}
          </span>
          <Icon
            :name="platformOpen ? 'chevronUp' : 'chevronDown'"
            size="sm"
            class="text-gray-400 dark:text-dark-500"
          />
        </span>
      </button>

      <div v-show="platformOpen" class="metric-strip mt-2.5">
        <div
          v-for="item in platformCards"
          :key="item.platform"
          data-testid="platform-card"
          :data-platform="item.platform"
          class="metric-card"
          :class="item.isOther ? 'border-dashed' : ''"
        >
          <div class="flex items-baseline justify-between gap-2">
            <span class="metric-label">
              {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
            </span>
            <span class="metric-value metric-accent" :title="t('dashboard.actual')">
              ${{ formatCost(item.total_actual_cost) }}
            </span>
          </div>

          <!-- 三行键值：今日消费 / 请求 / Token -->
          <div class="mt-2">
            <div class="platform-stat-row">
              <span class="stat-pair-label">{{ t('dashboard.todayCost') }}</span>
              <span class="stat-pair-value">${{ formatCost(item.today_actual_cost) }}</span>
            </div>
            <div class="platform-stat-row">
              <span class="stat-pair-label">{{ t('dashboard.requests') }}</span>
              <span class="stat-pair-value">
                {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
              </span>
            </div>
            <div class="platform-stat-row">
              <span class="stat-pair-label">{{ t('dashboard.tokens') }}</span>
              <span class="stat-pair-value">
                {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
              </span>
            </div>
          </div>

          <!-- 配额窗口：仅当 quota 配置存在、非 __other__ 且至少有一个窗口配了 limit 时显示 -->
          <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-3 space-y-2">
            <div class="metric-label">{{ t('dashboard.platformQuota.title') }}</div>
            <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
              <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="space-y-1">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="stat-pair-label">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <!-- limit=0：完全禁用 -->
                  <span v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0" class="text-2xs font-medium text-red-600 dark:text-red-400">
                    {{ t('dashboard.platformQuota.disabled') }}
                  </span>
                  <!-- limit>0：已用 / 限额 -->
                  <span v-else class="stat-pair-value">
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
                <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="stat-pair-label">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                </p>
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
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
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

// 「按平台拆分」折叠状态（参考用量概览：默认展开）
const platformOpen = ref(true)

// 缓存 token = 创建 + 读取（今日 / 累计）
const todayCacheTokens = computed(() =>
  (props.stats?.today_cache_creation_tokens || 0) + (props.stats?.today_cache_read_tokens || 0)
)
const totalCacheTokens = computed(() =>
  (props.stats?.total_cache_creation_tokens || 0) + (props.stats?.total_cache_read_tokens || 0)
)

onMounted(() => {
  void refreshBatchImageAccess()
})

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
