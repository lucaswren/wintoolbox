<script setup>
import { computed, watch } from 'vue'
import { t } from '../i18n/index.js'
const props = defineProps({ operation: { type: Object, required: true } })
const emit = defineEmits(['dismiss', 'show-log'])
watch(
  () => props.operation.phase,
  (phase, _, onCleanup) => {
    if (phase !== 'completed') return
    const timer = setTimeout(() => {
      if (props.operation.phase === 'completed') emit('dismiss')
    }, 3000)
    onCleanup(() => clearTimeout(timer))
  },
  { immediate: true, flush: 'sync' },
)
const pending = computed(() => ['running', 'verifying'].includes(props.operation.phase))
const label = computed(() =>
  t(
    {
      running: '正在执行',
      verifying: '正在复查状态',
      completed: '操作完成',
      warning: '需要复查',
      failed: '操作未完成',
    }[props.operation.phase] || '系统操作',
  ),
)
const summary = computed(() => {
  const detail = props.operation.detail || ''
  if (props.operation.phase === 'completed' && detail.length <= 60 && !/[\r\n]/.test(detail))
    return detail ? t(detail) : label.value
  return label.value
})
</script>
<template>
  <section
    v-if="operation.phase !== 'idle'"
    class="operation-status"
    :class="operation.phase"
    role="status"
    aria-live="polite"
    aria-atomic="true"
    data-testid="operation-status"
  >
    <span v-if="pending" class="operation-spinner" aria-hidden="true"></span>
    <span class="operation-summary">{{ summary }}</span>
    <el-button v-if="!pending" link @click="$emit('show-log')">{{ t('查看日志') }}</el-button>
    <el-button v-if="!pending" link @click="$emit('dismiss')">{{ t('关闭提示') }}</el-button>
  </section>
</template>
<style scoped>
.operation-status {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 6px 22px;
  background: #eff6ff;
  border-bottom: 1px solid #bfdbfe;
  font-size: 13px;
}
.operation-status.failed {
  background: #fef2f2;
  border-color: #fecaca;
}
.operation-status.warning {
  background: #fffbeb;
  border-color: #fde68a;
}
.operation-status.completed {
  background: #f0fdf4;
  border-color: #bbf7d0;
}
.operation-summary {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.operation-status > .el-button,
.operation-spinner {
  flex-shrink: 0;
}
.operation-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid #93c5fd;
  border-top-color: #2563eb;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
@media (prefers-reduced-motion: reduce) {
  .operation-spinner {
    animation: none;
  }
}
</style>
