import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlazaModelPricingTable from '../PlazaModelPricingTable.vue'
import type { PlazaBillingExpr, PlazaModel } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function tokenModel(overrides: Partial<PlazaModel> = {}): PlazaModel {
  return {
    name: 'claude-sonnet',
    platform: 'anthropic',
    pricing: {
      billing_mode: 'token',
      input_price: 3e-6,
      output_price: 1.5e-5,
      cache_write_price: 3.75e-6,
      cache_read_price: 3e-7,
      image_input_price: null,
      image_output_price: null,
      per_request_price: null,
      intervals: []
    },
    official_pricing: {
      input_price: 3e-6,
      output_price: 1.5e-5,
      cache_write_price: 3.75e-6,
      cache_write_1h_price: 6e-6,
      cache_read_price: 3e-7
    },
    ...overrides
  }
}

function mountTable(
  models: PlazaModel[],
  rateMultiplier: number,
  userRateMultiplier?: number | null,
  extraProps?: {
    imageRateIndependent?: boolean
    imageRateMultiplier?: number | null
    peakWindow?: string
    peakRateMultiplier?: number | null
  }
) {
  return mount(PlazaModelPricingTable, {
    props: { models, rateMultiplier, userRateMultiplier: userRateMultiplier ?? null, ...extraProps }
  })
}

