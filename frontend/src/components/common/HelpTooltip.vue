<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, useTemplateRef, nextTick } from 'vue'

const props = withDefaults(defineProps<{
  content?: string
  trigger?: 'hover' | 'click'
  widthClass?: string
}>(), {
  trigger: 'hover',
  widthClass: 'w-64',
})

/** 触发元素与提示框之间的间距（与模板里的 before:h-2 / -bottom-1 保持一致） */
const GAP = 8

const show = ref(false)
const triggerRef = useTemplateRef<HTMLElement>('trigger')
const tooltipRef = useTemplateRef<HTMLElement>('tooltip')
const tooltipStyle = ref({ top: '0px', left: '0px' })
/** 空间不足时翻到触发元素下方 */
const placement = ref<'top' | 'bottom'>('top')

function openTooltip() {
  show.value = true
  nextTick(updatePosition)
}

function closeTooltip() {
  show.value = false
}

function onEnter() {
  if (props.trigger !== 'hover') return
  openTooltip()
}

function isInside(container: HTMLElement | null, target: EventTarget | null): boolean {
  return target instanceof Node && !!container?.contains(target)
}

// 悬停模式下指针在触发图标与提示框之间往返时保持打开，便于选中提示里的文字。
function onLeave(event: MouseEvent) {
  if (props.trigger !== 'hover') return
  if (isInside(tooltipRef.value, event.relatedTarget)) return
  closeTooltip()
}

function onTooltipLeave(event: MouseEvent) {
  if (props.trigger !== 'hover') return
  if (isInside(triggerRef.value, event.relatedTarget)) return
  closeTooltip()
}

function onClick(event: MouseEvent) {
  if (props.trigger !== 'click') return
  event.stopPropagation()
  if (show.value) {
    closeTooltip()
    return
  }
  openTooltip()
}

function onDocumentClick(event: MouseEvent) {
  if (props.trigger !== 'click' || !show.value) return
  const target = event.target as Node | null
  if (!target) return
  if (triggerRef.value?.contains(target) || tooltipRef.value?.contains(target)) return
  closeTooltip()
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (props.trigger !== 'click') return
  if (event.key === 'Escape') {
    closeTooltip()
  }
}

function onViewportChange() {
  if (!show.value) return
  updatePosition()
}

/**
 * 提示框是 `position: fixed`（Teleport 到 body 躲开表格/弹窗的 overflow 裁剪），
 * 因此必须使用**视口坐标**：rect.top/left 本身就是视口坐标，不能再叠加 window.scrollY/scrollX，
 * 否则页面或表格滚动后提示框会整体下移，直接盖住触发元素（表格里被盖住的 URL 就点不到了）。
 */
function updatePosition() {
  const el = triggerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const tip = tooltipRef.value
  const tipHeight = tip?.offsetHeight ?? 0
  const tipWidth = tip?.offsetWidth ?? 0

  // 上方放不下就翻到下方（下方也放不下时仍留在上方，尽量不遮住触发元素）
  const fitsAbove = rect.top - GAP >= tipHeight
  const fitsBelow = rect.bottom + GAP + tipHeight <= window.innerHeight
  placement.value = !fitsAbove && fitsBelow ? 'bottom' : 'top'

  const top = placement.value === 'top' ? rect.top - GAP : rect.bottom + GAP

  // 水平夹取到视口内（提示框以中心对齐，所以按半宽夹取）
  const half = tipWidth / 2
  const minLeft = half + GAP
  const maxLeft = Math.max(minLeft, window.innerWidth - half - GAP)
  const left = Math.min(Math.max(rect.left + rect.width / 2, minLeft), maxLeft)

  tooltipStyle.value = { top: `${top}px`, left: `${left}px` }
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick, true)
  document.addEventListener('keydown', onDocumentKeydown)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick, true)
  document.removeEventListener('keydown', onDocumentKeydown)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
})
</script>

<template>
  <div
    ref="trigger"
    class="group relative ml-1 inline-flex items-center align-middle"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @click="onClick"
  >
    <!-- Trigger Icon -->
    <slot name="trigger">
      <svg
        class="h-4 w-4 cursor-help text-gray-400 transition-colors duration-150 hover:text-gray-600 dark:text-dark-400 dark:hover:text-gray-200"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
    </slot>

    <!-- Teleport to body to escape modal overflow clipping -->
    <Teleport to="body">
      <!--
        before: 伪元素只盖住触发元素与提示框之间那 8px 空隙（同 GAP），让指针能连续移入提示框；
        高度必须等于空隙，否则会压到触发元素上，表格里的链接/按钮就点不到了。
      -->
      <div
        ref="tooltip"
        v-show="show"
        role="tooltip"
        :class="[
          'fixed z-[99999] -translate-x-1/2 rounded-lg bg-gray-900 p-2.5 text-xs leading-relaxed text-white shadow-popover ring-1 ring-white/10 selection:bg-primary-200 selection:text-gray-900 dark:bg-dark-800 dark:ring-dark-600 dark:selection:bg-primary-200 dark:selection:text-gray-900',
          placement === 'top'
            ? '-translate-y-full before:absolute before:inset-x-0 before:top-full before:h-2'
            : 'before:absolute before:inset-x-0 before:bottom-full before:h-2',
          props.widthClass,
        ]"
        :style="{ top: tooltipStyle.top, left: tooltipStyle.left }"
        :data-placement="placement"
        @mouseleave="onTooltipLeave"
      >
        <button
          v-if="props.trigger === 'click'"
          type="button"
          class="absolute right-1.5 top-1.5 rounded-md p-1 text-gray-300 transition-colors duration-150 hover:bg-white/10 hover:text-white"
          aria-label="Close"
          @click.stop="closeTooltip"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
        <slot>{{ content }}</slot>
        <div
          :class="[
            'absolute left-1/2 h-2 w-2 -translate-x-1/2 rotate-45 bg-gray-900 dark:bg-dark-800',
            placement === 'top' ? '-bottom-1' : '-top-1',
          ]"
        ></div>
      </div>
    </Teleport>
  </div>
</template>
