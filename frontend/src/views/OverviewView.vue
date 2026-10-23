<script setup>
import { t } from '../i18n/index.js'

import { computed } from 'vue'

const descLabelStyle = {
  width: '112px',
  minWidth: '112px',
  maxWidth: '112px',
  whiteSpace: 'normal',
}

const props = defineProps({
  overview: { type: Object, default: null },
  detailLoading: { type: Boolean, default: false },
})

const actType = computed(() => {
  const s = props.overview?.activationStatus
  if (s === '已激活') return 'success'
  if (s === '检测中…' || props.detailLoading) return 'info'
  if (s === '获取失败' || s === '获取超时') return 'danger'
  if (s === '未知' || s === '—') return 'info'
  if (s?.includes('宽限期') || s?.includes('通知')) return 'warning'
  return 'danger'
})

function memText() {
  const o = props.overview
  if (!o?.memoryTotalGB) return '—'
  const used = Math.max(0, o.memoryTotalGB - (o.memoryAvailGB || 0))
  const pct = Math.round((used / o.memoryTotalGB) * 100)
  return t('已用 {0} / 共 {1} GB（{2}%）', [used.toFixed(1), o.memoryTotalGB.toFixed(1), pct])
}
</script>

<template>
  <div data-view="overview" class="overview-page">
    <p class="page-desc">{{ t('快速查看这台计算机的系统与硬件信息') }}</p>

    <div class="overview-grid">
      <el-card shadow="never" :header="t('系统')" class="wt-card">
        <el-descriptions :column="1" border class="wt-desc" :label-style="descLabelStyle">
          <el-descriptions-item :label="t('主机名')">{{
            overview?.hostname || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('架构')">{{
            overview?.arch || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('操作系统')">
            {{ overview?.osName || '—' }}
            <template v-if="overview?.osBuild">（Build {{ overview.osBuild }}）</template>
          </el-descriptions-item>
          <el-descriptions-item :label="t('激活状态')">
            <el-tag :type="actType">{{ t(overview?.activationStatus || '—') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('分辨率')">{{
            overview?.resolution || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('IP 地址')">{{
            (overview?.ips || []).join(', ') || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('磁盘')">{{
            (overview?.disks || []).join('  ·  ') || '—'
          }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="never" :header="t('硬件')" class="wt-card" v-loading="detailLoading">
        <el-descriptions :column="1" border class="wt-desc" :label-style="descLabelStyle">
          <el-descriptions-item :label="t('制造商')">{{
            overview?.manufacturer || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('型号')">{{
            overview?.model || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('主板')">{{
            overview?.board || '—'
          }}</el-descriptions-item>
          <el-descriptions-item label="BIOS">{{ overview?.bios || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('处理器')">
            {{ overview?.cpu || '—' }}
            <template v-if="overview?.cpuCores">{{
              t('（{0} 逻辑处理器）', [overview.cpuCores])
            }}</template>
          </el-descriptions-item>
          <el-descriptions-item :label="t('内存')">{{ memText() }}</el-descriptions-item>
          <el-descriptions-item :label="t('内存条')">{{
            t(overview?.memoryModules || '—')
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('硬盘')">{{
            (overview?.physicalDisks || []).join('  ·  ') || '—'
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('显卡')">{{
            (overview?.gpus || []).join('  ·  ') || '—'
          }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
  align-items: start;
  gap: 16px;
}
.overview-grid .wt-card {
  margin-bottom: 0;
}
.overview-grid :deep(.el-card__header) {
  padding: 12px 14px !important;
}
.overview-grid :deep(.el-card__body) {
  padding: 12px !important;
}
.overview-grid :deep(.el-descriptions__label) {
  width: 112px !important;
  min-width: 112px !important;
  max-width: 112px !important;
}
.overview-grid :deep(.el-descriptions__cell) {
  padding: 7px 9px !important;
}
.overview-grid :deep(.el-descriptions__content) {
  overflow-wrap: anywhere;
  word-break: normal;
}
@media (max-width: 900px) {
  .overview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
