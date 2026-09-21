import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const styleSource = readFileSync(resolve(here, '../../../../style.css'), 'utf8')
const statsSource = readFileSync(resolve(here, '../UserDashboardStats.vue'), 'utf8')

/**
 * 回归：仪表盘卡片过大会让一整屏看不到关键信息。
 * 这里锁住「紧凑指标卡」的关键尺寸 —— 改这些数值应当是有意为之，而不是悄悄回退。
 */
describe('dashboard metric density', () => {
  it('keeps the metric card padding compact via a single var', () => {
    expect(styleSource).toContain('--metric-pad: 13px;')
    expect(styleSource).toContain('padding: var(--metric-pad);')
    // 满出血页脚跟着同一个变量取负边距，避免出现悬空的 1px 线
    expect(styleSource).toContain('margin-inline: calc(var(--metric-pad) * -1);')
    expect(styleSource).toContain('margin-bottom: calc(var(--metric-pad) * -1);')
  })

  it('keeps the metric value and lead height small', () => {
    expect(styleSource).toMatch(/\.metric-value \{[^}]*text-\[22px\]/)
    expect(styleSource).toMatch(/\.metric-lead \{[^}]*min-h-\[104px\]/)
  })

  it('reaches 3 and 4 columns earlier so more cards fit per row', () => {
    expect(styleSource).toMatch(
      /\.metric-grid \{[^}]*gap-2\.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4/
    )
  })

  it('renders the balance focus card inside the metric grid instead of a full-width band', () => {
    expect(statsSource).toContain('metric-card metric-card-hero')
    expect(statsSource).not.toContain('lg:col-span-3')
    expect(statsSource).not.toContain('metric-grid-lead')
  })
})
