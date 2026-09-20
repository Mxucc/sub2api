<template>
  <div class="relative w-full">
    <div
      class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5 text-gray-400 dark:text-dark-400"
    >
      <Icon name="search" size="sm" />
    </div>
    <input
      :value="modelValue"
      type="text"
      class="input pr-9 pl-10"
      :placeholder="placeholder"
      @input="handleInput"
    />
    <button
      v-if="modelValue"
      type="button"
      class="icon-btn icon-btn-sm absolute right-1.5 top-1/2 -translate-y-1/2"
      aria-label="Clear search"
      @click="clear"
    >
      <Icon name="x" size="xs" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  debounceMs?: number
}>(), {
  placeholder: 'Search...',
  debounceMs: 300
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search', value: string): void
}>()

const debouncedEmitSearch = useDebounceFn((value: string) => {
  emit('search', value)
}, props.debounceMs)

const clear = () => {
  emit('update:modelValue', '')
  debouncedEmitSearch('')
}

const handleInput = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
  debouncedEmitSearch(value)
}
</script>
