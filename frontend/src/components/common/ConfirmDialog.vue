<template>
  <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
    <div class="space-y-3">
      <p class="text-control leading-relaxed text-gray-600 dark:text-dark-400">{{ message }}</p>
      <slot></slot>
    </div>

    <template #footer>
      <button
        @click="handleCancel"
        type="button"
        class="btn btn-secondary h-control px-4 text-control"
      >
        {{ cancelText }}
      </button>
      <button
        @click="handleConfirm"
        type="button"
        :class="['btn h-control px-4 text-control', danger ? 'btn-danger' : 'btn-primary']"
      >
        {{ confirmText }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'

const { t } = useI18n()

interface Props {
  show: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface Emits {
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  danger: false
})

const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

const emit = defineEmits<Emits>()

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
}
</script>
