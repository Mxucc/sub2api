import { defineComponent } from 'vue'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  ModelPricingCatalogItem,
  ModelPricingCatalogResponse,
  ModelPricingCatalogTier,
  ModelPricingOverrideEntry
} from '@/api/admin/channels'
import zhMessages from '@/i18n/locales/zh'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelPricingView from '@/views/admin/ModelPricingView.vue'

// 模型价格总览页（只读）。重点回归「计费方式」列的三种渲染形态：
//   1) 有可识别分档 → 每档一行「档名（时段）：计价项 价格」
//   2) 有表达式但不可识别 → 原文 + 说明
//   3) 无表达式 → 「按表价」
// 下面的 items 是**演示用 fixture**（形状对齐后端契约），不是真实价格数据。
//
// 注：测试环境里 vue-i18n 只打包了 runtime（无编译器），因此这里把 useI18n 换成
// 直接读 src/i18n/locales/zh 的查表实现 —— 断言用的是真实中文文案，key 写错即失败。

const {
  listModelPricingCatalog,
  listPricingOverrides,
  putPricingOverride,
  deletePricingOverride,
  refreshPricingOverrides,
  exportPricingOverrides,
  importPricingOverrides,
  getPricingDefaults,
  putPricingDefaults,
  deletePricingDefaults,
  saveAs,
  apiGet,
  apiPost,
  apiPut,
  apiDelete,
  showSuccess,
  showError
} = vi.hoisted(() => ({
  listModelPricingCatalog: vi.fn(),
  listPricingOverrides: vi.fn(),
  putPricingOverride: vi.fn(),
  deletePricingOverride: vi.fn(),
  refreshPricingOverrides: vi.fn(),
  exportPricingOverrides: vi.fn(),
  importPricingOverrides: vi.fn(),
  getPricingDefaults: vi.fn(),
  putPricingDefaults: vi.fn(),
  deletePricingDefaults: vi.fn(),
  saveAs: vi.fn(),
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPut: vi.fn(),
  apiDelete: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channels: {
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
  }
}))

vi.mock('@/api/client', () => ({
  apiClient: { get: apiGet, post: apiPost, put: apiPut, delete: apiDelete }
}))

vi.mock('file-saver', () => ({ saveAs }))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError })
}))

function lookupMessage(key: string): unknown {
  return key.split('.').reduce<unknown>((node, segment) => {
    if (node && typeof node === 'object') {
      return (node as Record<string, unknown>)[segment]
    }
    return undefined
  }, zhMessages)
}

function translate(key: string, params?: Record<string, unknown>): string {
  const message = lookupMessage(key)
  if (typeof message !== 'string') return key
  if (!params) return message
  return message.replace(/\{(\w+)\}/g, (_, name: string) => String(params[name] ?? `{${name}}`))
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: translate,
      te: (key: string) => typeof lookupMessage(key) === 'string',
      locale: { value: 'zh' }
    })
  }
})

const AppLayoutStub = defineComponent({ template: '<div><slot /></div>' })

const TablePageLayoutStub = defineComponent({
  template:
    '<div><slot name="header" /><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
})

const PEAK_TIER: ModelPricingCatalogTier = {
  name: 'peak',
  condition: 'weekday("Asia/Shanghai") >= 1 && hour("Asia/Shanghai") >= 9',
  coefficients: { p: 0.3, cr: 0.006, c: 1.2 },
  unit: 'per_million_tokens',
  variables: ['p', 'cr', 'c'],
  time_windows: ['周一至周五 09:00-12:00, 14:00-18:00'],
  timezone: 'Asia/Shanghai'
}

const OFF_PEAK_TIER: ModelPricingCatalogTier = {
  name: 'off_peak',
  coefficients: { p: 0.15, cr: 0.003, c: 0.6 },
  unit: 'per_million_tokens',
  variables: ['p', 'cr', 'c']
}

function makeItem(overrides: Partial<ModelPricingCatalogItem> = {}): ModelPricingCatalogItem {
  return {
    model: 'demo-model',
    provider: 'demo-provider',
    mode: 'chat',
    source: 'catalog',
    input_price_per_token: 1.5e-7,
    output_price_per_token: 6e-7,
    cache_read_price_per_token: 3e-9,
    cache_creation_price_per_token: 0,
    image_input_price_per_token: 0,
    image_output_price_per_token: 0,
    input_price_per_million: 0.15,
    output_price_per_million: 0.6,
    cache_read_price_per_million: 0.003,
    cache_creation_price_per_million: 0,
    time_dependent: false,
    expr_recognized: true,
    token_pricing_absent: false,
    fallback_only: false,
    ...overrides
  }
}

