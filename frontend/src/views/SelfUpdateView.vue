<script setup>
import { t } from '../i18n/index.js'

defineProps({
  info: { type: Object, default: null },
  busy: { type: Boolean, default: false },
  checking: { type: Boolean, default: false },
  autoDownload: { type: Boolean, default: true },
})

defineEmits(['check', 'download', 'apply', 'cancel', 'update:autoDownload'])
</script>

<template>
  <div data-view="selfupdate">
    <p class="page-desc">
      {{ t('检测 GitHub 仓库最新 Release，有更新时自动下载并校验，确认后安装并重启') }}
    </p>

    <el-checkbox
      :model-value="autoDownload"
      :disabled="busy || checking"
      @update:model-value="$emit('update:autoDownload', $event)"
      >{{ t('发现更新时自动下载') }}</el-checkbox
    >
    <el-card shadow="never" :header="t('版本信息')" class="wt-card">
      <div class="wt-status-block">
        <div class="wt-status-line">
          <span class="wt-status-label">{{ t('当前') }}</span>
          <el-tag size="small">{{ info?.currentVersion ? 'v' + info.currentVersion : '—' }}</el-tag>
          <template v-if="info?.latestVersion">
            <span class="wt-status-label">{{ t('最新') }}</span>
            <el-tag :type="info.hasUpdate ? 'warning' : 'success'" size="small">
              v{{ info.latestVersion }}
            </el-tag>
          </template>
          <el-tag
            v-if="info?.hasUpdate && info?.downloaded && info?.verified"
            type="success"
            effect="dark"
            >{{ t('已下载·已校验') }}</el-tag
          >
          <el-tag v-else-if="info?.hasUpdate && info?.downloaded" type="warning" effect="dark">{{
            t('已下载·未校验')
          }}</el-tag>
          <el-tag v-else-if="info?.hasUpdate" type="warning" effect="dark">{{
            t('有更新')
          }}</el-tag>
          <el-tag
            v-else-if="info?.latestVersion && !info?.hasUpdate"
            type="success"
            effect="dark"
            >{{ t('已是最新') }}</el-tag
          >
        </div>
        <p v-if="info?.assetName" class="wt-detail">{{ t('安装包：') }}{{ info.assetName }}</p>
        <p v-if="info?.assetSHA256" class="wt-detail">SHA256：{{ info.assetSHA256 }}</p>
        <p v-if="info?.downloadPath" class="wt-detail">
          {{ t('本地路径：') }}{{ info.downloadPath }}
        </p>
        <p v-if="info?.error" class="wt-detail wt-detail--err">{{ t(info.error) }}</p>
        <p v-else-if="info?.notes" class="wt-notes">{{ info.notes }}</p>
        <p v-else class="wt-detail">{{ t('点击「检查更新」从 GitHub 仓库获取最新版本信息。') }}</p>
      </div>

      <div v-if="info?.downloading" class="wt-status-block" aria-live="polite">
        <el-progress
          :percentage="
            info.assetSize
              ? Math.min(100, Math.floor((info.downloadedBytes * 100) / info.assetSize))
              : 0
          "
          :indeterminate="!info.assetSize"
        />
        <p class="wt-detail">
          {{ t('已下载') }} {{ (info.downloadedBytes / 1048576).toFixed(1) }} MB ·
          {{ (info.bytesPerSecond / 1048576).toFixed(1) }} MB/s
        </p>
        <el-button type="warning" @click="$emit('cancel')">{{ t('取消下载') }}</el-button>
      </div>
      <div class="wt-actions">
        <el-button
          type="primary"
          :loading="checking"
          :disabled="busy || checking"
          @click="$emit('check')"
          >{{ t('检查更新') }}</el-button
        >
        <el-button
          type="warning"
          :disabled="busy || checking || !info?.hasUpdate || (info?.downloaded && info?.verified)"
          @click="$emit('download')"
          >{{ t('下载更新') }}</el-button
        >
        <el-button
          type="success"
          :disabled="busy || checking || !info?.downloaded || !info?.verified"
          @click="$emit('apply')"
          >{{ t('安装并重启') }}</el-button
        >
      </div>
    </el-card>
  </div>
</template>
