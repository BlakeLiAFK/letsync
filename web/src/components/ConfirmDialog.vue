<script setup lang="ts">
import { useConfirm } from '@/stores/confirm'
import { AlertTriangle, AlertCircle, Info, X } from 'lucide-vue-next'

const { state, confirm, cancel } = useConfirm()

const icons = {
  warning: AlertTriangle,
  danger: AlertCircle,
  info: Info
}

const btnClass = {
  warning: 'btn-warning',
  danger: 'btn-error',
  info: 'btn-info'
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="state.visible" class="modal modal-open">
        <div class="modal-box max-w-sm">
          <!-- Header -->
          <div class="flex items-start gap-3">
            <div
              :class="[
                'p-2 rounded-full',
                state.type === 'danger' ? 'bg-error/10 text-error' : '',
                state.type === 'warning' ? 'bg-warning/10 text-warning' : '',
                state.type === 'info' ? 'bg-info/10 text-info' : ''
              ]"
            >
              <component :is="icons[state.type]" class="w-6 h-6" />
            </div>
            <div class="flex-1">
              <h3 class="font-bold text-lg">{{ state.title }}</h3>
              <p class="py-2 text-base-content/70">{{ state.message }}</p>
            </div>
            <button class="btn btn-ghost btn-sm btn-circle" @click="cancel">
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- Actions -->
          <div class="modal-action">
            <button class="btn btn-ghost" @click="cancel">
              {{ state.cancelText }}
            </button>
            <button :class="['btn', btnClass[state.type]]" @click="confirm">
              {{ state.confirmText }}
            </button>
          </div>
        </div>
        <div class="modal-backdrop bg-black/50" @click="cancel"></div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal-box,
.modal-leave-active .modal-box {
  transition: transform 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-box,
.modal-leave-to .modal-box {
  transform: scale(0.95);
}
</style>
