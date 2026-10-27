<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/index.js'
import { locale, t } from '../i18n/index.js'
import '../components/log-results.css'

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
const filtersOpen = ref(false)
const activeFilters = computed(() => Number(!!category.value) + Number(!!search.value.trim()))
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
  <div data-view="systemlogs" class="system-logs log-results">
    <el-card shadow="never" class="wt-card">
      <div class="log-actions">
        <el-button type="primary" :loading="loading" @click="refresh">{{
          t('读取并分析')
        }}</el-button>
        <el-button
          :type="activeFilters ? 'primary' : 'default'"
          :aria-expanded="filtersOpen"
          aria-controls="system-log-options"
          @click="filtersOpen = !filtersOpen"
          >{{ t('筛选与范围') }}{{ activeFilters ? ' · ' + activeFilters : '' }}</el-button
        >
        <el-button :disabled="loading || !report" @click="exportReport">{{
          t('导出诊断报告')
        }}</el-button>
        <el-button
          :disabled="loading || (!report && !error)"
          :title="t('清空结果仅清除当前显示，不删除 Windows 原始日志。')"
          @click="clearResults"
          >{{ t('清空结果') }}</el-button
        >
      </div>
      <div v-show="filtersOpen" id="system-log-options" class="log-options">
        <label
          ><span>{{ t('时间范围') }}</span
          ><el-select v-model="days" :disabled="loading" :aria-label="t('时间范围')"
            ><el-option
              v-for="n in [1, 7, 30, 90]"
              :key="n"
              :value="n"
              :label="t('最近 {0} 天', [n])" /></el-select
        ></label>
        <label
          ><span>{{ t('条数上限') }}</span
          ><el-select v-model="limit" :disabled="loading" :aria-label="t('条数上限')"
            ><el-option
              v-for="n in [100, 300, 1000]"
              :key="n"
              :value="n"
              :label="String(n)" /></el-select
        ></label>
        <label
          ><span>{{ t('事件类型') }}</span
          ><el-select
            v-model="category"
            clearable
            :placeholder="t('事件类型（全部）')"
            :aria-label="t('事件类型（全部）')"
            ><el-option
              v-for="name in categories"
              :key="name"
              :value="name"
              :label="t(names[name])" /></el-select
        ></label>
        <label
          ><span>{{ t('搜索') }}</span
          ><el-input v-model="search" clearable :placeholder="t('搜索来源、事件 ID、进程或详情')"
        /></label>
      </div>
      <el-alert
        v-if="error"
        type="error"
        :closable="false"
        :title="
          report ? t('本次读取失败，下面保留的是上次成功结果。') : t('读取失败，请查看日志说明。')
        "
        class="log-result-notice"
      />
      <el-alert
        v-if="report?.truncated"
        type="warning"
        :closable="false"
        :title="t('日志可能不完整，请查看日志说明。')"
        class="log-result-notice"
      />
      <div v-if="report" class="log-summary" data-testid="log-summary">
        <span v-for="[label, value] in stats" :key="label"
          >{{ label }} <strong>{{ value }}</strong></span
        >
        <span class="log-scope">{{ t('最近 {0} 天', [loadedScope?.days || days]) }} · UTC</span>
      </div>
      <p v-if="activeFilters" class="wt-hint">
        {{ t('筛选结果 {0} / {1} 条', [rows.length, report?.summary?.total || 0]) }}
      </p>
      <el-table
        :data="rows"
        row-key="recordId"
        stripe
        highlight-current-row
        max-height="540"
        class="log-table log-result-table"
        v-loading="loading"
        @row-click="inspect"
        :empty-text="report ? t('当前条件下无记录；不代表系统没有故障。') : t('请读取系统日志')"
      >
        <el-table-column :label="t('时间（UTC）')" width="178"
          ><template #default="{ row }"
            ><span class="log-time">{{ time(row.time) }}</span></template
          ></el-table-column
        >
        <el-table-column :label="t('事件类型')" min-width="130" show-overflow-tooltip
          ><template #default="{ row }"
            ><el-tag
              :type="['unexpected', 'bugcheck'].includes(row.category) ? 'danger' : 'info'"
              effect="plain"
              >{{ t(names[row.category]) }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column :label="t('记录解读')" min-width="220" show-overflow-tooltip
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
      <el-collapse class="log-extra">
        <el-collapse-item :title="t('日志说明')" name="notes">
          <p>
            {{ t('41 或 6008 表示上次未正常关机，不能单独证明断电；多条事件也不等于多次重启。') }}
          </p>
          <p v-if="report" class="wt-hint">
            {{ t('采集时间（UTC）') }} {{ time(report.collectedAt) }} · {{ t('起始时间（UTC）') }}
            {{ time(report.startTime) }}
          </p>
          <p class="wt-hint">
            {{
              t('只读取日志，不更改审计或系统配置。导出包含主机、账户及事件详情，分享前请核对。')
            }}
          </p>
          <p class="wt-hint">{{ t('清空结果仅清除当前显示，不删除 Windows 原始日志。') }}</p>
          <p v-if="error" role="alert" class="wt-detail">{{ error }}</p>
          <p v-for="warning in report?.warnings || []" :key="warning" class="wt-detail">
            {{ t(warning) }}
          </p>
          <p class="wt-hint">
            {{ t('点击记录查看原文、结构化字段和关联证据；列表支持横向滚动。') }}
          </p>
        </el-collapse-item>
      </el-collapse>
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
          <el-collapse
            ><el-collapse-item :title="t('前后十分钟的关联记录')" name="evidence">
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
            </el-collapse-item></el-collapse
          >
        </section>
        <el-collapse class="log-extra">
          <el-collapse-item :title="t('事件原文')" name="message">
            <pre>{{ selected.message || t('消息资源不可用，请查看结构化字段。') }}</pre>
          </el-collapse-item>
          <el-collapse-item :title="t('结构化字段')" name="fields"
            ><el-table :data="selected.data || []" max-height="300"
              ><el-table-column prop="name" :label="t('名称')" min-width="160" /><el-table-column
                prop="value"
                :label="t('值')"
                min-width="180"
                show-overflow-tooltip /></el-table
          ></el-collapse-item>
          <el-collapse-item title="XML" name="xml">
            <pre>{{ selected.xml }}</pre>
          </el-collapse-item>
        </el-collapse>
        <p v-if="selected.limited" class="wt-hint">{{ t('原文或 XML 过长，已截断。') }}</p>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.log-time {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.log-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  overflow-wrap: anywhere;
}
.log-section {
  margin-top: 18px;
}
.log-section h3 {
  margin: 0 0 10px;
}
.log-section li {
  line-height: 1.7;
}
.wt-detail,
pre {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
</style>
