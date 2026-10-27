<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/index.js'
import { locale, t } from '../i18n/index.js'

const days = ref(7),
  limit = ref(300),
  category = ref(''),
  search = ref('')
const loading = ref(false),
  error = ref(''),
  report = ref(null),
  selected = ref(null),
  finding = ref(null)
const loadedScope = ref(null)
const categories = [
  'unexpected',
  'bugcheck',
  'request',
  'boot',
  'shutdown',
  'hardware',
  'storage',
  'dump',
  'service',
  'update',
  'clock',
  'logclear',
  'other',
]
const names = {
  unexpected: '异常关机记录',
  bugcheck: '停止错误',
  request: '关机/重启请求',
  boot: '启动记录',
  shutdown: '关机过程',
  hardware: '硬件报告',
  storage: '存储异常',
  dump: '转储异常',
  service: '服务异常',
  update: '更新记录',
  clock: '时间调整',
  logclear: '日志清除',
  other: '其他',
}
const rows = computed(() => {
  const query = search.value.trim().toLowerCase()
  return (report.value?.events || []).filter(
    (e) =>
      (!category.value || e.category === category.value) &&
      (!query ||
        `${e.provider} ${e.id} ${e.message} ${t(e.meaning)} ${e.computer}`
          .toLowerCase()
          .includes(query)),
  )
})
const evidence = computed(() => {
  const ids = new Set(finding.value?.evidence || [])
  return (report.value?.events || []).filter((e) => ids.has(e.recordId))
})
const stats = computed(() => [
  [t('已读取事件'), report.value?.summary?.total ?? '—'],
  [t('异常关机记录'), report.value?.summary?.unexpected ?? '—'],
  [t('停止错误'), report.value?.summary?.bugchecks ?? '—'],
  [t('关机/重启请求'), report.value?.summary?.requests ?? '—'],
])
function time(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    timeZone: 'UTC',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}
async function refresh() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  const scope = { days: days.value, limit: limit.value }
  try {
    report.value = await api.getSystemEventReport(scope.days, scope.limit)
    loadedScope.value = scope
    selected.value = null
    finding.value = null
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}
function inspect(event) {
  selected.value = event
  finding.value = (report.value?.findings || []).find((f) => f.recordId === event.recordId) || null
}
function closeDetails() {
  selected.value = null
  finding.value = null
}
function clearResults() {
  if (loading.value) return
  closeDetails()
  report.value = null
  loadedScope.value = null
  error.value = ''
  category.value = ''
  search.value = ''
}
function exportReport() {
  const data = {
    ...report.value,
    exportedAt: new Date().toISOString(),
    lastReadError: error.value,
    requestedDays: loadedScope.value?.days,
    requestedLimit: loadedScope.value?.limit,
    note: 'System events and heuristic correlations; not a verified root cause.',
  }
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(data, null, 2)], { type: 'application/json;charset=utf-8' }),
  )
  const a = document.createElement('a')
  a.href = url
  a.download = `WinToolBox-system-events-${new Date().toISOString().replace(/[:.]/g, '-')}.json`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
onMounted(refresh)
defineExpose({ refresh })
</script>

