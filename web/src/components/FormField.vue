<script setup lang="ts">
// 表单字段组件 - 统一的水平布局
// 用法:
// <FormField label="标题">
//   <input class="input input-bordered" />
// </FormField>

interface Props {
  label: string
  required?: boolean
  hint?: string
}

withDefaults(defineProps<Props>(), {
  required: false,
  hint: ''
})
</script>

<template>
  <div class="form-field">
    <label class="form-field-label">
      <span>{{ label }}</span>
      <span v-if="required" class="text-error">*</span>
    </label>
    <div class="form-field-content">
      <slot />
      <p v-if="hint" class="text-xs text-base-content/50 mt-1">{{ hint }}</p>
    </div>
  </div>
</template>

<style scoped>
.form-field {
  display: contents;
}

.form-field-label {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.875rem;
  white-space: nowrap;
  padding-top: 0.75rem;
  align-self: start;
}

.form-field-content {
  min-width: 0;
}

/* 文本输入框、下拉框、文本域自动填充宽度 */
.form-field-content :deep(input:not([type="checkbox"]):not([type="radio"])),
.form-field-content :deep(select),
.form-field-content :deep(textarea) {
  width: 100%;
}
</style>
