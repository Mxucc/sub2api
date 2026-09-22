/**
 * Model Plaza API（公开端点，可匿名访问）
 * 以分组为中心的模型价目：分组信息 + 模型渠道定价 + LiteLLM 官方参考价。
 * 带 token 请求时后端会额外返回专属分组与用户专属倍率。
 */

import { apiClient } from './client'
import type { UserPricingInterval, UserSupportedModelPricing } from './channels'

/** 官方参考价（USD per token，与计费目录同源；字段缺失 = 目录未覆盖）。 */
export interface PlazaOfficialPricing {
  input_price: number | null
  output_price: number | null
  /** 5m 缓存写入（= LiteLLM cache_creation）。 */
  cache_write_price: number | null
  /** 1h 缓存写入（LiteLLM cache_creation_above_1hr），多数模型缺失。 */
  cache_write_1h_price?: number | null
  cache_read_price: number | null
  /** 官方长上下文阶梯（多档模型才有），不受分组开关影响。 */
  intervals?: UserPricingInterval[]
  /**
   * 声明式计费表达式的展示形态，仅该模型带表达式时给出。
   * 存在时上面的 input_price / output_price 只是**基线价**：峰谷价模型（如 DeepSeek）
   * 高峰时段实际单价比基线高，真实价格以 billing_expr.tiers 为准。
   */
  billing_expr?: PlazaBillingExpr
}

/**
 * 计价变量：表达式中带价格的 token 维度，与后端 billingexpr.PricedVars 同源。
 * p=输入 / c=输出 / cr=缓存命中 / cc=缓存写入 / cc1h=1h 缓存写入 /
 * img=图片输入 / img_cr=图片缓存命中 / ai=音频输入 / ao=音频输出。
 */
export type PlazaBillingVariable =
  | 'p'
  | 'c'
  | 'cr'
  | 'cc'
  | 'cc1h'
  | 'img'
  | 'img_cr'
  | 'ai'
  | 'ao'

/** 分档计价单位：按百万 token / 按次 / 两者相加。 */
export type PlazaBillingUnit =
  | 'per_million_tokens'
  | 'per_request'
  | 'per_million_tokens_plus_request'

/** 表达式中的一档价格。 */
export interface PlazaBillingTier {
  /** tier() 的标签，如 peak / off_peak / base。 */
  name: string
  /** 该档的条件原文；表达式最后的分支（else）没有条件。 */
  condition?: string
  /**
   * 计价变量 → **USD / 百万 token**（即展示单位，不需要再乘除换算）。
   * 该档不按 token 计价（per_request）时为空。
   */
  coefficients?: Partial<Record<PlazaBillingVariable, number>>
  /** 该档的按次固定费用（USD），与 token 费相加；为 0 时后端省略该字段。 */
  constant?: number
  unit: PlazaBillingUnit
  /** 该档引用到的计价变量（后端按固定顺序给出），用于动态生成表头列。 */
  variables?: PlazaBillingVariable[]
  /** 人读中文时段，如「周一至周五 09:00-12:00, 14:00-18:00」；条件不是可识别的时间谓词时省略。 */
  time_windows?: string[]
  /** 时间窗所用时区（IANA 名），如 Asia/Shanghai。 */
  timezone?: string
}

/** 计费表达式的展示形态。 */
export interface PlazaBillingExpr {
  /** 原始表达式，recognized 为 false 时按原文展示。 */
  expression: string
  /** 表达式来源：'catalog'（价格表 / override 文件）| 'builtin'（内置）。 */
  source: 'catalog' | 'builtin'
  /** false 表示不是可识别的条件分档形态，此时 tiers 为空，应展示原文而非分档表。 */
  recognized: boolean
  /** true 表示价格随时刻变化（存在时段条件）。 */
  time_dependent: boolean
  /** 所有档引用到的计价变量并集（后端按固定顺序给出）。 */
  variables?: PlazaBillingVariable[]
  tiers: PlazaBillingTier[]
}

/**
 * 多档时的计价基准：
 * - whole_request：整单按所在档单价计价（目录阶梯、渠道区间）；
 * - marginal：仅超出阈值的部分按该档单价计价（平台旧规则）。
 */
export type PlazaLongContextBasis = 'whole_request' | 'marginal'

/** 分时倍率时段：配置时区当天 [start_time, end_time) 内整单实付乘 multiplier。 */
export interface PlazaTimePricingPeriod {
  start_time: string
  end_time: string
  multiplier: number
}

/** 计费会生效的分时倍率（仅倍率 ≠ 1 的时段，已按开始时间升序）。 */
export interface PlazaTimePricing {
  /** IANA 时区名，如 Asia/Shanghai。 */
  timezone: string
  /** true 时时段仅周一至周五生效，周末整天按标准价计费。 */
  weekdays_only?: boolean
  periods: PlazaTimePricingPeriod[]
}

export interface PlazaModel {
  name: string
  platform: string
  /** 实收口径的展示定价：档位可提供绝对单价或相对基础价倍率；均为标准时段价。 */
  pricing: UserSupportedModelPricing | null
  official_pricing: PlazaOfficialPricing | null
  /** 仅多档模型返回。 */
  long_context_basis?: PlazaLongContextBasis
  /** 仅配置了分时倍率的模型返回。 */
  time_pricing?: PlazaTimePricing
}

export interface ModelPlazaGroup {
  id: number
  name: string
  description: string
  platform: string
  /** 'standard' | 'subscription' */
  subscription_type: string
  rate_multiplier: number
  /** 登录且管理员为该用户配了专属倍率时返回；生效倍率 = user_rate ?? rate_multiplier。 */
  user_rate_multiplier?: number
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
  is_exclusive: boolean
  /** 生图独立倍率：true 时图片计费模型的实付倍率取 image_rate_multiplier，不取分组/专属倍率。 */
  image_rate_independent: boolean
  image_rate_multiplier: number
  /** 分组是否启用长上下文阶梯计费；false 时实付列只展示最低档，官方阶梯仅供参考。 */
  long_context_pricing_enabled: boolean
  models: PlazaModel[]
}

export interface ModelPlazaResponse {
  /** 管理员配置的全局价格说明（Markdown）。 */
  description: string
  groups: ModelPlazaGroup[]
}

/** 获取模型广场数据。开关未启用时后端返回 404。 */
export async function getModelPlaza(options?: { signal?: AbortSignal }): Promise<ModelPlazaResponse> {
  const { data } = await apiClient.get<ModelPlazaResponse>('/model-plaza', {
    signal: options?.signal
  })
  return data
}

export const modelPlazaAPI = { getModelPlaza }

export default modelPlazaAPI