describe('PlazaModelPricingTable', () => {
  it('倍率为 1 时展示渠道单价原值($/1M),价格保底 2 位小数', () => {
    const wrapper = mountTable([tokenModel()], 1)
    const text = wrapper.text()
    expect(text).toContain('$3.00')
    expect(text).toContain('$15.00')
    // 缓存写 / 读(超过 2 位小数原样保留)
    expect(text).toContain('$3.75')
    expect(text).toContain('$0.30')
    // 倍率列
    expect(text).toContain('1x')
  })

  it('shows all configured reasoning multipliers in level order', () => {
    const model = tokenModel()
    model.pricing!.reasoning_effort_multipliers = { max: 3, none: 0.5, high: 1.5 }
    const wrapper = mountTable([model], 1)

    const badges = wrapper.findAll('[data-reasoning-effort]')
    expect(badges.map(badge => badge.attributes('data-reasoning-effort'))).toEqual(['none', 'high', 'max'])
    expect(badges.every(badge => badge.attributes('title') === 'modelPlaza.table.reasoningMultiplierHint')).toBe(true)
    expect(wrapper.text()).toContain('modelPlaza.table.reasoningMultiplierBadge')
  })

  it('does not show automatic reasoning charges for an unconfigured Fable model', () => {
    const wrapper = mountTable([tokenModel({ name: 'claude-fable-5-1' })], 1)
    expect(wrapper.find('[data-reasoning-effort]').exists()).toBe(false)
  })

  it('omits invalid or unsupported multipliers from display', () => {
    const model = tokenModel()
    model.pricing!.reasoning_effort_multipliers = { max: 0, high: Infinity, unknown: 2, low: 1 }
    const wrapper = mountTable([model], 1)
    expect(wrapper.findAll('[data-reasoning-effort]').map(badge => badge.attributes('data-reasoning-effort'))).toEqual(['low'])
  })

  it('倍率 ≠ 1 时价格列为折后实付价,官方价列保持原价', () => {
    const wrapper = mountTable([tokenModel()], 0.5)
    const text = wrapper.text()
    // 实付 = 3 × 0.5 / 15 × 0.5
    expect(text).toContain('$1.50')
    expect(text).toContain('$7.50')
    // 官方价原值仍在(官方列不乘倍率)
    expect(text).toContain('$3.00')
    expect(text).toContain('$15.00')
    expect(text).toContain('0.5x')
  })

  it('用户专属倍率覆盖分组倍率,并划线展示原倍率', () => {
    const wrapper = mountTable([tokenModel()], 1, 0.8)
    const text = wrapper.text()
    // 实付按 0.8:3 × 0.8 = 2.4
    expect(text).toContain('$2.40')
    expect(text).toContain('$12.00')
    // 倍率列:原倍率划线 + 专属倍率
    const struck = wrapper.find('td .line-through')
    expect(struck.exists()).toBe(true)
    expect(struck.text()).toBe('1x')
    expect(text).toContain('0.8x')
  })

  it('模型按官方输出价从高到低排序,无官方价的排最后', () => {
    const expensive = tokenModel({
      name: 'model-expensive',
      official_pricing: {
        input_price: 1e-5,
        output_price: 7.5e-5,
        cache_write_price: null,
        cache_write_1h_price: null,
        cache_read_price: null
      }
    })
    const cheap = tokenModel({
      name: 'model-cheap',
      official_pricing: {
        input_price: 1e-6,
        output_price: 5e-6,
        cache_write_price: null,
        cache_write_1h_price: null,
        cache_read_price: null
      }
    })
    const noOfficial = tokenModel({ name: 'model-no-official', official_pricing: null })

    const wrapper = mountTable([cheap, noOfficial, expensive], 1)
    const names = wrapper.findAll('tbody tr').map((tr) => tr.find('td').text())
    expect(names).toEqual(['model-expensive', 'model-cheap', 'model-no-official'])
  })

  it('官方输出价相同时按模型名降序(新版本号在前)', () => {
    const older = tokenModel({ name: 'gpt-5.5' })
    const newer = tokenModel({ name: 'gpt-5.6-sol' })

    const wrapper = mountTable([older, newer], 1)
    const names = wrapper.findAll('tbody tr').map((tr) => tr.find('td').text())
    expect(names).toEqual(['gpt-5.6-sol', 'gpt-5.5'])
  })

  it('按图片/按次计费的模型沉到末尾,不与 token 模型按官方价混排', () => {
    // 官方输出价 $10,介于下面两个 token 模型之间,但因计费模式不同应排最后
    const image = tokenModel({
      name: 'gpt-image-2',
      pricing: {
        billing_mode: 'image',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: 0.002,
        intervals: []
      },
      official_pricing: {
        input_price: 5e-6,
        output_price: 1e-5,
        cache_write_price: null,
        cache_write_1h_price: null,
        cache_read_price: 1.25e-6
      }
    })
    const pricier = tokenModel({
      name: 'gpt-5.6-terra',
      official_pricing: {
        input_price: 2.5e-6,
        output_price: 1.5e-5,
        cache_write_price: null,
        cache_write_1h_price: null,
        cache_read_price: null
      }
    })
    const cheaper = tokenModel({
      name: 'gpt-5.6-luna',
      official_pricing: {
        input_price: 1e-6,
        output_price: 6e-6,
        cache_write_price: null,
        cache_write_1h_price: null,
        cache_read_price: null
      }
    })

    const wrapper = mountTable([pricier, image, cheaper], 1)
    const names = wrapper.findAll('tbody tr').map((tr) => tr.find('td').text())
    expect(names[0]).toBe('gpt-5.6-terra')
    expect(names[1]).toBe('gpt-5.6-luna')
    // 首列含「按图片计费」徽章文本,只断言模型名
    expect(names[2]).toContain('gpt-image-2')
  })

  it('两级表头:实付区与官方区各拆输入/输出/缓存列', () => {
    const wrapper = mountTable([tokenModel()], 1)
    const text = wrapper.text()
    expect(text).toContain('modelPlaza.table.paidPrice')
    expect(text).toContain('modelPlaza.table.officialPrice')
    // token 行:模型 + 实付 3 列 + 官方 3 列 + 倍率
    expect(wrapper.findAll('tbody td')).toHaveLength(8)
  })

  it('官方价包含 1h 缓存写入价;official_pricing 为 null 时官方三列显示 -', () => {
    const withOfficial = mountTable([tokenModel()], 1)
    expect(withOfficial.text()).toContain('$6.00')
    expect(withOfficial.text()).toContain('(1h')

    const withoutOfficial = mountTable([tokenModel({ official_pricing: null })], 1)
    const cells = withoutOfficial.findAll('tbody td')
    // 官方 输入/输出/缓存 三列均为 -
    expect(cells[4].text().trim()).toBe('-')
    expect(cells[5].text().trim()).toBe('-')
    expect(cells[6].text().trim()).toBe('-')
  })

  it('实付价分别展示自定义 5m 与 1h 缓存写入价', () => {
    const model = tokenModel()
    model.pricing!.cache_write_1h_price = 7e-6

    const wrapper = mountTable([model], 1)
    expect(wrapper.text()).toContain('$3.75')
    expect(wrapper.text()).toContain('$7.00')
    expect(wrapper.text()).toContain('(1h')
  })

  it('per_request 模型按单次价 × 倍率展示,官方价列显示 -', () => {
    const model = tokenModel({
      name: 'search-tool',
      pricing: {
        billing_mode: 'per_request',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: 0.04,
        intervals: []
      },
      official_pricing: null
    })
    const wrapper = mountTable([model], 0.5)
    const text = wrapper.text()
    // 0.04 × 0.5 = 0.02,scale=1
    expect(text).toContain('$0.02')
    expect(text).toContain('modelPlaza.table.perRequest')
    // 单位后缀跟在价格后(按次 → / 次)
    expect(text).toContain('modelPlaza.table.perUnitRequest')
  })

  it('token 模型阶梯定价内联进输入/输出列,按倍率折算', () => {
    const model = tokenModel({
      pricing: {
        billing_mode: 'token',
        input_price: 3e-6,
        output_price: 1.5e-5,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: [
          {
            min_tokens: 0,
            max_tokens: 200000,
            tier_label: '',
            input_price: 3e-6,
            output_price: 1.5e-5,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: null
          },
          {
            min_tokens: 200000,
            max_tokens: null,
            tier_label: '',
            input_price: 6e-6,
            output_price: 3e-5,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: null
          }
        ]
      }
    })
    const wrapper = mountTable([model], 0.5)
    const text = wrapper.text()
    // 区间标签按 token 数生成
    expect(text).toContain('≤200K')
    expect(text).toContain('>200K')
    // 折后:输入 1.5 / 3,输出 7.5 / 15
    expect(text).toContain('$1.50')
    expect(text).toContain('$7.50')
    expect(text).toContain('$15.00')
  })

  it('仅配置区间倍率时按基础价格展示 input/output/cache 各档价格', () => {
    const model = tokenModel({
      pricing: {
        billing_mode: 'token',
        input_price: 10e-6,
        output_price: 50e-6,
        cache_write_price: 12.5e-6,
        cache_write_1h_price: 12.5e-6,
        cache_read_price: 2e-6,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: [{
          min_tokens: 272000,
          max_tokens: null,
          tier_label: '>272K',
          input_price: null,
          output_price: null,
          cache_write_price: null,
          cache_write_1h_price: null,
          cache_read_price: null,
          input_multiplier: 2,
          output_multiplier: 1.5,
          cache_write_multiplier: 2,
          cache_read_multiplier: 2,
          per_request_price: null
        }]
      },
      official_pricing: null
    })

    const text = mountTable([model], 1).text()
    expect(text).toContain('$20.00')
    expect(text).toContain('$75.00')
    expect(text).toContain('$25.00')
    expect(text).toContain('$4.00')
    model.pricing!.intervals.unshift({
      ...model.pricing!.intervals[0], min_tokens: 0, max_tokens: 272000,
      tier_label: '<=272K', cache_write_multiplier: null, cache_read_multiplier: null
    })
    const cells = mountTable([model], 1).findAll('tbody td')
    expect(cells[3].text()).toContain('$12.50')
    expect(cells[3].text()).toContain('$2.00')
    expect(cells[3].text()).toContain('$25.00')
    expect(cells[3].text()).toContain('$4.00')
  })

  it('生图独立倍率开启时,按图价格 × 独立倍率,不乘分组倍率;倍率列展示独立倍率', () => {
    const model = tokenModel({
      name: 'gpt-image-2',
      pricing: {
        billing_mode: 'image',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: [
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '1K',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: 0.02
          }
        ]
      },
      official_pricing: null
    })
    const wrapper = mountTable([model], 0.1, null, {
      imageRateIndependent: true,
      imageRateMultiplier: 1
    })
    const text = wrapper.text()
    // 0.02 × 1(独立倍率),而非 0.02 × 0.1
    expect(text).toContain('$0.02')
    expect(text).not.toContain('$0.002')
    // 倍率列展示独立倍率 1x,而非分组倍率 0.1x
    const rateCell = wrapper.findAll('tbody tr td').at(-1)!
    expect(rateCell.text()).toBe('1x')
  })

  it('生图独立倍率关闭时,按图价格仍乘分组/专属生效倍率', () => {
    const model = tokenModel({
      name: 'gpt-image-2',
      pricing: {
        billing_mode: 'image',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: 0.2,
        intervals: []
      },
      official_pricing: null
    })
    const wrapper = mountTable([model], 0.1, null, { imageRateIndependent: false })
    const text = wrapper.text()
    expect(text).toContain('$0.02')
    const rateCell = wrapper.findAll('tbody tr td').at(-1)!
    expect(rateCell.text()).toBe('0.1x')
  })

  it('按图模型主行展示阶梯芯片,不把 image_output_price(每 token)当按次价', () => {
    const model = tokenModel({
      name: 'gpt-image-2',
      pricing: {
        billing_mode: 'image',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        // 每 token 图片输出价:不应被当作按次单价展示
        image_output_price: 3e-5,
        per_request_price: null,
        intervals: [
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '1K',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: 0.01
          },
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '2K',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: 0.02
          }
        ]
      },
      official_pricing: null
    })
    const wrapper = mountTable([model], 0.1)
    const text = wrapper.text()
    expect(text).toContain('modelPlaza.table.perImage')
    // 芯片:1K $0.001 / 2K $0.002,单位后缀内嵌(按图 → / 张)
    expect(text).toContain('1K')
    expect(text).toContain('$0.001')
    expect(text).toContain('2K')
    expect(text).toContain('$0.002')
    expect(text).toContain('modelPlaza.table.perUnitImage')
    // 旧 bug:image_output_price × 0.1 = 0.000003 被当按次价
    expect(text).not.toContain('$0.000003')
  })

  it('Composite 分组中相同模型名按具体平台分别展示徽章', () => {
    const anthropic = tokenModel({ name: 'shared-model', platform: 'anthropic' })
    const openai = tokenModel({ name: 'shared-model', platform: 'openai' })
    const wrapper = mount(PlazaModelPricingTable, {
      props: {
        models: [anthropic, openai],
        platform: 'composite',
        rateMultiplier: 1
      }
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows.map((row) => row.find('td').text())).toEqual([
      'shared-modelAnthropic',
      'shared-modelOpenAI'
    ])
    expect(wrapper.text()).toContain('Anthropic')
    expect(wrapper.text()).toContain('OpenAI')
  })
})

