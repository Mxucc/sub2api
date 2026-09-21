import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../PlazaNavBar.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('PlazaNavBar sticky offset', () => {
  // 回归：#模型广场与页头重叠。
  // PlazaNavBar 只在「独立形态」出现（后台形态走 AppLayout，见 ModelPlazaView.vue），
  // 因此它上面没有 56px 的通栏顶栏；写死 top-14 会让它悬在距顶 56px 处，
  // 页面内容从上方那道空隙里穿过去，看起来就是导航条与标题重叠。
  it('sticks to the viewport top instead of assuming a 56px app header above it', () => {
    expect(componentSource).toContain('sticky top-0')
    expect(componentSource).not.toContain('sticky top-14')
  })
})