<template>
  <div data-view="systemlogs" class="system-logs">
    <p class="page-desc">
      {{ t('按时间梳理开机、关机、重启和故障证据，辅助排查 Windows Server 与 VPS 异常重启。') }}
    </p>
    <el-card shadow="never" class="wt-card">
      <div class="log-toolbar">
        <label class="log-filter"
          ><span>{{ t('时间范围') }}</span
          ><el-select v-model="days" :disabled="loading" :aria-label="t('时间范围')"
            ><el-option
              v-for="n in [1, 7, 30, 90]"
              :key="n"
              :value="n"
              :label="t('最近 {0} 天', [n])" /></el-select
        ></label>
        <label class="log-filter"
          ><span>{{ t('条数上限') }}</span
          ><el-select v-model="limit" :disabled="loading" :aria-label="t('条数上限')"
            ><el-option
              v-for="n in [100, 300, 1000]"
              :key="n"
              :value="n"
              :label="String(n)" /></el-select
        ></label>
        <el-button type="primary" :loading="loading" @click="refresh">{{
          t('读取并分析')
        }}</el-button>
        <el-button :disabled="loading || !report" @click="exportReport">{{
          t('导出诊断报告')
        }}</el-button>
        <el-button :disabled="loading || (!report && !error)" @click="clearResults">{{
          t('清空结果')
        }}</el-button>
      </div>
      <p class="wt-hint">
        {{ t('只读取日志，不更改审计或系统配置。导出包含主机、账户及事件详情，分享前请核对。') }}
        {{ t('清空结果仅清除当前显示，不删除 Windows 原始日志。') }}
      </p>
      <el-alert v-if="error" type="error" :closable="false" :title="error" class="log-notice" />
      <p v-if="error && report" class="wt-hint">
        {{ t('本次读取失败，下面保留的是上次成功结果。') }}
      </p>
      <p v-if="report" class="wt-hint">
        {{ t('采集时间（UTC）') }} {{ time(report.collectedAt) }} · {{ t('起始时间（UTC）') }}
        {{ time(report.startTime) }}
      </p>
    </el-card>
    <div class="log-stats">
      <div v-for="[label, value] in stats" :key="label" class="log-stat">
        <span>{{ label }}</span
        ><strong>{{ value }}</strong>
      </div>
    </div>
    <el-alert
      type="info"
      :closable="false"
      show-icon
      :title="t('41 或 6008 表示上次未正常关机，不能单独证明断电；多条事件也不等于多次重启。')"
      class="wt-card"
    />
    <el-card shadow="never" class="wt-card" :header="t('系统事件时间线')" v-loading="loading">
      <el-collapse v-if="report?.warnings?.length" class="log-warnings"
        ><el-collapse-item :title="t('读取范围与分析限制')"
          ><p v-for="warning in report.warnings" :key="warning" class="wt-detail">
            {{ t(warning) }}
          </p></el-collapse-item
        ></el-collapse
      >
      <div class="log-toolbar">
        <el-select
          v-model="category"
          clearable
          :placeholder="t('事件类型（全部）')"
          :aria-label="t('事件类型（全部）')"
          style="width: 180px"
          ><el-option v-for="name in categories" :key="name" :value="name" :label="t(names[name])"
        /></el-select>
        <el-input
          v-model="search"
          clearable
          :placeholder="t('搜索来源、事件 ID、进程或详情')"
          style="flex: 1; min-width: 180px"
        />
        <span class="wt-count">{{ t('共 {0} 条', [rows.length]) }}</span>
      </div>
      <el-table
        :data="rows"
        row-key="recordId"
        stripe
        highlight-current-row
        max-height="540"
        class="log-table"
        @row-click="inspect"
        :empty-text="report ? t('当前条件下无记录；不代表系统没有故障。') : t('请读取系统日志')"
      >
        <el-table-column :label="t('时间（UTC）')" width="178"
          ><template #default="{ row }"
            ><span class="log-time">{{ time(row.time) }}</span></template
          ></el-table-column
        >
        <el-table-column prop="id" label="ID" width="76" />
        <el-table-column :label="t('事件类型')" min-width="130"
          ><template #default="{ row }"
            ><el-tag
              :type="['unexpected', 'bugcheck'].includes(row.category) ? 'danger' : 'info'"
              effect="plain"
              >{{ t(names[row.category]) }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column
          prop="provider"
          :label="t('事件来源')"
          min-width="170"
          show-overflow-tooltip
        />
        <el-table-column :label="t('记录解读')" min-width="270" show-overflow-tooltip
          ><template #default="{ row }">{{ t(row.meaning) }}</template></el-table-column
        >
        <el-table-column :label="t('操作')" width="88" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click.stop="inspect(row)">{{
              t('查看详情')
            }}</el-button></template
          ></el-table-column
        >
      </el-table>
      <p class="wt-hint">{{ t('点击记录查看原文、结构化字段和关联证据；列表支持横向滚动。') }}</p>
    </el-card>
    <el-drawer
      :model-value="!!selected"
      @close="closeDetails"
      :title="t('事件详情与排查建议')"
      size="min(760px, 94vw)"
      destroy-on-close
    >
      <template v-if="selected">
        <div class="log-meta">
          <el-tag>{{ selected.provider }} / {{ selected.id }}</el-tag
          ><span>{{ time(selected.time) }} UTC</span>
        </div>
        <p class="wt-detail">
          {{ selected.computer }} · {{ t('记录编号') }} {{ selected.recordId }}
        </p>
        <el-alert type="info" :closable="false" :title="t(selected.meaning)" class="log-notice" />
        <section v-if="finding" class="log-section">
          <h3>{{ t('排查建议') }}</h3>
          <p v-if="finding.conclusion !== selected.meaning">{{ t(finding.conclusion) }}</p>
          <p v-if="finding.stopCode">
            {{ t('停止代码') }}: <code>{{ finding.stopCode }}</code>
          </p>
          <ul>
            <li v-for="step in finding.nextSteps" :key="step">{{ t(step) }}</li>
          </ul>
          <h3>{{ t('前后十分钟的关联记录') }}</h3>
          <el-table :data="evidence" max-height="230"
            ><el-table-column :label="t('时间（UTC）')" width="176"
              ><template #default="{ row }">{{ time(row.time) }}</template></el-table-column
            ><el-table-column prop="id" label="ID" width="70" /><el-table-column
              prop="provider"
              :label="t('事件来源')"
              show-overflow-tooltip
            /><el-table-column :label="t('操作')" width="80"
              ><template #default="{ row }"
                ><el-button link @click="inspect(row)">{{ t('详情') }}</el-button></template
              ></el-table-column
            ></el-table
          >
        </section>
        <section class="log-section">
          <h3>{{ t('事件原文') }}</h3>
          <pre>{{ selected.message || t('消息资源不可用，请查看结构化字段。') }}</pre>
          <p v-if="selected.limited" class="wt-hint">{{ t('原文或 XML 过长，已截断。') }}</p>
        </section>
        <section class="log-section">
          <h3>{{ t('结构化字段') }}</h3>
          <el-table :data="selected.data || []" max-height="300"
            ><el-table-column prop="name" :label="t('名称')" width="180" /><el-table-column
              prop="value"
              :label="t('值')"
              show-overflow-tooltip
          /></el-table>
        </section>
        <el-collapse
          ><el-collapse-item title="XML">
            <pre>{{ selected.xml }}</pre>
          </el-collapse-item></el-collapse
        >
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.log-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}
.log-toolbar :deep(.el-button) {
  margin-left: 0;
}
.log-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--wt-text-secondary);
}
.log-filter .el-select {
  width: 132px;
}
.log-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 18px;
}
.log-stat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: #fff;
  border: 1px solid var(--wt-border);
  border-radius: 10px;
  padding: 16px;
}
.log-stat span {
  font-size: 13px;
  color: var(--wt-text-secondary);
}
.log-stat strong {
  font-size: 26px;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.log-time {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.log-table {
  margin-top: 14px;
  cursor: pointer;
}
.log-notice {
  margin-top: 14px;
}
.log-warnings {
  margin-bottom: 16px;
}
.log-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.log-section {
  margin-top: 22px;
}
.log-section h3 {
  font-size: 15px;
  margin: 0 0 10px;
}
.log-section p,
.log-section li {
  font-size: 14px;
  line-height: 1.7;
}
.log-section li + li {
  margin-top: 8px;
}
pre {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: Consolas, 'Microsoft YaHei UI', monospace;
  font-size: 13px;
  line-height: 1.65;
  background: #f6f8fb;
  padding: 14px;
  border: 1px solid var(--wt-border);
  border-radius: 8px;
}
@media (max-width: 1120px) {
  .log-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