describe('PlazaModelPricingTable 长上下文阶梯', () => {
  function ladderIntervals() {
    return [
      {
        min_tokens: 0,
        max_tokens: 272000,
        tier_label: '≤272K',
        input_price: 5e-6,
        output_price: 3e-5,
        cache_write_price: 6.25e-6,
        cache_read_price: 5e-7,
        per_request_price: null
      },
      {
        min_tokens: 272000,
        max_tokens: null,
        tier_label: '>272K',
        input_price: 1e-5,
        output_price: 4.5e-5,
        cache_write_price: 1.25e-5,
        cache_read_price: 1e-6,
        per_request_price: null
      }
    ]
  }

  function ladderModel(overrides: Partial<PlazaModel> = {}): PlazaModel {
    return tokenModel({
      name: 'gpt-5.6-sol',
      platform: 'openai',
      pricing: {
        billing_mode: 'token',
        input_price: 5e-6,
        output_price: 3e-5,
        cache_write_price: 6.25e-6,
        cache_read_price: 5e-7,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: ladderIntervals()
      },
      official_pricing: {
        input_price: 5e-6,
        output_price: 3e-5,
        cache_write_price: 6.25e-6,
        cache_read_price: 5e-7,
        intervals: ladderIntervals()
      },
      long_context_basis: 'whole_request',
      ...overrides
    })
  }

  it('实付缓存列按档分行并乘倍率,每档一行与输入/输出列对齐;档位标签只在输入列', () => {
    const wrapper = mountTable([ladderModel()], 0.5)
    const cells = wrapper.findAll('tbody td')
    const cacheCell = cells[3]
    const rows = cacheCell.findAll('.leading-5')
    expect(rows).toHaveLength(2)
    // 写 6.25 × 0.5 / 读 0.5 × 0.5;高档 12.5 × 0.5 / 1 × 0.5
    expect(rows[0].text()).toContain('modelPlaza.table.cacheWriteShort')
    expect(rows[0].text()).toContain('$3.125')
    expect(rows[0].text()).toContain('$0.25')
    expect(rows[1].text()).toContain('$6.25')
    expect(rows[1].text()).toContain('$0.50')
    // 输入列带标签,输出/缓存列只按行对齐不重复标签
    expect(cells[1].text()).toContain('≤272K')
    expect(cells[1].text()).toContain('>272K')
    expect(cells[2].text()).not.toContain('272K')
    expect(cacheCell.text()).not.toContain('272K')
    expect(cells[1].findAll('.leading-5')).toHaveLength(2)
    expect(cells[2].findAll('.leading-5')).toHaveLength(2)
  })

  it('官方三列按 official_pricing.intervals 分档且不乘倍率,不内联 1h', () => {
    const wrapper = mountTable([ladderModel()], 0.5)
    const cells = wrapper.findAll('tbody td')
    expect(cells[4].text()).toContain('≤272K')
    expect(cells[4].text()).toContain('$5.00')
    expect(cells[4].text()).toContain('>272K')
    expect(cells[4].text()).toContain('$10.00')
    expect(cells[5].text()).toContain('$30.00')
    expect(cells[5].text()).toContain('$45.00')
    expect(cells[6].text()).toContain('$6.25')
    expect(cells[6].text()).toContain('$12.50')
    expect(cells[6].text()).toContain('$1.00')
    expect(cells[6].text()).not.toContain('(1h')
  })

  it('整单计价的档位标签带 tooltip;边际计价在模型名旁加徽章并换用边际说明', () => {
    const whole = mountTable([ladderModel()], 1)
    const wholeLabels = whole.findAll('tbody td span[title="modelPlaza.table.tierHint"]')
    expect(wholeLabels.length).toBeGreaterThan(0)
    expect(whole.text()).not.toContain('modelPlaza.table.marginalBadge')

    const marginal = mountTable([ladderModel({ long_context_basis: 'marginal' })], 1)
    const marginalLabels = marginal.findAll('tbody td span[title="modelPlaza.table.tierHintMarginal"]')
    expect(marginalLabels.length).toBeGreaterThan(0)
    expect(marginal.findAll('tbody td')[0].text()).toContain('modelPlaza.table.marginalBadge')
  })

  it('无标签的多档按区间生成统一形态(≤上限 / >下限),并按下限升序展示', () => {
    const model = ladderModel({
      pricing: {
        ...ladderModel().pricing!,
        // 故意乱序:展示必须按上下文从低到高
        intervals: [
          { ...ladderIntervals()[1], min_tokens: 1000000, tier_label: '' },
          { ...ladderIntervals()[0], min_tokens: 100000, max_tokens: 200000, tier_label: '' },
          { ...ladderIntervals()[0], max_tokens: 100000, tier_label: '' },
          { ...ladderIntervals()[1], min_tokens: 200000, max_tokens: 1000000, tier_label: '' }
        ]
      }
    })
    const rows = mountTable([model], 1).findAll('tbody td')[1].findAll('.leading-5')
    expect(rows.map((r) => r.text().split(/\s+/)[0])).toEqual(['≤100K', '≤200K', '≤1M', '>1M'])
  })

  it('官方无 intervals 字段(旧响应)时官方列保持平价,实付无阶梯时缓存列保持两行', () => {
    const wrapper = mountTable([tokenModel()], 1)
    const cells = wrapper.findAll('tbody td')
    expect(cells[3].text()).toContain('modelPlaza.table.cacheWrite')
    expect(cells[3].text()).toContain('modelPlaza.table.cacheRead')
    expect(cells[3].findAll('.leading-5')).toHaveLength(0)
    expect(cells[6].text()).toContain('(1h')
  })
})