const TIERED_ROW = makeItem({
  model: 'deepseek-flash',
  billing_expr: 'v1:weekday("Asia/Shanghai") >= 1 ? tier("peak", p * 0.30) : tier("off_peak", p * 0.15)',
  billing_expr_source: 'builtin',
  tiers: [PEAK_TIER, OFF_PEAK_TIER],
  time_dependent: true
})

const RAW_EXPR_ROW = makeItem({
  model: 'odd-model',
  billing_expr: 'v1:something_unrecognized(p)',
  tiers: [],
  expr_recognized: false
})

const TABLE_ROW = makeItem({ model: 'plain-model', source: 'builtin', fallback_only: true })

const IMAGE_ONLY_ROW = makeItem({
  model: 'flux-image',
  provider: '',
  token_pricing_absent: true,
  input_price_per_million: 0,
  output_price_per_million: 0,
  cache_read_price_per_million: 0
})

function catalogResponse(items: ModelPricingCatalogItem[]): ModelPricingCatalogResponse {
  return { items, total: items.length, page: 1, page_size: 50 }
}

/** Demo entries for the override file — the shape the backend contract returns. */
function overrideEntry(overrides: Partial<ModelPricingOverrideEntry> = {}): ModelPricingOverrideEntry {
  return { model: 'demo-model', tuned: true, fields: {}, ...overrides }
}

interface MountOptions {
  /** Nothing tuned yet when omitted. */
  overrides?: ModelPricingOverrideEntry[]
  overridesPath?: string
  /** Make the override list fail so the inline error strip is exercised. */
  overridesFail?: boolean
  /** Global default unit prices already tuned; `{}` by default (demo data). */
  defaults?: Record<string, number>
  defaultsPath?: string
}

async function mountView(items: ModelPricingCatalogItem[], options: MountOptions = {}) {
  const overrides = options.overrides ?? []
  listModelPricingCatalog.mockImplementation(() => Promise.resolve(catalogResponse(items)))
  getPricingDefaults.mockImplementation(() =>
    Promise.resolve({
      fields: options.defaults ?? {},
      path: options.defaultsPath ?? '/data/model_pricing_defaults.json'
    })
  )
  if (options.overridesFail) {
    listPricingOverrides.mockRejectedValue({ message: 'overrides down' })
  } else {
    listPricingOverrides.mockImplementation(() =>
      Promise.resolve({
        items: overrides,
        count: overrides.length,
        path: options.overridesPath ?? '/data/model_pricing_overrides.json'
      })
    )
  }

  const wrapper = mount(ModelPricingView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub
      }
    }
  })
  await flushPromises()
  await flushPromises()
  return wrapper
}

/** Row-scoped action button, e.g. the per-row 「微调」 button. */
async function clickRowButton(wrapper: VueWrapper, model: string, label: string) {
  const row = wrapper.findAll('tbody tr').find((candidate) => candidate.text().includes(model))
  if (!row) throw new Error(`row not rendered: ${model}`)
  const button = row.findAll('button').find((candidate) => candidate.text().includes(label))
  if (!button) throw new Error(`button not found in row ${model}: ${label}`)
  await button.trigger('click')
  await flushPromises()
}

/** Dialog controls live in `document.body` because BaseDialog teleports. */
function bodyButton(label: string): DOMWrapper<Element> {
  const button = Array.from(document.body.querySelectorAll('button')).find((candidate) =>
    (candidate.textContent ?? '').includes(label)
  )
  if (!button) throw new Error(`button not found in dialogs: ${label}`)
  return new DOMWrapper(button)
}

/** The tuning dialog is the BaseDialog whose title carries the model name. */
function tuneDialogShow(wrapper: VueWrapper): boolean {
  const dialog = wrapper
    .findAllComponents(BaseDialog)
    .find((candidate) => String(candidate.props('title')).includes('手动微调'))
  return dialog !== undefined && dialog.props('show') === true
}

async function setTuneInput(id: string, value: string) {
  const input = new DOMWrapper(document.body.querySelector<HTMLInputElement>(id)!)
  await input.setValue(value)
}

function setInputFiles(element: Element, files: File[]) {
  Object.defineProperty(element, 'files', { value: files, configurable: true })
}

function jsonFile(name: string, content: string): File {
  const file = new File([content], name, { type: 'application/json' })
  Object.defineProperty(file, 'text', { value: () => Promise.resolve(content) })
  return file
}

