/**
 * Admin Channels API endpoints
 * Handles channel management for administrators
 */

import { apiClient } from '../client'
import type { BillingMode, ChannelStatus, BillingModelSource } from '@/constants/channel'

export type { BillingMode } from '@/constants/channel'

export interface PricingInterval {
  id?: number
  min_tokens: number
  max_tokens: number | null
  tier_label: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_write_1h_price?: number | null
  cache_read_price: number | null
  input_multiplier: number | null
  output_multiplier: number | null
  cache_write_multiplier: number | null
  cache_read_multiplier: number | null
  per_request_price: number | null
  sort_order: number
}

export interface ChannelTimePricingPeriod {
  start_time: string
  end_time: string
  multiplier: number
}

export interface ChannelTimePricing {
  timezone: string
  weekdays_only?: boolean
  periods: ChannelTimePricingPeriod[]
}

export interface ChannelModelPricing {
  id?: number
  platform: string
  models: string[]
  billing_mode: BillingMode
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_write_1h_price?: number | null
  cache_read_price: number | null
  fast_multiplier?: number | null
  flex_multiplier?: number | null
  max_reasoning_effort_multiplier?: number | null
  image_input_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: PricingInterval[]
  time_pricing: ChannelTimePricing | null
}

export interface AccountStatsPricingRule {
  id?: number
  name: string
  group_ids: number[]
  account_ids: number[]
  pricing: ChannelModelPricing[]
}

export interface Channel {
  id: number
  name: string
  description: string
  status: ChannelStatus
  billing_model_source: BillingModelSource
  restrict_models: boolean
  features_config?: Record<string, unknown>
  group_ids: number[]
  model_pricing: ChannelModelPricing[]
  model_mapping: Record<string, Record<string, string>> // platform → {src→dst}
  apply_pricing_to_account_stats: boolean
  account_stats_pricing_rules: AccountStatsPricingRule[]
  created_at: string
  updated_at: string
}

export interface CreateChannelRequest {
  name: string
  description?: string
  group_ids?: number[]
  model_pricing?: ChannelModelPricing[]
  model_mapping?: Record<string, Record<string, string>>
  billing_model_source?: string
  restrict_models?: boolean
  features_config?: Record<string, unknown>
  apply_pricing_to_account_stats?: boolean
  account_stats_pricing_rules?: AccountStatsPricingRule[]
}

export interface UpdateChannelRequest {
  name?: string
  description?: string
  status?: string
  group_ids?: number[]
  model_pricing?: ChannelModelPricing[]
  model_mapping?: Record<string, Record<string, string>>
  billing_model_source?: string
  restrict_models?: boolean
  features_config?: Record<string, unknown>
  apply_pricing_to_account_stats?: boolean
  account_stats_pricing_rules?: AccountStatsPricingRule[]
}

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

/**
 * List channels with pagination
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<Channel>> {
  const { data } = await apiClient.get<PaginatedResponse<Channel>>('/admin/channels', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get channel by ID
 */
export async function getById(id: number): Promise<Channel> {
  const { data } = await apiClient.get<Channel>(`/admin/channels/${id}`)
  return data
}

/**
 * Create a new channel
 */
export async function create(req: CreateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.post<Channel>('/admin/channels', req)
  return data
}

/**
 * Update a channel
 */
export async function update(id: number, req: UpdateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.put<Channel>(`/admin/channels/${id}`, req)
  return data
}