describe('PlazaModelPricingTable 分时计价', () => {
  function timePricedModel() {
    return tokenModel({
      name: 'deepseek-chat',
      platform: 'deepseek',
      time_pricing: {
        timezone: 'Asia/Shanghai',
        periods: [
          { start_time: '00:30', end_time: '08:30:00', multiplier: 0.5 },
          { start_time: '18:00', end_time: '22:00', multiplier: 1.2 }
        ]
      }
    })
  }

  it('有分时倍率的模型展开为标准行 + 每时段一行,时段行价格按倍率折算且倍率列显示生效倍率', () => {
    const wrapper = mountTable([timePricedModel()], 0.8)
    const trs = wrapper.findAll('tbody tr')
    expect(trs).toHaveLength(3)

    // 标准行:输入 3 × 0.8
    const baseCells = trs[0].findAll('td')
    expect(baseCells[0].text()).toBe('deepseek-chat')
    expect(baseCells[1].text()).toContain('$2.40')
    expect(baseCells[7].text()).toContain('0.8x')

    // 夜间时段行:输入 3 × 0.8 × 0.5,倍率 0.4x,标注时段不含时区
    const nightCells = trs[1].findAll('td')
    expect(nightCells[0].text()).toContain('deepseek-chat')
    expect(nightCells[0].text()).toContain('00:30–08:30')
    expect(nightCells[0].text()).not.toContain('Asia/Shanghai')
    // 时区只放在 tooltip 里(i18n mock 不做插值,这里只断言挂了说明)
    expect(nightCells[0].find('[title="modelPlaza.table.timePricingRowHint"]').exists()).toBe(true)
    expect(nightCells[1].text()).toContain('$1.20')
    expect(nightCells[2].text()).toContain('$6.00')
    expect(nightCells[3].text()).toContain('$1.50')
    expect(nightCells[7].text()).toContain('0.4x')

    // 晚高峰行:3 × 0.8 × 1.2 = 2.88,倍率 0.96x
    const peakCells = trs[2].findAll('td')
    expect(peakCells[0].text()).toContain('18:00–22:00')
    expect(peakCells[1].text()).toContain('$2.88')
    expect(peakCells[7].text()).toContain('0.96x')

    // 官方列不受时段影响
    expect(nightCells[4].text()).toContain('$3.00')
  })

  it('仅工作日生效时时段行带工作日前缀,tooltip 换用周末回落文案', () => {
    const model = timePricedModel()
    model.time_pricing!.weekdays_only = true
    const wrapper = mountTable([model], 1)
    const trs = wrapper.findAll('tbody tr')
    expect(trs).toHaveLength(3)

    const nightCells = trs[1].findAll('td')
    expect(nightCells[0].text()).toContain('modelPlaza.table.timePricingWeekdays')
    expect(nightCells[0].text()).toContain('00:30–08:30')
    expect(nightCells[0].find('[title="modelPlaza.table.timePricingRowHintWeekdays"]').exists()).toBe(true)
    expect(nightCells[0].find('[title="modelPlaza.table.timePricingRowHint"]').exists()).toBe(false)
  })

  it('每日生效(无 weekdays_only)不渲染工作日前缀', () => {
    const wrapper = mountTable([timePricedModel()], 1)
    expect(wrapper.find('tbody').text()).not.toContain('modelPlaza.table.timePricingWeekdays')
    expect(wrapper.find('[title="modelPlaza.table.timePricingRowHint"]').exists()).toBe(true)
  })

  it('分组启用高峰倍率时时段行 tooltip 追加高峰披露,价格与倍率列保持不含高峰的口径', () => {
    const wrapper = mountTable([timePricedModel()], 0.8, null, {
      peakWindow: '14:00-18:00 ×1.5 (UTC+08:00)',
      peakRateMultiplier: 1.5
    })
    const nightCells = wrapper.findAll('tbody tr')[1].findAll('td')
    const title = nightCells[0].find('[title*="modelPlaza.table.timePricingRowHint"]').attributes('title')
    expect(title).toContain('modelPlaza.table.timePricingRowHintPeak')
    // 行内数字仍是 基础倍率 × 时段倍率(0.8 × 0.5),高峰只进披露不进价格
    expect(nightCells[1].text()).toContain('$1.20')
    expect(nightCells[7].text()).toContain('0.4x')
  })

  it('分组未启用高峰(peakWindow 缺省)时 tooltip 不含高峰披露', () => {
    const wrapper = mountTable([timePricedModel()], 1)
    const badge = wrapper.find('[title*="modelPlaza.table.timePricingRowHint"]')
    expect(badge.attributes('title')).not.toContain('modelPlaza.table.timePricingRowHintPeak')
  })

  it('无分时倍率时只有一行,不渲染时段标注', () => {
    const wrapper = mountTable([tokenModel()], 1)
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.find('[title*="modelPlaza.table.timePricingRowHint"]').exists()).toBe(false)
  })
})

