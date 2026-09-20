import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Icon from '../Icon.vue'
import { iconMap } from '../iconMap'

describe('Icon（Lucide）', () => {
  it('所有图标名都能渲染出 svg（防止映射遗漏导致图标静默消失）', () => {
    const names = Object.keys(iconMap) as Array<keyof typeof iconMap>
    expect(names.length).toBeGreaterThanOrEqual(80)

    const broken: string[] = []
    for (const name of names) {
      const wrapper = mount(Icon, { props: { name } })
      const svg = wrapper.find('svg')
      if (!svg.exists() || svg.findAll('path, circle, rect, line, polyline').length === 0) {
        broken.push(name)
      }
    }
    expect(broken).toEqual([])
  })

  it('默认线宽 1.5，尺寸类名按 size 生效', () => {
    const wrapper = mount(Icon, { props: { name: 'play' } })
    expect(wrapper.attributes('stroke-width')).toBe('1.5')
    expect(wrapper.classes()).toContain('h-5')
    expect(wrapper.classes()).toContain('w-5')

    const large = mount(Icon, { props: { name: 'play', size: 'lg', strokeWidth: 2 } })
    expect(large.attributes('stroke-width')).toBe('2')
    expect(large.classes()).toContain('h-6')
    expect(large.classes()).toContain('w-6')
  })

  it('未知图标名回退到兜底图标而不是渲染空节点', () => {
    const wrapper = mount(Icon, { props: { name: 'not-exist' as never } })
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('path').length).toBeGreaterThan(0)
  })
})
