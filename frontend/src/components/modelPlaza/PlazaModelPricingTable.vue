<template>
  <div class="plaza-pricing-table" :style="accentStyle">
    <div class="overflow-x-auto">
      <table class="w-full min-w-[1000px] table-auto border-collapse text-sm tabular-nums">
        <colgroup>
          <col class="w-[25%]" />
          <col class="w-[11%]" />
          <col class="w-[9%]" />
          <col class="w-[14%]" />
          <col class="w-[11%]" />
          <col class="w-[8%]" />
          <col class="w-[14%]" />
          <col class="w-[8%]" />
        </colgroup>
        <thead>
          <tr
            class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400"
          >
            <th
              rowspan="2"
              class="border-r border-gray-100 py-2.5 pl-5 pr-4 text-left align-middle dark:border-dark-800"
            >
              {{ t('modelPlaza.table.model') }}
            </th>
            <th colspan="3" class="pz-bg pt-2 text-center">
              <div class="pz-title border-b pb-2 font-semibold">
                {{ t('modelPlaza.table.paidPrice') }}
                <span class="pz-unit ml-1 normal-case font-normal">{{ t('modelPlaza.table.unitPerMillion') }}</span>
              </div>
            </th>
            <th
              colspan="3"
              class="border-l border-gray-100 pt-2 text-center dark:border-dark-800"
            >
              <div class="border-b border-gray-200 pb-2 text-gray-400 dark:border-dark-600 dark:text-dark-500">
                {{ t('modelPlaza.table.officialPrice') }}
                <span class="ml-1 normal-case font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.unitPerMillion') }}</span>
              </div>
            </th>
            <th
              rowspan="2"
              class="border-l border-gray-100 py-2.5 pl-3 pr-5 text-right align-middle dark:border-dark-800"
            >
              {{ t('modelPlaza.table.rate') }}
            </th>
          </tr>
          <tr
            class="border-b border-gray-200 text-left text-[11px] font-medium uppercase leading-4 tracking-wide text-gray-400 dark:border-dark-700 dark:text-dark-500"
          >
            <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.input') }}</th>
            <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.output') }}</th>
            <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.cache') }}</th>
            <th class="border-l border-gray-100 px-3 py-2 font-medium dark:border-dark-800">
              {{ t('modelPlaza.table.input') }}
            </th>
            <th class="px-3 py-2 font-medium">{{ t('modelPlaza.table.output') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('modelPlaza.table.cache') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="{ model: m, period, key } in rows"
            :key="key"
            class="border-b border-gray-100 transition-colors last:border-b-0 hover:bg-gray-50/70 dark:border-dark-800 dark:hover:bg-dark-800/50"
          >
            <!-- 模型名 + 非 token 计费模式徽章;分时时段行额外标注时段 -->
            <td class="border-r border-gray-100 py-2.5 pl-5 pr-4 align-middle dark:border-dark-800">
              <div class="flex flex-wrap items-center gap-1.5">
                <span class="font-semibold tracking-tight text-gray-900 dark:text-white">{{ m.name }}</span>
                <!-- 时段徽章紧跟模型名,其余徽章排在后面,空间不足时先换行的是它们 -->
                <span
                  v-if="period"
                  class="inline-flex items-center whitespace-nowrap rounded-full bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] font-medium text-gray-500 dark:bg-dark-700/60 dark:text-dark-300"
                  :title="timePricingRowHint(m)"
                >
                  <span v-if="m.time_pricing?.weekdays_only" class="mr-1 font-sans">{{
                    t('modelPlaza.table.timePricingWeekdays')
                  }}</span>
                  {{ formatTimeWindow(period) }}
                </span>
                <span
                  v-if="platform && m.platform !== platform"
                  :class="[
                    'inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium',
                    platformBadgeLightClass(m.platform)
                  ]"
                >
                  {{ platformLabel(m.platform) }}
                </span>
                <span
                  v-if="billingMode(m) !== BILLING_MODE_TOKEN"
                  class="rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700/60 dark:text-dark-300"
                >
                  {{ billingModeLabel(m) }}
                </span>
                <span
                  v-if="m.long_context_basis === 'marginal'"
                  class="rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700/60 dark:text-dark-300"
                  :title="t('modelPlaza.table.tierHintMarginal')"
                >
                  {{ t('modelPlaza.table.marginalBadge') }}
                </span>
                <span
                  v-for="([effort, multiplier]) in reasoningEffortMultipliers(m)"
                  :key="effort"
                  class="rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300"
                  :title="t('modelPlaza.table.reasoningMultiplierHint', { effort, multiplier })"
                  :data-reasoning-effort="effort"
                >
                  {{ t('modelPlaza.table.reasoningMultiplierBadge', { effort, multiplier }) }}
                </span>
              </div>
            </td>

            <!-- token 计费:输入 / 输出 / 缓存(写/读),有阶梯时每档一行;档位标签只放输入列,其余列按行对齐 -->
            <template v-if="billingMode(m) === BILLING_MODE_TOKEN">
              <td class="pz-cell px-3 py-2.5 align-middle font-mono font-semibold text-gray-900 dark:text-white">
                <template v-if="tokenIntervals(m).length">
                  <div
                    v-for="(iv, idx) in tokenIntervals(m)"
                    :key="idx"
                    class="whitespace-nowrap text-xs leading-5"
                  >
                    <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500" :title="tierHint(m)">{{ tierLabel(iv) }}</span>
                    {{ paidPerMillion(iv.input_price, period) }}
                  </div>
                </template>
                <template v-else>{{ paidPerMillion(m.pricing?.input_price, period) }}</template>
              </td>
              <td class="pz-cell px-3 py-2.5 align-middle font-mono font-semibold text-gray-900 dark:text-white">
                <template v-if="tokenIntervals(m).length">
                  <div
                    v-for="(iv, idx) in tokenIntervals(m)"
                    :key="idx"
                    class="whitespace-nowrap text-xs leading-5"
                    :title="tierHint(m)"
                  >
                    {{ paidPerMillion(iv.output_price, period) }}
                  </div>
                </template>
                <template v-else>{{ paidPerMillion(m.pricing?.output_price, period) }}</template>
              </td>
              <td class="pz-cell px-3 py-2.5 align-middle">
                <template v-if="hasTierCachePricing(tokenIntervals(m))">
                  <div
                    v-for="(iv, idx) in tokenIntervals(m)"
                    :key="idx"
                    class="whitespace-nowrap font-mono text-xs leading-5 text-gray-700 dark:text-dark-200"
                    :title="tierHint(m)"
                  >
                    <template v-if="iv.cache_write_price != null || iv.cache_write_1h_price != null || iv.cache_read_price != null">
                      <span class="font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWriteShort') }}</span>
                      {{ paidPerMillion(iv.cache_write_price, period) }}
                      <template v-if="iv.cache_write_1h_price != null"
                        ><span class="font-sans font-normal text-gray-400 dark:text-dark-500"> (1h </span>{{ paidPerMillion(iv.cache_write_1h_price, period)
                        }}<span class="font-sans font-normal text-gray-400 dark:text-dark-500">)</span></template
                      >
                      <span class="ml-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheReadShort') }}</span>
                      {{ paidPerMillion(iv.cache_read_price, period) }}
                    </template>
                    <span v-else class="text-gray-400 dark:text-dark-500">-</span>
                  </div>
                </template>
                <div
                  v-else-if="hasCachePricing(m)"
                  class="space-y-0.5 font-mono text-xs text-gray-700 dark:text-dark-200"
                >
                  <div>
                    <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWrite') }}</span>
                    {{ paidPerMillion(m.pricing?.cache_write_price, period)
                    }}<template v-if="m.pricing?.cache_write_1h_price != null"
                      ><span class="font-sans font-normal text-gray-400 dark:text-dark-500"> (1h </span>{{ paidPerMillion(m.pricing.cache_write_1h_price, period)
                      }}<span class="font-sans font-normal text-gray-400 dark:text-dark-500">)</span></template
                    >
                  </div>
                  <div>
                    <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheRead') }}</span>
                    {{ paidPerMillion(m.pricing?.cache_read_price, period) }}
                  </div>
                </div>
                <span v-else class="text-gray-400 dark:text-dark-500">-</span>
              </td>
            </template>

            <!-- 按次 / 按图片计费:实付区整体合并,阶梯芯片或单一按次价 -->
            <template v-else>
              <td colspan="3" class="pz-cell px-3 py-2.5 align-middle">
                <div
                  v-if="requestIntervals(m).length"
                  class="flex flex-wrap items-center gap-1.5"
                >
                  <span
                    v-for="(iv, idx) in requestIntervals(m)"
                    :key="idx"
                    class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-700 dark:bg-dark-700/60 dark:text-dark-200"
                  >
                    <span class="font-sans text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
                    {{ paidRequestPrice(m, iv.per_request_price)
                    }}<span class="font-sans text-gray-400 dark:text-dark-500">{{ perUnitSuffix(m) }}</span>
                  </span>
                </div>
                <template v-else-if="m.pricing?.per_request_price != null">
                  <span class="font-mono font-semibold text-gray-900 dark:text-white">
                    {{ paidRequestPrice(m, m.pricing.per_request_price) }}
                  </span>
                  <span class="ml-1 text-xs text-gray-400 dark:text-dark-500">{{ perUnitSuffix(m) }}</span>
                </template>
                <span v-else class="text-gray-400 dark:text-dark-500">-</span>
              </td>
            </template>

            <!-- 官方价格(参考价,不乘倍率;官方有阶梯时每档一行);带计费表达式时该列只是基线 -->
            <td
              class="border-l border-gray-100 px-3 py-2.5 align-middle font-mono text-xs text-gray-500 dark:border-dark-800 dark:text-dark-400"
            >
              <template v-if="officialIntervals(m).length">
                <div
                  v-for="(iv, idx) in officialIntervals(m)"
                  :key="idx"
                  class="whitespace-nowrap leading-5"
                >
                  <span class="mr-1 font-sans text-gray-400 dark:text-dark-500" :title="t('modelPlaza.table.tierHint')">{{ tierLabel(iv) }}</span>
                  {{ official(iv.input_price) }}
                </div>
                <span
                  v-if="hasBillingExpr(m)"
                  class="chip mt-0.5"
                  :title="t('modelPlaza.table.timeTierBaselineNote')"
                >{{ t('modelPlaza.table.timeTierBaselineBadge') }}</span>
              </template>
              <template v-else>
                {{ official(m.official_pricing?.input_price) }}<span
                  v-if="hasBillingExpr(m)"
                  class="chip ml-1"
                  :title="t('modelPlaza.table.timeTierBaselineNote')"
                >{{ t('modelPlaza.table.timeTierBaselineBadge') }}</span>
              </template>
            </td>
            <td class="px-3 py-2.5 align-middle font-mono text-xs text-gray-500 dark:text-dark-400">
              <template v-if="officialIntervals(m).length">
                <div
                  v-for="(iv, idx) in officialIntervals(m)"
                  :key="idx"
                  class="whitespace-nowrap leading-5"
                  :title="t('modelPlaza.table.tierHint')"
                >
                  {{ official(iv.output_price) }}
                </div>
              </template>
              <template v-else>{{ official(m.official_pricing?.output_price) }}</template>
            </td>
            <td class="px-3 py-2.5 align-middle">
              <template v-if="hasTierCachePricing(officialIntervals(m))">
                <div
                  v-for="(iv, idx) in officialIntervals(m)"
                  :key="idx"
                  class="whitespace-nowrap font-mono text-xs leading-5 text-gray-500 dark:text-dark-400"
                  :title="t('modelPlaza.table.tierHint')"
                >
                  <template v-if="iv.cache_write_price != null || iv.cache_write_1h_price != null || iv.cache_read_price != null">
                    <span class="font-sans text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWriteShort') }}</span>
                    {{ official(iv.cache_write_price) }}
                    <template v-if="iv.cache_write_1h_price != null"
                      ><span class="font-sans text-gray-400 dark:text-dark-500"> (1h </span>{{ official(iv.cache_write_1h_price)
                      }}<span class="font-sans text-gray-400 dark:text-dark-500">)</span></template
                    >
                    <span class="ml-1 font-sans text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheReadShort') }}</span>
                    {{ official(iv.cache_read_price) }}
                  </template>
                  <span v-else class="text-gray-400 dark:text-dark-500">-</span>
                </div>
              </template>
              <div
                v-else-if="m.official_pricing && hasOfficialCache(m.official_pricing)"
                class="space-y-0.5 font-mono text-xs text-gray-500 dark:text-dark-400"
              >
                <div>
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWrite') }}</span>
                  {{ official(m.official_pricing.cache_write_price)
                  }}<template v-if="m.official_pricing.cache_write_1h_price != null"
                    ><span class="font-sans text-gray-400 dark:text-dark-500"> (1h </span>{{ official(m.official_pricing.cache_write_1h_price)
                    }}<span class="font-sans text-gray-400 dark:text-dark-500">)</span></template
                  >
                </div>
                <div>
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheRead') }}</span>
                  {{ official(m.official_pricing.cache_read_price) }}
                </div>
              </div>
              <span v-else class="text-gray-400 dark:text-dark-500">-</span>
            </td>

            <!-- 折扣倍率(分时时段行展示 生效倍率×时段倍率;生图独立倍率行展示独立倍率;专属倍率划线展示原倍率) -->
            <td
              class="border-l border-gray-100 py-2.5 pl-3 pr-5 text-right align-middle font-mono text-xs dark:border-dark-800"
            >
              <span
                v-if="period"
                class="font-semibold text-primary-600 dark:text-primary-300"
                :title="t('modelPlaza.table.timePricingRateHint', { rate: effectiveRate, multiplier: period.multiplier })"
                >{{ periodRate(period) }}x</span
              >
              <span
                v-else-if="usesIndependentImageRate(m)"
                class="font-semibold text-gray-700 dark:text-dark-300"
                >{{ requestRate(m) }}x</span
              >
              <template v-else-if="hasCustomRate">
                <span class="mr-1 text-gray-400 line-through dark:text-dark-500">{{ rateMultiplier }}x</span>
                <span class="font-semibold text-primary-600 dark:text-primary-300">{{ effectiveRate }}x</span>
              </template>
              <span v-else class="font-semibold text-gray-700 dark:text-dark-300">{{ effectiveRate }}x</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!--
      计费表达式分档价：billing_expr 存在时上方官方价格列只是基线（峰谷价模型高峰时段单价更高），
      这里按档列出真实单价；表达式形态识别不出（recognized=false / tiers 为空）时回退展示原文。
    -->
    <div v-if="billingExprRows.length > 0" class="border-t border-gray-100 dark:border-dark-800">
      <section
        v-for="(row, rowIndex) in billingExprRows"
        :key="row.key"
        class="border-b border-gray-100 px-5 py-3.5 last:border-b-0 dark:border-dark-800"
      >
        <header class="flex flex-wrap items-center gap-x-2 gap-y-1.5">
          <span class="font-semibold tracking-tight text-gray-900 dark:text-white">{{ row.model.name }}</span>
          <span class="chip border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-500/25 dark:bg-primary-500/10 dark:text-primary-300">
            {{ row.title }}
          </span>
          <span class="chip">{{ row.sourceLabel }}</span>
          <button
            type="button"
            class="btn btn-ghost ml-auto h-6 gap-1 px-1.5 text-2xs"
            :aria-expanded="isTierPanelOpen(row.key)"
            :aria-controls="tierPanelId(rowIndex)"
            @click="toggleTierPanel(row.key)"
          >
            {{ isTierPanelOpen(row.key) ? t('modelPlaza.table.timeTierCollapse') : t('modelPlaza.table.timeTierExpand') }}
            <Icon
              name="chevronDown"
              size="xs"
              :class="['tt-chevron', { 'is-open': isTierPanelOpen(row.key) }]"
            />
          </button>
        </header>

        <div v-if="isTierPanelOpen(row.key)" :id="tierPanelId(rowIndex)" class="mt-3">
          <template v-if="row.recognized && row.tiers.length > 0">
            <div class="overflow-x-auto">
              <table class="w-full min-w-[520px] table-auto border-collapse text-xs tabular-nums">
                <caption class="sr-only">{{ t('modelPlaza.table.timeTierCaption', { model: row.model.name }) }}</caption>
                <thead>
                  <tr class="border-b border-gray-200 text-2xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
                    <th scope="col" class="whitespace-nowrap py-1.5 pr-4 text-left font-medium">
                      {{ t('modelPlaza.table.timeTierTierColumn') }}
                    </th>
                    <th scope="col" class="py-1.5 pr-4 text-left font-medium">
                      {{ t('modelPlaza.table.timeTierConditionColumn') }}
                    </th>
                    <th
                      v-for="variable in row.variables"
                      :key="variable"
                      scope="col"
                      class="py-1.5 pl-4 text-right align-top font-medium"
                    >
                      <span class="block">{{ variableLabel(variable) }}</span>
                      <span class="block font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.timeTierUnitPerMillion') }}</span>
                    </th>
                    <th v-if="row.hasFlatFee" scope="col" class="py-1.5 pl-4 text-right align-top font-medium">
                      <span class="block">{{ t('modelPlaza.table.timeTierFlatColumn') }}</span>
                      <span class="block font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.timeTierFlatUnit') }}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(tier, tierIndex) in row.tiers"
                    :key="tierIndex"
                    class="tt-tier-row border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                    :style="{ '--tt-row': tierIndex }"
                  >
                    <th scope="row" class="whitespace-nowrap py-2 pr-3 text-left align-top font-medium text-gray-800 dark:text-dark-200 sm:pr-4">
                      {{ tierName(tier, tierIndex) }}
                    </th>
                    <td class="py-2 pr-3 align-top text-gray-500 dark:text-dark-400 sm:pr-4">
                      <template v-if="tier.time_windows?.length">
                        <span class="break-words">{{ tier.time_windows.join(' · ') }}</span>
                        <span v-if="tier.timezone" class="ml-1 text-gray-400 dark:text-dark-500">({{ tier.timezone }})</span>
                      </template>
                      <span v-else-if="tier.condition" class="break-all font-mono text-[11px]">{{ tier.condition }}</span>
                      <span v-else class="text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.timeTierUnconditional') }}</span>
                    </td>
                    <td
                      v-for="variable in row.variables"
                      :key="variable"
                      class="whitespace-nowrap py-2 pl-3 text-right align-top font-mono text-gray-800 dark:text-dark-200 sm:pl-4"
                    >
                      {{ tierCoefficient(tier, variable) }}
                    </td>
                    <td
                      v-if="row.hasFlatFee"
                      class="whitespace-nowrap py-2 pl-3 text-right align-top font-mono text-gray-800 dark:text-dark-200 sm:pl-4"
                    >
                      {{ tierFlatFee(tier) }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p class="mt-2 text-2xs text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.timeTierUnitNote') }}</p>
          </template>

          <template v-else>
            <p class="text-2xs text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.timeTierRawNote') }}</p>
            <code class="rise mt-1.5 block whitespace-pre-wrap break-all bg-gray-100 px-2.5 py-2 font-mono text-[11px] leading-5 text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ row.expression }}</code>
          </template>
        </div>
      </section>

      <p class="border-t border-gray-100 px-5 py-2.5 text-2xs text-gray-400 dark:border-dark-800 dark:text-dark-500">
        {{ t('modelPlaza.table.timeTierBaselineNote') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'
import { platformAccentColor, platformBadgeLightClass, platformLabel } from '@/utils/platformColors'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_IMAGE,
  REASONING_EFFORT_LEVELS,
  type BillingMode
} from '@/constants/channel'
import type {
  PlazaBillingTier,
  PlazaBillingVariable,
  PlazaModel,
  PlazaTimePricingPeriod
} from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'

/** 配置了倍率的推理等级,按 REASONING_EFFORT_LEVELS 顺序返回「等级 → 倍率」。 */
function reasoningEffortMultipliers(model: PlazaModel): [string, number][] {
  const multipliers = model.pricing?.reasoning_effort_multipliers
  return REASONING_EFFORT_LEVELS.flatMap(effort => {
    const multiplier = multipliers?.[effort]
    return typeof multiplier === 'number' && Number.isFinite(multiplier) && multiplier > 0
      ? [[effort, multiplier] as [string, number]]
      : []
  })
}

const props = defineProps<{
  models: PlazaModel[]
  /** 分组平台;实付分区底色随平台着色,未知平台回退品牌青。 */
  platform?: string
  /** 分组默认倍率。 */
  rateMultiplier: number
  /** 用户专属倍率;与默认不同,实付价按此计算并划线展示原倍率。 */
  userRateMultiplier?: number | null
  /** 生图独立倍率:true 时图片计费模型的实付倍率取 imageRateMultiplier,不取分组/专属倍率。 */
  imageRateIndependent?: boolean
  imageRateMultiplier?: number | null
  /**
   * 高峰窗口描述(含倍率与服务器时区标注),空串/缺省 = 分组未启用高峰。
   * 表格所有价格均为不含高峰因子的口径,该窗口仅用于分时时段行的 tooltip 披露:
   * 与高峰重叠的部分实付还会再乘高峰倍率。
   */
  peakWindow?: string
  peakRateMultiplier?: number | null
}>()

const { t } = useI18n()

/** 实付分区只从平台拿一个主色,浅底/标题/下划线全部由 scoped CSS 用 color-mix 派生。 */
const accentStyle = computed(() => ({ '--plaza-accent': platformAccentColor(props.platform ?? '') }))

const PER_MILLION = 1_000_000

/**
 * 展示顺序:
 * 1. token 计费的排在前,按图/按次计费的沉到末尾——它们的官方 token 价与实付的按张/按次价不同量纲,混排无意义;
 * 2. 组内按官方输出价从高到低,无官方价的排最后;
 * 3. 同价按名称降序(新版本号在前,如 gpt-5.6 先于 gpt-5.5)。
 */
const sortedModels = computed(() => {
  return [...props.models].sort((a, b) => {
    const ta = billingMode(a) === BILLING_MODE_TOKEN
    const tb = billingMode(b) === BILLING_MODE_TOKEN
    if (ta !== tb) return ta ? -1 : 1
    const pa = a.official_pricing?.output_price ?? null
    const pb = b.official_pricing?.output_price ?? null
    if (pa != null && pb != null && pa !== pb) return pb - pa
    if (pa != null && pb == null) return -1
    if (pa == null && pb != null) return 1
    return b.name.localeCompare(a.name)
  })
})

const effectiveRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const hasCustomRate = computed(
  () => props.userRateMultiplier != null && props.userRateMultiplier !== props.rateMultiplier
)

function billingMode(m: PlazaModel): BillingMode {
  return (m.pricing?.billing_mode || BILLING_MODE_TOKEN) as BillingMode
}

function billingModeLabel(m: PlazaModel): string {
  return billingMode(m) === BILLING_MODE_IMAGE
    ? t('modelPlaza.table.perImage')
    : t('modelPlaza.table.perRequest')
}

/** 价格统一保底 2 位小数,更长的有效小数原样保留。 */
const MIN_DECIMALS = 2

/** 表格行:每个模型一行标准价;配置了分时倍率的模型再按时段各加一行。 */
interface PlazaRow {
  model: PlazaModel
  period: PlazaTimePricingPeriod | null
  key: string
}

const rows = computed<PlazaRow[]>(() =>
  sortedModels.value.flatMap((m) => {
    const base: PlazaRow = { model: m, period: null, key: `${m.platform}:${m.name}` }
    const periodRows = timePeriods(m).map<PlazaRow>((p, idx) => ({
      model: m,
      period: p,
      key: `${m.platform}:${m.name}:${idx}`
    }))
    return [base, ...periodRows]
  })
)

/** 时段行的生效倍率 = 生效倍率 × 时段倍率(去掉浮点噪声)。 */
function periodRate(period: PlazaTimePricingPeriod): number {
  return Math.round(effectiveRate.value * period.multiplier * 1000) / 1000
}

/** 实付价 = 渠道单价 × 生效倍率(时段行再乘时段倍率),按 $/1M token 展示。 */
function paidPerMillion(value: number | null | undefined, period: PlazaTimePricingPeriod | null = null): string {
  if (value == null) return '-'
  const rate = period ? periodRate(period) : effectiveRate.value
  return formatScaled(value * rate, PER_MILLION, MIN_DECIMALS)
}

/** 图片计费模型且分组开启生图独立倍率:实付倍率取独立倍率,与计费口径一致。 */
function usesIndependentImageRate(m: PlazaModel): boolean {
  return billingMode(m) === BILLING_MODE_IMAGE && props.imageRateIndependent === true
}

/** 按次/按图片行的生效倍率。 */
function requestRate(m: PlazaModel): number {
  return usesIndependentImageRate(m) ? (props.imageRateMultiplier ?? 1) : effectiveRate.value
}

/** 按次 / 按图片单价(乘该行生效倍率,不换算 1M)。 */
function paidRequestPrice(m: PlazaModel, value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value * requestRate(m), 1, MIN_DECIMALS)
}

/** 官方参考价不乘倍率。 */
function official(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value, PER_MILLION, MIN_DECIMALS)
}

/** 非 token 计费的单位后缀:按图片 → “/ 张”,按次 → “/ 次”。 */
function perUnitSuffix(m: PlazaModel): string {
  return billingMode(m) === BILLING_MODE_IMAGE
    ? t('modelPlaza.table.perUnitImage')
    : t('modelPlaza.table.perUnitRequest')
}

function hasCachePricing(m: PlazaModel): boolean {
  return m.pricing?.cache_write_price != null || m.pricing?.cache_write_1h_price != null || m.pricing?.cache_read_price != null
}

function hasOfficialCache(o: NonNullable<PlazaModel['official_pricing']>): boolean {
  return o.cache_write_price != null || o.cache_read_price != null || o.cache_write_1h_price != null
}

/** 分时倍率时段(后端只给出倍率 ≠ 1 的时段,已升序)。 */
function timePeriods(m: PlazaModel): PlazaTimePricingPeriod[] {
  return m.time_pricing?.periods ?? []
}

/**
 * 时段行 tooltip:仅工作日生效的配置换用带周末回落说明的文案;
 * 分组启用高峰倍率时追加披露——本行价格不含高峰因子,与高峰窗口重叠的部分实付再乘高峰倍率。
 */
function timePricingRowHint(m: PlazaModel): string {
  const key = m.time_pricing?.weekdays_only
    ? 'modelPlaza.table.timePricingRowHintWeekdays'
    : 'modelPlaza.table.timePricingRowHint'
  let hint = t(key, { timezone: m.time_pricing?.timezone })
  if (props.peakWindow) {
    hint += t('modelPlaza.table.timePricingRowHintPeak', {
      window: props.peakWindow,
      multiplier: props.peakRateMultiplier ?? 1
    })
  }
  return hint
}

/** “00:30–08:30”;整分钟的 HH:mm:ss 省略秒。 */
function formatTimeWindow(p: PlazaTimePricingPeriod): string {
  const clock = (v: string) => v.replace(/^(\d{2}:\d{2}):00$/, '$1')
  return `${clock(p.start_time)}–${clock(p.end_time)}`
}

/** 上下文档位按下限升序展示(后端已升序,此处兜底)。 */
function sortByContext(intervals: UserPricingInterval[]): UserPricingInterval[] {
  return [...intervals].sort((a, b) => a.min_tokens - b.min_tokens)
}

/** token 模式的阶梯定价(内联进输入/输出/缓存列)。 */
function tokenIntervals(m: PlazaModel): UserPricingInterval[] {
  return sortByContext(m.pricing?.intervals ?? []).map(iv => resolveIntervalPrices(iv, m.pricing!))
}

/** 官方阶梯(后端按目录规则合成,不受分组开关影响)。 */
function officialIntervals(m: PlazaModel): UserPricingInterval[] {
  return sortByContext(m.official_pricing?.intervals ?? [])
}

/** 任一档带缓存价才按档渲染缓存列;否则沿用平价的写入/读取两行。 */
function hasTierCachePricing(intervals: UserPricingInterval[]): boolean {
  return intervals.some((iv) =>
    iv.cache_write_price != null || iv.cache_write_1h_price != null || iv.cache_read_price != null ||
    iv.cache_write_multiplier != null || iv.cache_read_multiplier != null
  )
}

/** 档位说明:整单按档计价,或(平台旧规则)仅超出部分按档计价。 */
function tierHint(m: PlazaModel): string {
  return m.long_context_basis === 'marginal'
    ? t('modelPlaza.table.tierHintMarginal')
    : t('modelPlaza.table.tierHint')
}


/** 按次/按图模式的阶梯定价(仅保留配了按次价的档位)。 */
function requestIntervals(m: PlazaModel): UserPricingInterval[] {
  return (m.pricing?.intervals ?? []).filter((iv) => iv.per_request_price != null)
}

/**
 * 档位标签:优先后端/管理员给出的 tier_label,否则按区间生成统一形态——
 * 有上限为「≤上限」,末档为「>下限」;档位升序排列,相邻的 ≤100K / ≤200K 即表示 (100K,200K]。
 */
function tierLabel(iv: UserPricingInterval): string {
  if (iv.tier_label) return iv.tier_label
  const { min_tokens: min, max_tokens: max } = iv
  return max == null ? `>${formatTokenCount(min)}` : `≤${formatTokenCount(max)}`
}

function formatTokenCount(n: number): string {
  if (n >= 1_000_000) return `${trimZero(n / 1_000_000)}M`
  if (n >= 1_000) return `${trimZero(n / 1_000)}K`
  return String(n)
}

function trimZero(n: number): string {
  return String(Math.round(n * 100) / 100)
}

/* ===== 计费表达式分档价(billing_expr) ===== */

/** 该模型是否带可展示的计费表达式;带表达式时官方价格列只是基线价。 */
function hasBillingExpr(m: PlazaModel): boolean {
  return !!m.official_pricing?.billing_expr?.expression?.trim()
}

/** 分档区块里一个模型的一行。 */
interface PlazaBillingExprRow {
  key: string
  model: PlazaModel
  expression: string
  /** 表达式形态是否可识别;false 或 tiers 为空时展示原文而非分档表。 */
  recognized: boolean
  tiers: PlazaBillingTier[]
  /** 表头列:该表达式引用到的计价变量并集(后端按固定顺序给出)。 */
  variables: PlazaBillingVariable[]
  /** 任一档带按次固定费用时才有「每次请求」列。 */
  hasFlatFee: boolean
  title: string
  sourceLabel: string
}

/** 需要分档展示的模型(表达式为空视作未提供,不渲染区块)。 */
const billingExprRows = computed<PlazaBillingExprRow[]>(() =>
  sortedModels.value.flatMap<PlazaBillingExprRow>((m) => {
    const expr = m.official_pricing?.billing_expr
    if (!expr || !expr.expression.trim()) return []
    const tiers = expr.tiers ?? []
    const variables = expr.variables?.length
      ? expr.variables
      : [...new Set(tiers.flatMap((tier) => tier.variables ?? []))]
    return [{
      key: `${m.platform}:${m.name}`,
      model: m,
      expression: expr.expression,
      recognized: expr.recognized === true,
      tiers,
      variables,
      hasFlatFee: tiers.some((tier) => (tier.constant ?? 0) > 0),
      title: expr.time_dependent
        ? t('modelPlaza.table.timeTierTitleTime')
        : t('modelPlaza.table.timeTierTitleTiers'),
      sourceLabel: expr.source === 'catalog'
        ? t('modelPlaza.table.timeTierSourceCatalog')
        : t('modelPlaza.table.timeTierSourceBuiltin')
    }]
  })
)

/** 展开/收起状态:key = 平台:模型名。缺省展开——真实单价不能被默认藏起来。 */
const collapsedPanels = ref<Record<string, boolean>>({})

function isTierPanelOpen(key: string): boolean {
  return collapsedPanels.value[key] !== true
}

function toggleTierPanel(key: string): void {
  const open = isTierPanelOpen(key)
  collapsedPanels.value = { ...collapsedPanels.value, [key]: open }
}

/** 分档面板 id,供 aria-controls 关联;区块内唯一即可。 */
function tierPanelId(index: number): string {
  return `plaza-billing-expr-${index}`
}

/**
 * 档位单价:coefficients 的单位就是 USD / 百万 token,直接按该口径展示
 * (与官方价格列同口径,不含分组倍率)。为零/缺失的变量按 “-” 展示。
 */
function tierCoefficient(tier: PlazaBillingTier, variable: PlazaBillingVariable): string {
  const value = tier.coefficients?.[variable]
  return value == null ? '-' : formatScaled(value, 1, MIN_DECIMALS)
}

/** 该档的按次固定费用(USD / 次);per_request 档只渲染这一列。 */
function tierFlatFee(tier: PlazaBillingTier): string {
  return tier.constant == null ? '-' : formatScaled(tier.constant, 1, MIN_DECIMALS)
}

/** 计价变量 → 中文标签;未知变量原样展示,不猜含义。 */
function variableLabel(variable: PlazaBillingVariable): string {
  switch (variable) {
    case 'p': return t('modelPlaza.table.varInput')
    case 'c': return t('modelPlaza.table.varOutput')
    case 'cr': return t('modelPlaza.table.varCacheRead')
    case 'cc': return t('modelPlaza.table.varCacheWrite')
    case 'cc1h': return t('modelPlaza.table.varCacheWrite1h')
    case 'img': return t('modelPlaza.table.varImageInput')
    case 'img_cr': return t('modelPlaza.table.varImageCacheRead')
    case 'ai': return t('modelPlaza.table.varAudioInput')
    case 'ao': return t('modelPlaza.table.varAudioOutput')
    default: return variable
  }
}

/** 档名:已知档名给中文,未知档名原样展示,空档名按序号兜底。 */
function tierName(tier: PlazaBillingTier, index: number): string {
  switch (tier.name) {
    case 'peak': return t('modelPlaza.table.timeTierNamePeak')
    case 'off_peak': return t('modelPlaza.table.timeTierNameOffPeak')
    case 'base': return t('modelPlaza.table.timeTierNameBase')
    default: return tier.name.trim() || t('modelPlaza.table.timeTierUnnamedTier', { index: index + 1 })
  }
}
</script>

<style scoped>
/* 实付分区配色统一从 --plaza-accent(平台主色)派生,新增平台无需扩展样式 */
.plaza-pricing-table {
  --pz-title: color-mix(in srgb, var(--plaza-accent) 82%, black);
  --pz-bg: color-mix(in srgb, var(--plaza-accent) 5%, transparent);
  --pz-bg-hover: color-mix(in srgb, var(--plaza-accent) 9%, transparent);
}

.dark .plaza-pricing-table {
  --pz-title: color-mix(in srgb, var(--plaza-accent) 72%, white);
  --pz-bg: color-mix(in srgb, var(--plaza-accent) 5%, transparent);
  --pz-bg-hover: color-mix(in srgb, var(--plaza-accent) 8%, transparent);
}

.pz-bg,
.pz-cell {
  background-color: var(--pz-bg);
}

.pz-cell {
  transition: background-color 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

tbody tr:hover .pz-cell {
  background-color: var(--pz-bg-hover);
}

.pz-title {
  /* color-mix 不可用的老浏览器回退为平台原色 */
  color: var(--plaza-accent);
  color: var(--pz-title);
  border-color: color-mix(in srgb, var(--pz-title) 30%, transparent);
}

.pz-unit {
  color: color-mix(in srgb, var(--pz-title) 62%, transparent);
}

/* 分档价:档位行错峰入场(45ms/行),展开/收起时重放;动效只表达状态变化 */
.tt-tier-row {
  animation: tt-tier-in 320ms cubic-bezier(0.16, 1, 0.3, 1) backwards;
  animation-delay: calc(var(--tt-row, 0) * 45ms);
}

@keyframes tt-tier-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

/* 展开/收起指示:只在展开态旋转,150ms 完成 */
.tt-chevron {
  transition: transform 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

.tt-chevron.is-open {
  transform: rotate(180deg);
}

@media (prefers-reduced-motion: reduce) {
  .tt-tier-row {
    animation: none;
  }

  .tt-chevron {
    transition: none;
  }
}
</style>
