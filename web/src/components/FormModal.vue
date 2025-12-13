<script setup lang="ts">
import { X, Save } from 'lucide-vue-next'

interface Props {
  show: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  loading?: boolean
  error?: string
  submitText?: string
  cancelText?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  size: 'md',
  loading: false,
  error: '',
  submitText: '保存',
  cancelText: '取消'
})

const emit = defineEmits<{
  close: []
  submit: []
}>()

const sizeClasses = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl'
}

function handleClose() {
  if (!props.loading) {
    emit('close')
  }
}

function handleSubmit() {
  emit('submit')
}
</script>

<template>
  <Teleport to="body">
    <dialog :class="['modal', show && 'modal-open']">
      <div :class="['modal-box', sizeClasses[size]]">
        <!-- Header -->
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-bold text-lg">{{ title }}</h3>
          <button
            class="btn btn-sm btn-circle btn-ghost"
            :disabled="loading"
            @click="handleClose"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Form -->
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- Error Alert -->
          <div v-if="error" class="alert alert-error text-sm">
            {{ error }}
          </div>

          <!-- Form Fields -->
          <slot />

          <!-- Footer Actions -->
          <div class="modal-action">
            <button
              type="button"
              class="btn"
              :disabled="loading"
              @click="handleClose"
            >
              {{ cancelText }}
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="loading"
            >
              <span v-if="loading" class="loading loading-spinner loading-sm"></span>
              <Save v-else class="w-4 h-4" />
              {{ submitText }}
            </button>
          </div>
        </form>
      </div>

      <!-- Backdrop -->
      <form method="dialog" class="modal-backdrop">
        <button type="button" @click="handleClose">close</button>
      </form>
    </dialog>
  </Teleport>
</template>