beforeEach(() => {
  listModelPricingCatalog.mockReset()
  listPricingOverrides.mockReset()
  putPricingOverride.mockReset()
  deletePricingOverride.mockReset()
  refreshPricingOverrides.mockReset()
  exportPricingOverrides.mockReset()
  importPricingOverrides.mockReset()
  saveAs.mockReset()
  apiGet.mockReset()
  apiPost.mockReset()
  apiPut.mockReset()
  apiDelete.mockReset()
  showSuccess.mockReset()
  showError.mockReset()
  importPricingOverrides.mockReset()
  getPricingDefaults.mockReset()
  putPricingDefaults.mockReset()
  deletePricingDefaults.mockReset()
  document.body.innerHTML = ''
})

describe('ModelPricingView', () => {
  it('renders a compact tier line per tier with translated labels and adaptive amounts', async () => {
    const wrapper = await mountView([TIERED_ROW])
    const text = wrapper.text()

    // 档名 + 人读时段（优先于条件原文）
    expect(text).toContain('高峰')
    expect(text).toContain('周一至周五 09:00-12:00, 14:00-18:00')
    // 计价变量翻译为中文标签，金额按数量级自适应（0.006 不会被显示成 0.00）
    expect(text).toContain('输入 $0.30')
    expect(text).toContain('输出 $1.20')
    expect(text).toContain('缓存命中 $0.006')
    expect(text).toContain('低谷')
    expect(text).toContain('输入 $0.15')
    expect(text).toContain('输出 $0.60')
    // 条件原文留在弹窗里，表格内不出现
    expect(text).not.toContain('hour("Asia/Shanghai")')
  })

  it('marks input/output as baseline (with tooltip) when a billing expression exists', async () => {
    const wrapper = await mountView([TIERED_ROW])
    const inputCell = wrapper.findAll('td').find((cell) => cell.text().includes('$0.15'))

    expect(inputCell).toBeTruthy()
    expect(inputCell!.text()).toContain('基线')
    expect(inputCell!.find('[title]').attributes('title')).toContain('真实价格')
    // 表头标注单位，且只标注一次
    expect(wrapper.text()).toContain('USD / 1M token')
  })

  it('shows the raw expression plus an explanation when tiers cannot be recognized', async () => {
    const wrapper = await mountView([RAW_EXPR_ROW])
    const text = wrapper.text()

    expect(text).toContain('v1:something_unrecognized(p)')
    expect(text).toContain('该表达式不是可识别的分档形态')
    expect(text).toContain('查看表达式')
  })

  it('labels expression-less models as table rates and hides the baseline tag', async () => {
    const wrapper = await mountView([TABLE_ROW])
    const text = wrapper.text()

    expect(text).toContain('按表价')
    expect(text).not.toContain('基线')
  })

  it('renders rows without token pricing as dashes plus a no-token-price flag', async () => {
    const wrapper = await mountView([IMAGE_ONLY_ROW])
    const text = wrapper.text()

    expect(text).toContain('无 token 价')
    expect(text).toContain('按表价')
    // 四个价格列显示 '-'，provider 为空时该列也是 '-'
    expect(wrapper.findAll('td').filter((cell) => cell.text() === '-').length).toBe(5)
  })

  it('opens a dialog with the full expression, its source and the tier detail', async () => {
    const wrapper = await mountView([TIERED_ROW])

    const button = wrapper
      .findAll('button')
      .find((candidate) => candidate.text().includes('查看表达式'))
    expect(button).toBeTruthy()
    await button!.trigger('click')
    await flushPromises()

    const dialogText = document.body.textContent ?? ''
    expect(dialogText).toContain('deepseek-flash · 计费方式')
    expect(dialogText).toContain('v1:weekday("Asia/Shanghai") >= 1')
    expect(dialogText).toContain('内置兜底表')
    expect(dialogText).toContain('分档明细')
    expect(dialogText).toContain('Asia/Shanghai')
    expect(dialogText).toContain('周一至周五 09:00-12:00, 14:00-18:00')
  })

  it('sends filters to the API and resets to page 1', async () => {
    const wrapper = await mountView([TIERED_ROW])
    listModelPricingCatalog.mockClear()

    const switches = wrapper.findAll('button[role="switch"]')
    expect(switches).toHaveLength(2)
    expect(switches[1].attributes('aria-label')).toBe('只看有计费表达式')

    await switches[1].trigger('click')
    await flushPromises()

    expect(listModelPricingCatalog).toHaveBeenCalledWith(
      expect.objectContaining({ has_expr: true, page: 1 }),
      expect.objectContaining({ signal: expect.anything() })
    )
  })

  it('renders the empty state with a clear-filters action when filters exclude everything', async () => {
    const wrapper = await mountView([TIERED_ROW])
    listModelPricingCatalog.mockImplementation(() => Promise.resolve(catalogResponse([])))

    // 搜索防抖：先用假定时器推进 300ms，再放行请求
    vi.useFakeTimers()
    try {
      await wrapper.find('input[type="text"]').setValue('no-such-model')
      await vi.advanceTimersByTimeAsync(400)
    } finally {
      vi.useRealTimers()
    }
    await flushPromises()
    await flushPromises()

    expect(listModelPricingCatalog).toHaveBeenCalledWith(
      expect.objectContaining({ q: 'no-such-model', page: 1 }),
      expect.objectContaining({ signal: expect.anything() })
    )

    expect(wrapper.text()).toContain('只看随时刻变化')
    expect(wrapper.text()).toContain('没有匹配的模型')
    expect(wrapper.text()).toContain('清除筛选')
  })

  it('surfaces a load failure with a retry action instead of an empty catalog', async () => {
    listModelPricingCatalog.mockRejectedValue({ message: 'boom' })
    const wrapper = mount(ModelPricingView, {
      global: {
        stubs: { AppLayout: AppLayoutStub, TablePageLayout: TablePageLayoutStub }
      }
    })
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('加载模型价格失败')
    expect(wrapper.text()).toContain('boom')
    expect(wrapper.text()).toContain('重试')
    expect(wrapper.text()).not.toContain('价格目录为空')
  })
})

