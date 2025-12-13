<script setup lang="ts">
import { X } from 'lucide-vue-next'

interface Props {
  show: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  closable?: boolean
  closeOnBackdrop?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  size: 'md',
  closable: true,
  closeOnBackdrop: true
})

const emit = defineEmits<{
  close: []
}>()

const sizeClasses = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
  full: 'max-w-full mx-4'
}

function handleClose() {
  if (props.closable) {
    emit('close')
  }
}

function handleBackdropClick() {
  if (props.closeOnBackdrop) {
    handleClose()
  }
}
</script>

<template>
  <Teleport to="body">
    <dialog :class="['modal', show && 'modal-open']">
      <div :class="['modal-box', sizeClasses[size]]">
        <!-- Header -->
        <div v-if="title || closable || $slots.header" class="flex items-center justify-between mb-4">
          <slot name="header">
            <h3 class="font-bold text-lg">{{ title }}</h3>
          </slot>
          <button
            v-if="closable"
            class="btn btn-sm btn-circle btn-ghost"
            @click="handleClose"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Body -->
        <div class="modal-body">
          <slot />
        </div>

        <!-- Footer -->
        <div v-if="$slots.footer" class="modal-action">
          <slot name="footer" />
        </div>
      </div>

      <!-- Backdrop -->
      <form method="dialog" class="modal-backdrop">
        <button type="button" @click="handleBackdropClick">close</button>
      </form>
    </dialog>
  </Teleport>
</template>
