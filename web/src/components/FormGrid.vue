<script setup lang="ts">
// 表单网格容器 - 为 FormField 提供 Grid 布局
// 用法:
// <FormGrid>
//   <FormField label="标题">
//     <input />
//   </FormField>
// </FormGrid>
</script>

<template>
  <div class="form-grid">
    <slot />
  </div>
</template>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.75rem 1rem;
  align-items: start;
}

/* 响应式：移动端改为垂直布局 */
@media (max-width: 480px) {
  .form-grid {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }

  .form-grid :deep(.form-field-label) {
    padding-top: 0;
  }
}
</style>