// 手动微调（覆盖文件）与导入/导出。overrides fixture 同样是演示数据。
describe('ModelPricingView pricing overrides', () => {
  it('badges only the rows the override file marks as manually tuned', async () => {
    const wrapper = await mountView([TIERED_ROW, TABLE_ROW], {
      overrides: [
        overrideEntry({
          model: 'deepseek-flash',
          fields: { input_cost_per_token: 1.5e-7 },
          updated_at: '2025-01-02T03:04:05Z'
        }),
        overrideEntry({ model: 'plain-model', tuned: false, fields: {} })
      ]
    })

    const rows = wrapper.findAll('tbody tr')
    const tunedRow = rows.find((row) => row.text().includes('deepseek-flash'))!
    const refreshedRow = rows.find((row) => row.text().includes('plain-model'))!

    expect(tunedRow.text()).toContain('已微调')
    expect(refreshedRow.text()).not.toContain('已微调')
    // 覆盖文件路径展示在表格下方
    expect(wrapper.text()).toContain('/data/model_pricing_overrides.json')
    expect(wrapper.text()).toContain('分组/渠道显式定价 ＞ 手动微调 ＞ 默认价卡')
  })

  it('fills the tuning form in USD per 1M tokens from per-token override values', async () => {
    const wrapper = await mountView([TIERED_ROW], {
      overrides: [
        overrideEntry({
          model: 'deepseek-flash',
          fields: {
            input_cost_per_token: 1.5e-7,
            output_cost_per_token: 6e-7,
            cache_read_input_token_cost: 3e-9,
            cache_creation_input_token_cost: 1e-6,
            billing_expr: 'v1:tuned_expr(p)'
          },
          updated_at: '2025-01-02T03:04:05Z'
        })
      ]
    })

    await clickRowButton(wrapper, 'deepseek-flash', '微调')

    const field = (id: string) => document.body.querySelector<HTMLInputElement>(id)?.value
    expect(field('#tune-field-input')).toBe('0.15')
    expect(field('#tune-field-output')).toBe('0.6')
    expect(field('#tune-field-cacheRead')).toBe('0.003')
    expect(field('#tune-field-cacheWrite')).toBe('1')
    expect(document.body.querySelector<HTMLTextAreaElement>('textarea')?.value).toBe('v1:tuned_expr(p)')
    // 价卡值作为参照展示（0.15 / 0.60 …）与重新推导的输入框并存
    expect(document.body.textContent).toContain('价卡值')
  })

  it('submits per-token values and reports the saved model', async () => {
    putPricingOverride.mockResolvedValue({ entry: overrideEntry({ model: 'deepseek-flash' }) })
    const wrapper = await mountView([TIERED_ROW])

    await clickRowButton(wrapper, 'deepseek-flash', '微调')
    await setTuneInput('#tune-field-input', '0.15')
    await new DOMWrapper(document.body.querySelector<HTMLFormElement>('#tune-pricing-form')!).trigger('submit')
    await flushPromises()

    expect(putPricingOverride).toHaveBeenCalledWith('deepseek-flash', { input_cost_per_token: 1.5e-7 })
    expect(showSuccess).toHaveBeenCalledWith(expect.stringContaining('已保存微调：deepseek-flash'))
    // 写操作后重新拉取目录与微调表
    // mount = catalog + provider list, then one reload after the write
    expect(listModelPricingCatalog).toHaveBeenCalledTimes(3)
    expect(listPricingOverrides).toHaveBeenCalledTimes(2)
  })

  it('refuses to save four blank prices so the admin is not sent into a 400', async () => {
    putPricingOverride.mockResolvedValue({ entry: overrideEntry({ model: 'deepseek-flash' }) })
    const wrapper = await mountView([TIERED_ROW])

    await clickRowButton(wrapper, 'deepseek-flash', '微调')
    await new DOMWrapper(document.body.querySelector<HTMLFormElement>('#tune-pricing-form')!).trigger('submit')
    await flushPromises()

    expect(putPricingOverride).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('至少填写一项价格')
  })

  it('runs the fetch with overwrite_tuned true/false and reports the counts', async () => {
    refreshPricingOverrides.mockResolvedValue({ added: 1, updated: 2, kept: 3, total: 6 })
    const wrapper = await mountView([TIERED_ROW], { overrides: [overrideEntry({ model: 'deepseek-flash' })] })

    const openDialog = wrapper
      .findAll('button')
      .find((candidate) => candidate.text().includes('获取最新价格'))!

    await openDialog.trigger('click')
    await flushPromises()
    expect(document.body.textContent).toContain('覆盖全部（含微调）')
    expect(document.body.textContent).toContain('保留微调')

    await bodyButton('覆盖全部（含微调）').trigger('click')
    await flushPromises()
    expect(refreshPricingOverrides).toHaveBeenLastCalledWith(true)
    expect(showSuccess).toHaveBeenCalledWith(expect.stringContaining('新增 1'))
    expect(listPricingOverrides).toHaveBeenCalledTimes(2)

    await openDialog.trigger('click')
    await flushPromises()
    await bodyButton('保留微调').trigger('click')
    await flushPromises()
    expect(refreshPricingOverrides).toHaveBeenLastCalledWith(false)
  })

  it('deletes a tuning with the model as its query parameter', async () => {
    deletePricingOverride.mockResolvedValue({ removed: true })
    const wrapper = await mountView([TIERED_ROW], {
      overrides: [overrideEntry({ model: 'deepseek-flash', fields: { input_cost_per_token: 1.5e-7 } })]
    })

    await clickRowButton(wrapper, 'deepseek-flash', '微调')
    await bodyButton('恢复价卡价').trigger('click')
    await flushPromises()
    await bodyButton('确认恢复').trigger('click')
    await flushPromises()

    expect(deletePricingOverride).toHaveBeenCalledWith('deepseek-flash')
    expect(showSuccess).toHaveBeenCalledWith(expect.stringContaining('已恢复价卡价：deepseek-flash'))
    expect(listPricingOverrides).toHaveBeenCalledTimes(2)
  })

  it('imports the file itself with the selected mode', async () => {
    importPricingOverrides.mockResolvedValue({ imported: 2, mode: 'replace' })
    const wrapper = await mountView([TIERED_ROW])

    await wrapper.findAll('button').find((candidate) => candidate.text().includes('导入'))!.trigger('click')
    await flushPromises()

    const payload = { 'deepseek-flash': { input_cost_per_token: 1.5e-7 } }
    const fileInput = new DOMWrapper(document.body.querySelector<HTMLInputElement>('input[type="file"]')!)
    setInputFiles(fileInput.element, [jsonFile('model_pricing_overrides.json', JSON.stringify(payload))])
    await fileInput.trigger('change')
    await flushPromises()

    // 默认「合并」，切到「替换全部」后导入
    const radios = document.body.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    expect(radios).toHaveLength(2)
    radios[0].click()
    await flushPromises()

    await new DOMWrapper(document.body.querySelector<HTMLFormElement>('#import-overrides-form')!).trigger('submit')
    await flushPromises()

    expect(importPricingOverrides).toHaveBeenCalledWith(payload, 'replace')
    expect(showSuccess).toHaveBeenCalledWith(expect.stringContaining('导入完成：2 条'))
    expect(listPricingOverrides).toHaveBeenCalledTimes(2)
  })

  it('reports a malformed import file without calling the API', async () => {
    const wrapper = await mountView([TIERED_ROW])

    await wrapper.findAll('button').find((candidate) => candidate.text().includes('导入'))!.trigger('click')
    await flushPromises()

    const fileInput = new DOMWrapper(document.body.querySelector<HTMLInputElement>('input[type="file"]')!)
    setInputFiles(fileInput.element, [jsonFile('broken.json', 'not json')])
    await fileInput.trigger('change')
    await flushPromises()
    await new DOMWrapper(document.body.querySelector<HTMLFormElement>('#import-overrides-form')!).trigger('submit')
    await flushPromises()

    expect(importPricingOverrides).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('文件不是合法的 JSON')
  })

  it('exports the override file through the blob endpoint', async () => {
    const blob = new Blob(['{}'], { type: 'application/json' })
    exportPricingOverrides.mockResolvedValue(blob)
    const wrapper = await mountView([TIERED_ROW])

    await wrapper.findAll('button').find((candidate) => candidate.text().includes('导出'))!.trigger('click')
    await flushPromises()

    expect(saveAs).toHaveBeenCalledWith(blob, 'model_pricing_overrides.json')
    expect(showSuccess).toHaveBeenCalledWith('覆盖文件已导出')
  })
})

