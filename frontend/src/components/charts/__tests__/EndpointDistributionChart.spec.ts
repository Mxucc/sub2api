import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import EndpointDistributionChart from '../EndpointDistributionChart.vue'

const messages: Record<string, string> = {
  'usage.endpointDistribution': 'Endpoint Distribution',
  'usage.endpoint': 'Endpoint',
  'usage.inbound': 'Inbound',
  'usage.upstream': 'Upstream',
  'usage.path': 'Path',
  'admin.dashboard.requests': 'Requests',
  'admin.dashboard.tokens': 'Tokens',
  'admin.dashboard.actual': 'Actual',
  'admin.dashboard.standard': 'Standard',
  'admin.dashboard.metricTokens': 'By Tokens',
  'admin.dashboard.metricActualCost': 'By Actual Cost',
  'admin.dashboard.noDataAvailable': 'No data available',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

describe('EndpointDistributionChart', () => {
  const endpointStats = [
    { endpoint: '/v1/messages', requests: 9, total_tokens: 1200, cost: 1.8, actual_cost: 0.1 },
    { endpoint: '/v1/chat/completions', requests: 4, total_tokens: 600, cost: 0.7, actual_cost: 0.9 },
  ]

  it('uses total_tokens and token ordering by default', () => {
    const wrapper = mount(EndpointDistributionChart, {
      props: { endpointStats },
      global: { stubs: { LoadingSpinner: true } },
    })

    const chartData = JSON.parse(wrapper.find('.chart-data').text())
    expect(chartData.labels).toEqual(['/v1/messages', '/v1/chat/completions'])
    expect(chartData.datasets[0].data).toEqual([1200, 600])

    const options = (wrapper.vm as any).$?.setupState.doughnutOptions
    const label = options.plugins.tooltip.callbacks.label({
      label: '/v1/messages',
      raw: 1200,
      dataset: { data: [1200, 600] },
    })
    expect(label).toBe('/v1/messages: 1.20K (66.7%)')
  })

  it('renders a bare hairline header by default and adds card chrome only when framed', () => {
    const bare = mount(EndpointDistributionChart, {
      props: { endpointStats },
      global: { stubs: { LoadingSpinner: true } },
    })

    expect(bare.find('.card').exists()).toBe(false)
    expect(bare.find('.chart-hair .section-heading-title').text()).toBe('Endpoint Distribution')
    expect(bare.find('.chart-canvas').exists()).toBe(true)

    const framed = mount(EndpointDistributionChart, {
      props: { endpointStats, framed: true },
      global: { stubs: { LoadingSpinner: true } },
    })

    expect(framed.find('.card').exists()).toBe(false)
    expect(framed.classes()).toContain('p-4')
    expect(framed.classes()).toContain('border')
  })
})
