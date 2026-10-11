<script setup>
import { t } from '../i18n/index.js'

import { nextTick, ref, watch } from 'vue'
import { logLevel } from '../composables/logBuffer.js'

const props = defineProps({
  lines: { type: Array, default: () => [] },
})
defineEmits(['clear'])
const scrollbar = ref()
const expanded = ref(false)
watch(
  () => props.lines[props.lines.length - 1],
  async () => {
    await nextTick()
    scrollbar.value?.setScrollTop(Number.MAX_SAFE_INTEGER)
  },
)
function exportLog() {
  const url = URL.createObjectURL(
    new Blob([props.lines.join('\n')], { type: 'text/plain;charset=utf-8' }),
  )
  const link = document.createElement('a')
  link.href = url
  link.download = 'wintoolbox-log.txt'
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="op-log">
    <div class="op-log__head">
      <span class="op-log__title">{{ t('活动日志') }}</span>
      <el-button text size="small" @click="expanded = !expanded">{{
        expanded ? t('收起') : t('展开')
      }}</el-button>
      <el-button text size="small" :disabled="!lines.length" @click="exportLog">{{
        t('导出')
      }}</el-button>
      <el-button text type="primary" size="small" @click="$emit('clear')">{{
        t('清空')
      }}</el-button>
    </div>
    <el-scrollbar
      ref="scrollbar"
      class="op-log__body"
      :style="{ height: expanded ? '180px' : '68px' }"
      always
    >
      <div
        v-for="(line, i) in lines"
        :key="i"
        class="op-log__line"
        :class="'op-log__line--' + logLevel(line)"
      >
        {{ line }}
      </div>
      <div v-if="!lines.length" class="op-log__empty">{{ t('暂无记录') }}</div>
    </el-scrollbar>
  </div>
</template>

<style scoped>
.op-log {
  border-top: 1px solid var(--wt-border);
  background: rgba(255, 255, 255, 0.92);
  padding: 10px 18px 12px;
}
.op-log__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.op-log__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--wt-text-secondary);
  letter-spacing: 0.02em;
}
.op-log__body {
  height: 68px;
}
.op-log__line {
  font-size: 12px;
  color: var(--wt-text-regular);
  line-height: 1.65;
  font-family: Consolas, 'Cascadia Mono', 'Microsoft YaHei UI', monospace;
}
.op-log__empty {
  font-size: 12px;
  color: var(--wt-text-placeholder);
}
.op-log__line--error {
  color: var(--el-color-danger);
}
.op-log__line--warning {
  color: var(--el-color-warning);
}
</style>