// 覆盖文件接口的传输层契约：查询参数与请求体形状。
describe('pricing override API contract', () => {
  it('deletes with the model as query parameter', async () => {
    const channels = await import('@/api/admin/channels')
    apiDelete.mockResolvedValue({ data: { removed: true } })

    await channels.deletePricingOverride('deepseek-flash')

    expect(apiDelete).toHaveBeenCalledWith('/admin/channels/pricing/overrides', {
      params: { model: 'deepseek-flash' }
    })
  })

  it('imports with the mode as query parameter and the file as the raw body', async () => {
    const channels = await import('@/api/admin/channels')
    apiPost.mockResolvedValue({ data: { imported: 1, mode: 'merge' } })
    const payload = { 'deepseek-flash': { input_cost_per_token: 1.5e-7 } }

    await channels.importPricingOverrides(payload, 'merge')

    expect(apiPost).toHaveBeenCalledWith('/admin/channels/pricing/overrides/import', payload, {
      params: { mode: 'merge' },
      headers: { 'Content-Type': 'application/json' }
    })
  })

  it('refreshes with the overwrite flag and exports as a blob download', async () => {
    const channels = await import('@/api/admin/channels')
    apiPost.mockResolvedValue({ data: { added: 0, updated: 0, kept: 0, total: 0 } })
    apiGet.mockResolvedValue({ data: new Blob(['{}'], { type: 'application/json' }) })

    await channels.refreshPricingOverrides(true)
    expect(apiPost).toHaveBeenCalledWith('/admin/channels/pricing/overrides/refresh', {
      overwrite_tuned: true
    })

    await channels.exportPricingOverrides()
    expect(apiGet).toHaveBeenCalledWith('/admin/channels/pricing/overrides/export', {
      responseType: 'blob'
    })
  })
})

