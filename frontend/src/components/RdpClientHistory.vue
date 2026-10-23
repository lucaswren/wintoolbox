<script setup>
import { computed, ref } from 'vue'
import { t } from '../i18n/index.js'

const props = defineProps({
  history: { type: Array, default: () => [] },
  historyLoading: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})
defineEmits(['clear-history', 'clear-history-kind', 'delete-history', 'refresh-history'])

const kindLabel = computed(() => ({
  mru: t('最近主机'),
  server: t('已保存主机'),
  credential: t('凭据'),
  file: t('文件'),
  cache: t('缓存'),
}))

const kindTag = {
  mru: 'primary',
  server: 'success',
  credential: 'warning',
  file: 'info',
  cache: 'danger',
}

function labelKind(kind) {
  return kindLabel.value[kind] || kind || '—'
}

function tagKind(kind) {
  return kindTag[kind] || 'info'
}

const searchText = ref('')
const selectedKind = ref('')

// Normalize possible field casing coming from backend (e.g. `Kind` vs `kind`).
// This prevents the filter/table from showing empty when JSON key casing differs.
const normalizedHistory = computed(() =>
  (props.history || []).map((row) => {
    const kind = (row.kind ?? row.Kind ?? '').toString().trim().toLowerCase()
    return {
      kind,
      host: row.host ?? row.Host ?? '',
      username: row.username ?? row.Username ?? '',
      source: row.source ?? row.Source ?? '',
      detail: row.detail ?? row.Detail ?? '',
      sid: row.sid ?? row.Sid ?? '',
    }
  }),
)

const filteredHistory = computed(() => {
  const q = searchText.value.trim().toLowerCase()
  const kind = selectedKind.value
  return normalizedHistory.value.filter((row) => {
    if (kind && row.kind !== kind) return false
    if (!q) return true
    const haystack =
      `${row.host || ''} ${row.username || ''} ${row.source || ''} ${row.detail || ''} ${row.kind || ''}`.toLowerCase()
    return haystack.includes(q)
  })
})

const selectedKindCount = computed(() => {
  if (!selectedKind.value) return 0
  return normalizedHistory.value.filter((r) => r.kind === selectedKind.value).length
})
</script>

<template>
  <el-card shadow="never" class="wt-card" v-loading="historyLoading">
    <p class="wt-detail" style="margin-top: 0">
      {{
        t(
          '下列为本地远程桌面客户端留下的历史主机、用户名提示、TERMSRV 凭据、相关文件与位图缓存。清理不影响远程桌面服务开关与端口。',
        )
      }}
    </p>

    <div class="wt-actions wt-actions--mt">
      <el-select
        v-model="selectedKind"
        :placeholder="t('类型（全部）')"
        clearable
        style="width: 160px"
      >
        <el-option v-for="(label, key) in kindLabel" :key="key" :label="label" :value="key" />
      </el-select>
      <el-input
        v-model="searchText"
        :placeholder="t('搜索主机 / 用户名 / 来源 / 详情')"
        clearable
        style="max-width: 520px; flex: 1"
      />
    </div>

    <div class="wt-actions wt-actions--mt">
      <el-button :disabled="busy || historyLoading" @click="$emit('refresh-history')">{{
        t('刷新列表')
      }}</el-button>
      <el-button
        type="danger"
        plain
        :disabled="busy || historyLoading || !history.length"
        @click="$emit('clear-history')"
        >{{ t('清理全部记录') }}</el-button
      >
      <el-button
        v-if="selectedKind"
        type="warning"
        :disabled="busy || historyLoading || selectedKindCount === 0"
        @click="$emit('clear-history-kind', selectedKind)"
        >{{ t('清理当前类型') }}</el-button
      >
      <span class="wt-count">{{ t('共 {0} 条', [filteredHistory.length]) }}</span>
    </div>
    <el-table
      :data="filteredHistory"
      stripe
      :empty-text="t('暂无连接记录')"
      highlight-current-row
      style="width: 100%; margin-top: 12px"
    >
      <el-table-column :label="t('类型')" width="140">
        <template #default="{ row }">
          <el-tag :type="tagKind(row.kind)" style="white-space: nowrap">{{
            labelKind(row.kind)
          }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column
        prop="host"
        :label="t('主机 / 名称')"
        min-width="140"
        show-overflow-tooltip
      />
      <el-table-column prop="username" :label="t('用户名')" min-width="110" show-overflow-tooltip>
        <template #default="{ row }">{{ row.username || '—' }}</template>
      </el-table-column>
      <el-table-column prop="source" :label="t('来源')" min-width="110" show-overflow-tooltip />
      <el-table-column prop="detail" :label="t('详情')" min-width="160" show-overflow-tooltip />
      <el-table-column :label="t('操作')" width="90" fixed="right">
        <template #default="{ row }">
          <el-button
            link
            type="danger"
            :disabled="busy || historyLoading"
            @click="$emit('delete-history', row)"
            >{{ t('删除') }}</el-button
          >
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>