/**
 * Delete a channel
 */
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/channels/${id}`)
}

export interface ModelDefaultPricing {
  found: boolean
  input_price?: number    // per-token price
  output_price?: number
  cache_write_price?: number
  cache_write_1h_price?: number | null
  cache_read_price?: number
  image_input_price?: number
  image_output_price?: number
  max_reasoning_effort_multiplier?: number | null
}

export async function getModelDefaultPricing(model: string): Promise<ModelDefaultPricing> {
  const { data } = await apiClient.get<ModelDefaultPricing>('/admin/channels/model-pricing', {
    params: { model }
  })
  return data
}

export interface SyncPricingModelsResult {
  models: string[]
}

/**
 * Fetch the latest model names from the LiteLLM pricing catalog for the given platform
 */
export async function syncPricingModels(platform: string): Promise<SyncPricingModelsResult> {
  const { data } = await apiClient.get<SyncPricingModelsResult>('/admin/channels/pricing/sync-models', {
    params: { platform }
  })
  return data
}

/**
 * Read-only model price catalog (`GET /admin/channels/pricing/catalog`).
 *
 * The catalog lists every model the price sources know about, whether or not a
 * channel serves it. Prices are the default price-card rates, before group or
 * channel overrides; `*_price_per_million` is already USD per 1M tokens.
 */
export interface ModelPricingCatalogTier {
  name: string
  condition?: string
  coefficients?: Record<string, number>
  constant?: number
  unit: string
  variables?: string[]
  time_windows?: string[]
  timezone?: string
}

export interface ModelPricingCatalogItem {
  model: string
  provider?: string
  mode?: string
  source: string
  input_price_per_token: number
  output_price_per_token: number
  cache_read_price_per_token: number
  cache_creation_price_per_token: number
  image_input_price_per_token: number
  image_output_price_per_token: number
  input_price_per_million: number
  output_price_per_million: number
  cache_read_price_per_million: number
  cache_creation_price_per_million: number
  billing_expr?: string
  billing_expr_source?: string
  tiers?: ModelPricingCatalogTier[]
  time_dependent: boolean
  expr_recognized: boolean
  token_pricing_absent: boolean
  fallback_only: boolean
}

export interface ModelPricingCatalogResponse {
  items: ModelPricingCatalogItem[]
  total: number
  page: number
  page_size: number
}

export interface ModelPricingCatalogQuery {
  /** Case-insensitive model name substring. */
  q?: string
  /** Exact provider match, e.g. "anthropic" / "vertex_ai-language-models". */
  provider?: string
  /** "catalog" | "builtin" */
  source?: string
  /** Only models whose price changes with the clock. */
  time_dependent?: boolean
  /** Only models that carry a declarative billing expression. */
  has_expr?: boolean
  page?: number
  /** Page size; 0 asks the backend for the whole catalog. */
  page_size?: number
}

export async function listModelPricingCatalog(
  query: ModelPricingCatalogQuery = {},
  options?: { signal?: AbortSignal }
): Promise<ModelPricingCatalogResponse> {
  const { data } = await apiClient.get<ModelPricingCatalogResponse>(
    '/admin/channels/pricing/catalog',
    {
      params: {
        q: query.q || undefined,
        provider: query.provider || undefined,
        source: query.source || undefined,
        // The backend compares these as strings, so send explicit 'true'
        // instead of relying on the axios boolean serializer.
        time_dependent: query.time_dependent ? 'true' : undefined,
        has_expr: query.has_expr ? 'true' : undefined,
        page: query.page,
        page_size: query.page_size
      },
      signal: options?.signal
    }
  )
  return data
}

// ── Manual price overrides (tuned prices) ───────────────────────────
/**
 * A per-model patch written to the override file, applied on top of the price
 * card. Effective order: group/channel explicit pricing > override > price card.
 *
 * `fields` keys are catalog JSON field names and values are **USD per token**
 * (not per million) — the page divides by 1e6 before sending and multiplies
 * back when filling its per-million inputs. `null` deletes a field, so the
 * price-card value takes over again.
 */
export type ModelPricingOverrideField =
  | 'input_cost_per_token'
  | 'output_cost_per_token'
  | 'cache_read_input_token_cost'
  | 'cache_creation_input_token_cost'
  | 'output_cost_per_image'
  | 'billing_expr'

export type ModelPricingOverrideValue = number | string | null

export interface ModelPricingOverrideEntry {
  model: string
  /** True when an admin tuned this model by hand (refresh may rewrite untuned entries). */
  tuned: boolean
  updated_at?: string
  fields: Record<string, ModelPricingOverrideValue>
}

export interface ModelPricingOverrideListResponse {
  items: ModelPricingOverrideEntry[]
  count: number
  /** Override file path on the server; shown to admins. */
  path: string
}

export interface ModelPricingOverrideSaveResponse {
  entry: ModelPricingOverrideEntry
}

export interface ModelPricingOverrideDeleteResponse {
  removed: boolean
}

export interface ModelPricingOverrideRefreshResult {
  added: number
  updated: number
  kept: number
  total: number
}

export interface ModelPricingOverrideImportResult {
  imported: number
  mode: string
}

export type ModelPricingOverrideImportMode = 'replace' | 'merge'

/**
 * Global default prices (`/admin/channels/pricing/overrides/defaults`).
 *
 * `fields` holds the non-token, global unit prices an admin tuned by hand; only
 * tuned keys appear. An empty object means nothing is tuned and every unit price
 * falls back to the price card. Units differ per key (USD per image / per second
 * / per call / per minute …), so the label carries the unit.
 */
export interface ModelPricingDefaultsResponse {
  fields: Record<string, number>
  /** Defaults override file path on the server; shown to admins. */
  path: string
}

/** Body of `PUT …/overrides/defaults`: a number tunes a key, `null` untunes it. */
export type ModelPricingDefaultsFields = Record<string, number | null>

export interface ModelPricingDefaultsSaveResponse {
  entry: ModelPricingOverrideEntry
}

export interface ModelPricingDefaultsDeleteResponse {
  removed: boolean
}
/** Filename the backend serves the override export under. */
export const PRICING_OVERRIDES_EXPORT_FILENAME = 'model_pricing_overrides.json'

/**
 * List every entry of the override file (`GET /admin/channels/pricing/overrides`)
 */
export async function listPricingOverrides(
  options?: { signal?: AbortSignal }
): Promise<ModelPricingOverrideListResponse> {
  const { data } = await apiClient.get<ModelPricingOverrideListResponse>(
    '/admin/channels/pricing/overrides',
    { signal: options?.signal }
  )
  return data
}

/**
 * Create or update one entry (`PUT /admin/channels/pricing/overrides`).
 * Values are per-token; `null` deletes the field.
 */
export async function putPricingOverride(
  model: string,
  fields: Record<string, ModelPricingOverrideValue>
): Promise<ModelPricingOverrideSaveResponse> {
  const { data } = await apiClient.put<ModelPricingOverrideSaveResponse>(
    '/admin/channels/pricing/overrides',
    { model, fields }
  )
  return data
}

/**
 * Delete one entry (`DELETE /admin/channels/pricing/overrides?model=…`)
 */
export async function deletePricingOverride(
  model: string
): Promise<ModelPricingOverrideDeleteResponse> {
  const { data } = await apiClient.delete<ModelPricingOverrideDeleteResponse>(
    '/admin/channels/pricing/overrides',
    { params: { model } }
  )
  return data
}

/**
 * Re-fetch the latest price card into the override file
 * (`POST /admin/channels/pricing/overrides/refresh`).
 *
 * `overwrite_tuned: true` rewrites every entry, including hand-tuned ones;
 * `false` keeps tuned entries and only refreshes the rest.
 */
export async function refreshPricingOverrides(
  overwriteTuned: boolean
): Promise<ModelPricingOverrideRefreshResult> {
  const { data } = await apiClient.post<ModelPricingOverrideRefreshResult>(
    '/admin/channels/pricing/overrides/refresh',
    { overwrite_tuned: overwriteTuned }
  )
  return data
}

/**
 * Download the override file (`GET /admin/channels/pricing/overrides/export`).
 * This endpoint answers with the raw file, not the `{ code, message, data }`
 * envelope, so it is fetched as a blob.
 */
export async function exportPricingOverrides(): Promise<Blob> {
  const response = await apiClient.get<Blob>('/admin/channels/pricing/overrides/export', {
    responseType: 'blob'
  })
  return response.data
}

/**
 * Import an override file (`POST /admin/channels/pricing/overrides/import`).
 * The body is the file's own JSON object (`{ "model": { … } }`), not a wrapper.
 */
export async function importPricingOverrides(
  payload: Record<string, unknown>,
  mode: ModelPricingOverrideImportMode
): Promise<ModelPricingOverrideImportResult> {
  const { data } = await apiClient.post<ModelPricingOverrideImportResult>(
    '/admin/channels/pricing/overrides/import',
    payload,
    { params: { mode }, headers: { 'Content-Type': 'application/json' } }
  )
  return data
}

/**
 * Read the global unit-price defaults (`GET …/overrides/defaults`).
 * Only hand-tuned keys come back; an empty `fields` object means none.
 */
export async function getPricingDefaults(
  options?: { signal?: AbortSignal }
): Promise<ModelPricingDefaultsResponse> {
  const { data } = await apiClient.get<ModelPricingDefaultsResponse>(
    '/admin/channels/pricing/overrides/defaults',
    { signal: options?.signal }
  )
  return data
}

/**
 * Write global unit-price defaults (`PUT …/overrides/defaults`).
 * A number tunes the key; `null` untunes it (the price card takes over again).
 */
export async function putPricingDefaults(
  fields: ModelPricingDefaultsFields
): Promise<ModelPricingDefaultsSaveResponse> {
  const { data } = await apiClient.put<ModelPricingDefaultsSaveResponse>(
    '/admin/channels/pricing/overrides/defaults',
    { fields }
  )
  return data
}

/** Drop every global unit-price default (`DELETE …/overrides/defaults`). */
export async function deletePricingDefaults(): Promise<ModelPricingDefaultsDeleteResponse> {
  const { data } = await apiClient.delete<ModelPricingDefaultsDeleteResponse>(
    '/admin/channels/pricing/overrides/defaults'
  )
  return data
}

const channelsAPI = {
  list,
  getById,
  create,
  update,
  remove,
  getModelDefaultPricing,
  syncPricingModels,
  listModelPricingCatalog,
  listPricingOverrides,
  putPricingOverride,
  deletePricingOverride,
  refreshPricingOverrides,
  exportPricingOverrides,
  importPricingOverrides,
  getPricingDefaults,
  putPricingDefaults,
  deletePricingDefaults
}

export default channelsAPI