// 键盘与表单接线：每个价格输入都有 label，保存是真 submit，表达式是等宽字体。
describe('ModelPricingView tuning dialog wiring', () => {
  it('keeps the tuning dialog wired for keyboard and form submission', async () => {
    const wrapper = await mountView([TIERED_ROW], {
      overrides: [overrideEntry({ model: 'deepseek-flash', fields: { input_cost_per_token: 1.5e-7 } })]
    })
    await clickRowButton(wrapper, 'deepseek-flash', '微调')

    for (const key of ['input', 'output', 'cacheRead', 'cacheWrite']) {
      const input = document.body.querySelector<HTMLInputElement>(`#tune-field-${key}`)
      expect(input).not.toBeNull()
      expect(input!.getAttribute('inputmode')).toBe('decimal')
      const label = document.body.querySelector(`label[for="tune-field-${key}"]`)
      expect(label?.textContent?.trim()).toBeTruthy()
    }

    // 单位与价卡参照
    expect(document.body.textContent).toContain('USD / 1M token')
    expect(document.body.textContent).toContain('价卡值 $0.15')
    expect(document.body.textContent).toContain('价卡值 $0.003')

    const save = Array.from(document.body.querySelectorAll('button')).find((button) =>
      (button.textContent ?? '').includes('保存微调')
    )!
    expect(save.getAttribute('type')).toBe('submit')
    expect(save.getAttribute('form')).toBe('tune-pricing-form')
    expect(document.body.querySelector('#tune-pricing-form textarea')?.className).toContain('font-mono')
  })

  it('cancels the tuning dialog without writing anything', async () => {
    const wrapper = await mountView([TIERED_ROW], {
      overrides: [overrideEntry({ model: 'deepseek-flash', fields: { input_cost_per_token: 1.5e-7 } })]
    })
    await clickRowButton(wrapper, 'deepseek-flash', '微调')
    expect(tuneDialogShow(wrapper)).toBe(true)

    await bodyButton('取消').trigger('click')
    await flushPromises()

    expect(tuneDialogShow(wrapper)).toBe(false)
    expect(putPricingOverride).not.toHaveBeenCalled()
  })

  it('keeps prices usable and offers a retry when the override list fails', async () => {
    const wrapper = await mountView([TIERED_ROW], { overridesFail: true })

    expect(wrapper.text()).toContain('加载微调列表失败')
    expect(wrapper.text()).toContain('overrides down')
    // 价格表本身仍然可用，微调入口也还在
    expect(wrapper.text()).toContain('deepseek-flash')
    expect(wrapper.findAll('button').some((button) => button.text().includes('微调'))).toBe(true)

    await wrapper.findAll('button').find((button) => button.text().includes('重新加载微调'))!.trigger('click')
    await flushPromises()
    expect(listPricingOverrides).toHaveBeenCalledTimes(2)
  })

  it('ignores a second fetch click while the first one is in flight', async () => {
    let resolveRefresh: (value: unknown) => void = () => {}
    refreshPricingOverrides.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRefresh = resolve
        })
    )
    const wrapper = await mountView([TIERED_ROW])

    await wrapper.findAll('button').find((button) => button.text().includes('获取最新价格'))!.trigger('click')
    await flushPromises()
    await bodyButton('保留微调').trigger('click')
    await flushPromises()

    expect(refreshPricingOverrides).toHaveBeenCalledTimes(1)
    expect(bodyButton('保留微调').attributes('disabled')).toBeDefined()

    await bodyButton('保留微调').trigger('click')
    await flushPromises()
    expect(refreshPricingOverrides).toHaveBeenCalledTimes(1)

    resolveRefresh({ added: 0, updated: 0, kept: 0, total: 0 })
    await flushPromises()
  })
})

