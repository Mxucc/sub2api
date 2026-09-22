import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

function getTooltipElement(): HTMLDivElement {
  const tooltip = document.body.querySelector('[role="tooltip"]')
  if (!(tooltip instanceof HTMLDivElement)) {
    throw new Error('tooltip element not found')
  }
  return tooltip
}

describe('HelpTooltip', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('keeps the existing hover interaction by default', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'hover details',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('mouseenter')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    await trigger.trigger('mouseleave')
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('keeps a hover tooltip open while the pointer moves between the trigger and the tooltip', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'copyable details',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    await trigger.trigger('mouseenter')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    await trigger.trigger('mouseleave', { relatedTarget: tooltip })
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: trigger.element }))
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: null }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('supports click-to-toggle details and closes on outside click', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'click details',
        trigger: 'click',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    expect(tooltip.textContent).toContain('click details')

    const closeButton = tooltip.querySelector('button[aria-label="Close"]')
    if (!(closeButton instanceof HTMLButtonElement)) {
      throw new Error('close button not found')
    }
    closeButton.click()
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  // 回归：提示框是 position: fixed，必须用视口坐标。
  // 之前叠加了 window.scrollY/scrollX，页面或表格滚动后提示框整体下移，
  // 直接盖住触发元素 —— 账号管理里被盖住的 URL 就点不到了。
  it('positions the tooltip with viewport coordinates, not page coordinates', async () => {
    Object.defineProperty(window, 'scrollY', { value: 137, configurable: true })
    Object.defineProperty(window, 'scrollX', { value: 42, configurable: true })

    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: { content: 'viewport metrics' },
    })

    const trigger = wrapper.get('.group')
    trigger.element.getBoundingClientRect = () =>
      ({ top: 200, left: 300, width: 100, height: 20, bottom: 220, right: 400, x: 300, y: 200, toJSON: () => ({}) }) as DOMRect

    await trigger.trigger('mouseenter')
    await nextTick()

    const tooltip = getTooltipElement()
    // 顶边 = 触发元素 top - 8px 间距；left 为触发元素水平中心（视口内未被夹取）
    expect(tooltip.style.top).toBe('192px')
    expect(tooltip.style.left).toBe('350px')

    wrapper.unmount()
  })

  it('keeps the hover bridge exactly as tall as the gap so it never covers the trigger', () => {
    const wrapper = mount(HelpTooltip, { attachTo: document.body, props: { content: 'gap' } })
    const tooltip = getTooltipElement()

    // 空隙是 8px：桥接层只能是 h-2，h-3(12px) 会压到触发元素上、挡掉点击
    expect(tooltip.className).toContain('before:h-2')
    expect(tooltip.className).not.toContain('before:h-3')

    wrapper.unmount()
  })
})
