<template>
  <button
    type="button"
    @click="toggle"
    class="switch inline-flex h-[18px] w-8 shrink-0 transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-45"
    :class="modelValue && 'switch-active'"
    role="switch"
    :aria-checked="modelValue"
  >
    <span
      class="switch-thumb h-3.5 w-3.5 transition-transform duration-150"
      :class="modelValue ? 'translate-x-3.5' : 'translate-x-0'"
    />
  </button>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  emit('update:modelValue', !props.modelValue)
}
</script>
