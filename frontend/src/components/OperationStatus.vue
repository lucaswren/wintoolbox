<script setup>
import { computed } from 'vue'
import { t } from '../i18n/index.js'
const props = defineProps({ operation: { type: Object, required: true } })
defineEmits(['dismiss'])
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
    <div class="operation-heading">
      <span v-if="pending" class="operation-spinner" aria-hidden="true"></span
      ><strong>{{ label }}</strong
      ><span>{{ t('耗时 {0} 秒', [operation.elapsed]) }}</span
      ><el-button v-if="!pending" link @click="$emit('dismiss')">{{ t('关闭提示') }}</el-button>
    </div>
    <div class="operation-title">{{ operation.title }}</div>
    <p v-if="operation.detail">{{ operation.detail }}</p>
    <p v-else>
      {{
        operation.phase === 'verifying'
          ? t('命令已返回，正在读取系统状态；复查完成前请勿重复操作。')
          : t('正在等待 Windows 完成操作，请勿重复操作。执行时间取决于系统服务和策略。')
      }}
    </p>
  </section>
</template>
<style scoped>
.operation-status {
  flex: 0 0 auto;
  padding: 10px 22px;
  background: #eff6ff;
  border-bottom: 1px solid #bfdbfe;
  font-size: 13px;
  max-height: 180px;
  overflow: auto;
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
.operation-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.operation-title,
p {
  overflow-wrap: anywhere;
  margin: 5px 0 0;
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
