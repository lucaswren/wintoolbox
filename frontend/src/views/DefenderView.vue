<script setup>
import { t } from '../i18n/index.js'

import { computed } from 'vue'

const props = defineProps({
  status: { type: Object, default: null },
  busy: { type: Boolean, default: false },
})

defineEmits(['disable', 'enable'])

const unknown = computed(() => !!props.status?.defenderUnknown)
const disabled = computed(() => !!props.status?.defenderDisabled)
const canDisable = computed(() => !props.busy && !unknown.value && !disabled.value)
const canEnable = computed(() => !props.busy && !unknown.value && disabled.value)
const tagType = computed(() => {
  if (unknown.value) return 'info'
  return disabled.value ? 'danger' : 'success'
})
const tagText = computed(() => {
  if (unknown.value) return t('未知')
  return disabled.value ? t('已关闭') : t('运行中')
})
</script>

<template>
  <div data-view="defender">
    <p class="page-desc">
      {{ t('一键关闭或启用 Windows Defender 实时防护（若开启篡改防护可能失败）') }}
    </p>

    <el-card shadow="never" :header="t('实时防护')" class="wt-card">
      <div class="wt-status-block">
        <div class="wt-status-line">
          <span class="wt-status-label">{{ t('状态') }}</span>
          <el-tag :type="tagType" effect="dark">
            {{ tagText }}
          </el-tag>
        </div>
        <p class="wt-detail">{{ t(status?.defenderDetail || '—') }}</p>
      </div>
      <div class="wt-actions">
        <el-button type="danger" :disabled="!canDisable" @click="$emit('disable')">{{
          t('关闭实时防护')
        }}</el-button>
        <el-button type="success" :disabled="!canEnable" @click="$emit('enable')">{{
          t('启用实时防护')
        }}</el-button>
      </div>
    </el-card>
  </div>
</template>