describe('PlazaModelPricingTable 计费表达式分档价', () => {
  /** 带峰谷价表达式的模型:官方基线价(低谷) + 解析出的分档。 */
  function exprModel(
    overrides: Partial<PlazaModel> = {},
    exprOverrides: Partial<PlazaBillingExpr> = {}
  ): PlazaModel {
    return tokenModel({
      name: 'deepseek-chat',
      platform: 'deepseek',
      official_pricing: {
        input_price: 1.5e-7,
        output_price: 6e-7,
        cache_write_price: 0,
        cache_read_price: 3e-9,
        billing_expr: {
          expression:
            'v1:weekday("Asia/Shanghai") >= 1 ? tier("peak", p * 0.30 + cr * 0.006 + c * 1.20) : tier("off_peak", p * 0.15 + cr * 0.003 + c * 0.60)',
          source: 'builtin',
          recognized: true,
          time_dependent: true,
          variables: ['p', 'c', 'cr'],
          tiers: [
            {
              name: 'peak',
              condition: 'weekday("Asia/Shanghai") >= 1',
              coefficients: { p: 0.3, cr: 0.006, c: 1.2 },
              unit: 'per_million_tokens',
              variables: ['p', 'c', 'cr'],
              time_windows: ['周一至周五 09:00-12:00, 14:00-18:00'],
              timezone: 'Asia/Shanghai'
            },
            {
              name: 'off_peak',
              coefficients: { p: 0.15, cr: 0.003, c: 0.6 },
              unit: 'per_million_tokens',
              variables: ['p', 'c', 'cr']
            }
          ],
          ...exprOverrides
        }
      },
      ...overrides
    })
  }

  it('带 tiers 时渲染分档表:档名 + 各变量单价,表头按表达式实际变量生成', () => {
    const wrapper = mountTable([exprModel()], 1)
    const tables = wrapper.findAll('table')
    expect(tables).toHaveLength(2)
    const table = tables[1]

    const headers = table.findAll('thead th').map((th) => th.text())
    expect(headers[0]).toContain('modelPlaza.table.timeTierTierColumn')
    expect(headers[1]).toContain('modelPlaza.table.timeTierConditionColumn')
    // 表头 = billing_expr.variables 并集(p/c/cr),不写死列
    expect(headers[2]).toContain('modelPlaza.table.varInput')
    expect(headers[3]).toContain('modelPlaza.table.varOutput')
    expect(headers[4]).toContain('modelPlaza.table.varCacheRead')
    expect(table.text()).not.toContain('modelPlaza.table.varCacheWrite')

    const rows = table.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].find('th').text()).toBe('modelPlaza.table.timeTierNamePeak')
    expect(rows[1].find('th').text()).toBe('modelPlaza.table.timeTierNameOffPeak')
    // coefficients 即 USD / 1M token:不乘倍率、不换算、保留 0.006 这类小数位
    expect(rows[0].findAll('td').slice(1).map((td) => td.text().trim())).toEqual([
      '$0.30',
      '$1.20',
      '$0.006'
    ])
    expect(rows[1].findAll('td').slice(1).map((td) => td.text().trim())).toEqual([
      '$0.15',
      '$0.60',
      '$0.003'
    ])

    const text = wrapper.text()
    expect(text).toContain('modelPlaza.table.timeTierTitleTime')
    expect(text).toContain('modelPlaza.table.timeTierSourceBuiltin')
    expect(text).toContain('modelPlaza.table.timeTierUnitNote')
    expect(text).toContain('modelPlaza.table.timeTierBaselineNote')

    // 官方价格列标注「基线」,避免基线价被当成全部
    const officialInput = tables[0].findAll('tbody tr')[0].findAll('td')[4]
    expect(officialInput.text()).toContain('modelPlaza.table.timeTierBaselineBadge')
  })

  it('有时段文案时优先展示人读时段(含时区),不展示 condition 原文', () => {
    const wrapper = mountTable([exprModel()], 1)
    const rows = wrapper.findAll('table')[1].findAll('tbody tr')

    expect(rows[0].findAll('td')[0].text()).toContain('周一至周五 09:00-12:00, 14:00-18:00')
    expect(rows[0].findAll('td')[0].text()).toContain('Asia/Shanghai')
    expect(rows[0].text()).not.toContain('weekday(')

    // 无 time_windows 也无 condition(else 分支) → 其余情况
    expect(rows[1].findAll('td')[0].text().trim()).toBe('modelPlaza.table.timeTierUnconditional')
  })

  it('没有 time_windows 时回退展示 condition 原文', () => {
    const wrapper = mountTable(
      [
        exprModel(
          {},
          {
            time_dependent: false,
            variables: ['p'],
            tiers: [
              {
                name: 'long_context',
                condition: 'len > 128000',
                coefficients: { p: 0.5 },
                unit: 'per_million_tokens',
                variables: ['p']
              }
            ]
          }
        )
      ],
      1
    )
    const table = wrapper.findAll('table')[1]
    expect(table.find('tbody td').text()).toBe('len > 128000')
    expect(wrapper.text()).toContain('modelPlaza.table.timeTierTitleTiers')
  })

  it('recognized=false 时展示表达式原文,不渲染分档表', () => {
    const wrapper = mountTable(
      [exprModel({}, { recognized: false, time_dependent: false, tiers: [], variables: [] })],
      1
    )
    expect(wrapper.findAll('table')).toHaveLength(1)
    expect(wrapper.text()).toContain('modelPlaza.table.timeTierRawNote')
    expect(wrapper.text()).toContain('tier("peak", p * 0.30 + cr * 0.006 + c * 1.20)')
  })

  it('recognized=true 但 tiers 为空数组时同样回退原文', () => {
    const wrapper = mountTable([exprModel({}, { tiers: [], variables: [] })], 1)
    expect(wrapper.findAll('table')).toHaveLength(1)
    expect(wrapper.text()).toContain('modelPlaza.table.timeTierRawNote')
  })

  it('没有 billing_expr 时(原有行为)不出现分档区块,也不标注基线', () => {
    const wrapper = mountTable([tokenModel()], 1)
    expect(wrapper.findAll('table')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('timeTier')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('per_request 档只渲染「每次请求」列,未知档名原样展示', () => {
    const wrapper = mountTable(
      [
        exprModel(
          {},
          {
            time_dependent: false,
            variables: [],
            tiers: [{ name: 'per_call', unit: 'per_request', constant: 0.02 }]
          }
        )
      ],
      1
    )
    const table = wrapper.findAll('table')[1]
    const headers = table.findAll('thead th')
    expect(headers).toHaveLength(3)
    expect(headers[2].text()).toContain('modelPlaza.table.timeTierFlatColumn')
    expect(table.text()).not.toContain('modelPlaza.table.varInput')

    const row = table.find('tbody tr')
    expect(row.find('th').text()).toBe('per_call')
    expect(row.findAll('td')[1].text().trim()).toBe('$0.02')
  })

  it('分档表默认展开,可收起再展开,并同步 aria-expanded', async () => {
    const wrapper = mountTable([exprModel()], 1)
    const toggle = wrapper.find('button')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('table')).toHaveLength(2)

    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.findAll('table')).toHaveLength(1)
    expect(toggle.text()).toContain('modelPlaza.table.timeTierExpand')

    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('table')).toHaveLength(2)
  })

  it('分组内多个模型带表达式时各有一个分档区块与开关', () => {
    const wrapper = mountTable(
      [exprModel({ name: 'deepseek-chat' }), exprModel({ name: 'deepseek-reasoner' })],
      1
    )
    expect(wrapper.findAll('table')).toHaveLength(3)
    expect(wrapper.findAll('button')).toHaveLength(2)
  })
})
