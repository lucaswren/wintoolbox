<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api } from '../api/index.js'
import { t } from '../i18n/index.js'
import { useRdpSafety } from '../composables/useRdpSafety.js'

const { status, error, loading, refresh } = useRdpSafety()
const working = ref(false)
const now = ref(Date.now())
const phaseLabel = computed(() =>
  t(
    {
      armed: '回滚保护已就绪',
      'awaiting-confirmation': '等待重新连接确认',
      'rolling-back': '正在恢复原配置',
      'rollback-failed': '恢复失败，任务将重试',
    }[status.value?.phase] || '状态未知',
  ),
)
const seconds = computed(
  () =>
    Math.min(
      300,
      Math.max(0, Math.ceil((Date.parse(status.value?.deadline) - now.value) / 1000)),
    ) || 0,
)
let poll, clock
onMounted(() => {
  refresh()
  poll = setInterval(refresh, 15000)
  clock = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  clearInterval(poll)
  clearInterval(clock)
})
async function finish(restore) {
  const transaction = status.value?.transaction
  working.value = true
  try {
    await ElMessageBox.confirm(
      t(
        restore
          ? '确认立即恢复此次变更前的配置？连接可能断开。'
          : '确认已通过新连接重新登录，并保留此次配置？',
      ),
      t('请确认'),
      { type: 'warning', closeOnClickModal: false },
    )
    if (!transaction || transaction !== status.value?.transaction)
      throw Error(t('待确认配置已变化，请刷新后重试'))
    if (restore) await api.rollbackRdpSafety(transaction)
    else await api.confirmRdpSafety(transaction)
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(e?.message || String(e))
  } finally {
    await refresh()
    working.value = false
  }
}
</script>

<template>
  <section
    v-if="status?.pending"
    class="rdp-safety-banner"
    role="status"
    data-testid="rdp-safety-banner"
  >
    <el-alert type="warning" :closable="false" show-icon>
      <template #title>{{ phaseLabel }} · {{ seconds }}s</template>
      <p>
        {{
          t(
            '请在五分钟内断开并通过新配置重新连接，再确认保留。超时由独立任务恢复；程序退出后仍有效。',
          )
        }}
      </p>
      <p v-if="error">{{ error }}</p>
      <p v-if="status.phase === 'rollback-failed'">
        {{ t('自动恢复尚未完成。请保留当前通道，检查警告或进入 VPS 控制台处理。') }}
      </p>
      <div class="rdp-safety-actions">
        <el-button :loading="loading" @click="refresh">{{ t('刷新') }}</el-button>
        <el-button
          type="success"
          :disabled="working || loading || !status.canConfirm || !!error || seconds === 0"
          @click="finish(false)"
          >{{ t('重新连接成功，保留配置') }}</el-button
        >
        <el-button type="danger" :disabled="working" @click="finish(true)">{{
          t('立即恢复原配置')
        }}</el-button>
      </div>
    </el-alert>
  </section>
</template>

<style scoped>
.rdp-safety-banner {
  flex: 0 0 auto;
  max-height: min(220px, 30vh);
  overflow: auto;
}
.rdp-safety-banner :deep(.el-alert) {
  align-items: flex-start;
}
.rdp-safety-banner :deep(.el-alert__content) {
  min-width: 0;
  width: 100%;
}
.rdp-safety-banner p {
  margin: 6px 0;
  overflow-wrap: anywhere;
}
.rdp-safety-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.rdp-safety-actions :deep(.el-button) {
  margin: 0;
}
</style>