// 全局默认价（非 token）区块：图片/视频/搜索/音频/按次。fixture 为演示数据。
describe('ModelPricingView global defaults', () => {
  async function openDefaultsDialog(wrapper: VueWrapper) {
    const button = wrapper.findAll('button').find((candidate) =>
      candidate.text().includes('全局默认价')
    )!
    await button.trigger('click')
    await flushPromises()
    await flushPromises()
  }

  function defaultsInput(key: string): HTMLInputElement {
    const input = document.body.querySelector<HTMLInputElement>(`#defaults-field-${key}`)
    if (!input) throw new Error(`defaults input not rendered: ${key}`)
    return input
  }

  async function submitDefaultsForm() {
    const form = document.body.querySelector<HTMLFormElement>('#defaults-pricing-form')!
    await new DOMWrapper(form).trigger('submit')
    await flushPromises()
  }

  function clearFieldButton(label: string): DOMWrapper<Element> {
    const button = Array.from(document.body.querySelectorAll('button')).find((candidate) =>
      (candidate.getAttribute('aria-label') ?? '').includes(label)
    )
    if (!button) throw new Error(`clear button not found: ${label}`)
    return new DOMWrapper(button)
  }

  it('fetches the defaults on mount and marks only the tuned fields', async () => {
    const wrapper = await mountView([TABLE_ROW], {
      defaults: { image_price_1k: 0.04, per_request_price: 0.002, video_price_1080p: 1.5e-3 }
    })

    expect(getPricingDefaults).toHaveBeenCalledTimes(1)

    await openDefaultsDialog(wrapper)
    expect(document.body.textContent).toContain('全局默认价')
    expect(document.body.textContent).toContain('USD/张')
    expect(document.body.textContent).toContain('USD/秒')
    expect(document.body.querySelector('#defaults-pricing-form')).not.toBeNull()

    expect(defaultsInput('image_price_1k').value).toBe('0.04')
    expect(defaultsInput('per_request_price').value).toBe('0.002')
    // 科学计数法的小值不会被截断
    expect(Number(defaultsInput('video_price_1080p').value)).toBeCloseTo(1.5e-3, 12)
    // 未微调的字段留空，而不是 0
    expect(defaultsInput('image_price_2k').value).toBe('')

    const tunedBadges = document.body.querySelectorAll('#defaults-pricing-form .tag-primary')
    expect(tunedBadges).toHaveLength(3)
    expect(document.body.textContent).toContain('已微调')
  })

  it('sends only the fields the admin filled in and refetches afterwards', async () => {
    putPricingDefaults.mockResolvedValue({ entry: overrideEntry({ model: '__defaults__' }) })
    const wrapper = await mountView([TABLE_ROW])
    await openDefaultsDialog(wrapper)

    await new DOMWrapper(defaultsInput('image_price_2k')).setValue('0.15')
    await submitDefaultsForm()

    expect(putPricingDefaults).toHaveBeenCalledWith({ image_price_2k: 0.15 })
    expect(showSuccess).toHaveBeenCalledWith('全局默认价已保存')
    // 保存后重新拉取，保证「已微调」状态刷新
    expect(getPricingDefaults).toHaveBeenCalledTimes(3)
  })

  it('untunes a single field by submitting null', async () => {
    putPricingDefaults.mockResolvedValue({ entry: overrideEntry({ model: '__defaults__' }) })
    const wrapper = await mountView([TABLE_ROW], { defaults: { image_price_1k: 0.04 } })
    await openDefaultsDialog(wrapper)

    expect(document.body.textContent).toContain('已微调')
    await clearFieldButton('1K 图片').trigger('click')
    await flushPromises()
    expect(defaultsInput('image_price_1k').value).toBe('')

    await submitDefaultsForm()
    expect(putPricingDefaults).toHaveBeenCalledWith({ image_price_1k: null })
  })

  it('rejects a non-numeric value before calling the API', async () => {
    const wrapper = await mountView([TABLE_ROW])
    await openDefaultsDialog(wrapper)

    await new DOMWrapper(defaultsInput('image_price_1k')).setValue('not-a-number')
    await submitDefaultsForm()

    expect(putPricingDefaults).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('需要是非负数字')
  })

  it('surfaces the backend error when the API rejects the payload', async () => {
    putPricingDefaults.mockRejectedValue({ message: 'unknown pricing field' })
    const wrapper = await mountView([TABLE_ROW])
    await openDefaultsDialog(wrapper)

    await new DOMWrapper(defaultsInput('image_price_1k')).setValue('0.04')
    await submitDefaultsForm()

    expect(document.body.textContent).toContain('unknown pricing field')
  })

  it('clears every tuning through DELETE after the confirm step', async () => {
    deletePricingDefaults.mockResolvedValue({ removed: true })
    const wrapper = await mountView([TABLE_ROW], { defaults: { image_price_1k: 0.04 } })
    await openDefaultsDialog(wrapper)

    await bodyButton('清除全部微调').trigger('click')
    await flushPromises()
    expect(document.body.textContent).toContain('确认清除全部全局默认价微调')

    await bodyButton('确认清除').trigger('click')
    await flushPromises()

    expect(deletePricingDefaults).toHaveBeenCalledTimes(1)
    expect(showSuccess).toHaveBeenCalledWith('已清除全部全局默认价微调')
    // 清除后再次拉取，微调标记归零
    expect(getPricingDefaults).toHaveBeenCalledTimes(3)
  })

  it('exposes output_cost_per_image per model but never the global keys', async () => {
    const wrapper = await mountView([TIERED_ROW])

    await clickRowButton(wrapper, 'deepseek-flash', '微调')

    const imageField = document.body.querySelector('#tune-field-imageOutput')
    expect(imageField).not.toBeNull()
    expect(document.body.textContent).toContain('USD / 张')
    expect(document.body.textContent).toContain('output_cost_per_image')

    // 全局键只属于「全局默认价」区块，逐个模型微调里一个都不能出现
    for (const key of [
      'image_price_1k',
      'image_price_2k',
      'image_price_4k',
      'video_price_480p',
      'video_price_720p',
      'video_price_1080p',
      'web_search_price_per_call',
      'search_price_per_1k',
      'audio_realtime_price_per_min',
      'audio_tts_price_per_million_chars',
      'audio_stt_price_per_hour',
      'per_request_price'
    ]) {
      expect(document.body.querySelector(`#tune-field-${key}`)).toBeNull()
    }
  })

  it('submits the per-image tuning without the per-million conversion', async () => {
    putPricingOverride.mockResolvedValue({ entry: overrideEntry({ model: 'deepseek-flash' }) })
    const wrapper = await mountView([TIERED_ROW])

    await clickRowButton(wrapper, 'deepseek-flash', '微调')
    await new DOMWrapper(document.body.querySelector<HTMLInputElement>('#tune-field-imageOutput')!).setValue('0.04')
    await new DOMWrapper(document.body.querySelector<HTMLFormElement>('#tune-pricing-form')!).trigger('submit')
    await flushPromises()

    expect(putPricingOverride).toHaveBeenCalledWith('deepseek-flash', { output_cost_per_image: 0.04 })
  })

  it('keeps the defaults dialog usable and offers a retry when the load fails', async () => {
    const wrapper = await mountView([TABLE_ROW])
    // The retry path is what fails first; mount already loaded once successfully.
    getPricingDefaults.mockRejectedValue({ message: 'defaults down' })
    await openDefaultsDialog(wrapper)

    expect(document.body.textContent).toContain('加载全局默认价失败')
    expect(document.body.textContent).toContain('defaults down')

    getPricingDefaults.mockImplementation(() =>
      Promise.resolve({ fields: { image_price_1k: 0.04 }, path: '/data/model_pricing_defaults.json' })
    )
    await bodyButton('重新加载').trigger('click')
    await flushPromises()

    expect(document.body.textContent).toContain('已微调')
    expect(defaultsInput('image_price_1k').value).toBe('0.04')
  })
})
