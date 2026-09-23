<template>
  <div class="relative w-full">
    <div
      class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-dark-400"
    >
      <Icon name="search" size="sm" />
    </div>
    <input
      v-model="searchValue"
      type="text"
      class="input h-control pr-8 pl-8 text-control"
      :placeholder="placeholder"
    />
    <button
      v-if="modelValue"
      type="button"
      class="icon-btn icon-btn-sm absolute right-1 top-1/2 -translate-y-1/2"
      aria-label="Clear search"
      @click="clear"
    >
      <Icon name="x" size="xs" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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

const searchValue = computed({
  get: () => props.modelValue,
  set: (value: string) => {
    emit('update:modelValue', value)
    debouncedEmitSearch(value)
  }
})

const clear = () => {
  searchValue.value = ''
}
</script>
