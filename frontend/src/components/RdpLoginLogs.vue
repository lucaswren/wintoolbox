<script setup>
import { computed, ref } from 'vue'
import { api } from '../api/index.js'
import { locale, t } from '../i18n/index.js'
import './log-results.css'

const props = defineProps({ busy: { type: Boolean, default: false } })
const days = ref(7),
  limit = ref(300),
  category = ref(''),
  search = ref('')
const loading = ref(false),
  report = ref(null),
  error = ref(''),
  selected = ref(null)
const loadedScope = ref(null)
const filtersOpen = ref(false)
const activeFilters = computed(() => Number(!!category.value) + Number(!!search.value.trim()))
const labels = {
  login: '登录成功',
  failure: '登录失败',
  'network-failure': '网络认证失败（未确认 RDP）',
  auth: '身份验证成功',
  desktop: '桌面已启动',
  logoff: '已注销',
  disconnect: '连接已断开',
  reconnect: '重新连接成功',
  logclear: '日志清除',
  other: '其他事件',
  session: '会话事件（来源未确认）',
}
const rows = computed(() => {
  const query = search.value.trim().toLowerCase()
  return (report.value?.events || []).filter(
    (e) =>
      (!category.value || e.category === category.value) &&
      (!query ||
        `${e.user} ${e.address} ${e.id} ${e.session} ${e.status} ${e.subStatus} ${e.detail} ${t(e.meaning)} ${t(e.reason)}`
          .toLowerCase()
          .includes(query)),
  )
})
const stats = computed(() => [
  ['已读取事件', report.value?.summary?.total ?? '—'],
  ['远程登录记录', report.value?.summary?.logins ?? '—'],
  ['远程登录失败', report.value?.summary?.failures ?? '—'],
  ['断连记录', report.value?.summary?.disconnects ?? '—'],
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
async function readEvents() {
  if (loading.value || props.busy) return
  loading.value = true
  error.value = ''
  const scope = { days: days.value, limit: limit.value }
  try {
    report.value = await api.getRdpLoginReport(scope.days, scope.limit)
    loadedScope.value = scope
    selected.value = null
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}
function clearResults() {
  if (loading.value) return
  report.value = null
  loadedScope.value = null
  selected.value = null
  error.value = ''
  category.value = ''
  search.value = ''
}
function exportReport() {
  const url = URL.createObjectURL(
    new Blob(
      [
        JSON.stringify(
          {
            ...report.value,
            exportedAt: new Date().toISOString(),
            lastReadError: error.value,
            requestedDays: loadedScope.value?.days,
            requestedLimit: loadedScope.value?.limit,
          },
          null,
          2,
        ),
      ],
      { type: 'application/json;charset=utf-8' },
    ),
  )
  const a = document.createElement('a')
  a.href = url
  a.download = `WinToolBox-remote-logins-${new Date().toISOString().replace(/[:.]/g, '-')}.json`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
</script>

<template>
  <el-card class="wt-card log-results remote-logins" shadow="never" data-testid="remote-login-logs">
    <div class="log-actions">
      <el-button type="primary" :disabled="busy" :loading="loading" @click="readEvents">{{
        t('读取记录')
      }}</el-button>
      <el-button
        :type="activeFilters ? 'primary' : 'default'"
        :aria-expanded="filtersOpen"
        aria-controls="remote-log-options"
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
    <div v-show="filtersOpen" id="remote-log-options" class="log-options">
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
        ><span>{{ t('日志分类') }}</span
        ><el-select
          v-model="category"
          clearable
          :placeholder="t('全部分类')"
          :aria-label="t('日志分类')"
          ><el-option
            v-for="(label, key) in labels"
            :key="key"
            :value="key"
            :label="t(label)" /></el-select
      ></label>
      <label
        ><span>{{ t('搜索登录记录') }}</span
        ><el-input
          v-model="search"
          clearable
          :placeholder="t('搜索账户、来源 IP、事件或状态码')"
          :aria-label="t('搜索登录记录')"
      /></label>
    </div>
    <el-alert
      v-if="error"
      class="log-result-notice"
      type="error"
      :closable="false"
      :title="
        report ? t('读取失败，下面保留上次结果；原因见日志说明。') : t('读取失败，请查看日志说明。')
      "
    />
    <el-alert
      v-if="report?.partial || report?.truncated"
      class="log-result-notice"
      type="warning"
      :closable="false"
      :title="t('日志可能不完整，请查看日志说明。')"
    />
    <template v-if="report">
      <div class="log-summary" data-testid="log-summary">
        <span v-for="[label, value] in stats" :key="label"
          >{{ t(label) }} <strong>{{ value }}</strong></span
        >
        <span class="log-scope">{{ t('最近 {0} 天', [loadedScope?.days || days]) }} · UTC</span>
      </div>
      <p v-if="activeFilters" class="wt-hint">
        {{ t('筛选结果 {0} / {1} 条', [rows.length, report.summary?.total || 0]) }}
      </p>
      <el-table
        :data="rows"
        data-testid="login-event-table"
        max-height="450"
        class="log-result-table"
        :empty-text="t('暂无记录；日志未启用或已清除时也可能为空。')"
        @row-click="selected = $event"
      >
        <el-table-column :label="t('时间（UTC）')" width="178"
          ><template #default="{ row }">{{ time(row.time) }}</template></el-table-column
        >
        <el-table-column :label="t('事件')" min-width="130" show-overflow-tooltip
          ><template #default="{ row }">{{
            t(labels[row.category] || '其他事件')
          }}</template></el-table-column
        >
        <el-table-column prop="user" :label="t('用户名')" min-width="110" show-overflow-tooltip />
        <el-table-column
          prop="address"
          :label="t('来源地址')"
          min-width="120"
          show-overflow-tooltip
        />
        <el-table-column :label="t('操作')" width="88" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click.stop="selected = row">{{
              t('查看详情')
            }}</el-button></template
          ></el-table-column
        >
      </el-table>
    </template>
    <el-collapse class="log-extra">
      <el-collapse-item v-if="report" :title="t('登录分析与来源汇总')" name="analysis">
        <p class="wt-hint">
          {{ t('网络认证失败（未确认 RDP）') }}: {{ report.summary?.networkFailures || 0 }} ·
          {{ t('重新连接成功') }}: {{ report.summary?.reconnects || 0 }}
        </p>
        <p v-if="!report.findings?.length" class="wt-hint">
          {{ t('当前样本未达到重复失败提示阈值，不代表没有异常。') }}
        </p>
        <p v-for="(item, index) in report.findings || []" :key="index" class="remote-text">
          {{ item.address }} · {{ item.count }} · {{ time(item.start) }} —
          {{ time(item.end) }} UTC<br />{{ t(item.meaning) }}
        </p>
        <el-table :data="report.sources || []" max-height="240">
          <el-table-column
            prop="address"
            :label="t('来源地址')"
            min-width="150"
            show-overflow-tooltip
          />
          <el-table-column prop="logins" :label="t('远程登录记录')" min-width="120" />
          <el-table-column prop="failures" :label="t('远程登录失败')" min-width="120" />
          <el-table-column
            prop="networkFailures"
            :label="t('网络认证失败（未确认 RDP）')"
            min-width="190"
          />
          <el-table-column :label="t('最近记录（UTC）')" min-width="180"
            ><template #default="{ row }">{{ time(row.lastSeen) }}</template></el-table-column
          >
        </el-table>
      </el-collapse-item>
      <el-collapse-item :title="t('日志说明')" name="notes">
        <p v-if="report" class="wt-hint">
          {{ t('采集时间（UTC）') }} {{ time(report.collectedAt) }} · {{ t('起始时间（UTC）') }}
          {{ time(report.startTime) }}
        </p>
        <p class="wt-hint">{{ t('清空结果仅清除当前显示，不删除 Windows 原始日志。') }}</p>
        <p v-if="error" role="alert" class="remote-text">{{ error }}</p>
        <p v-for="warning in report?.warnings || []" :key="warning" class="remote-text">
          {{ t(warning) }}
        </p>
      </el-collapse-item>
    </el-collapse>
    <el-drawer
      :model-value="!!selected"
      :title="t('登录事件详情')"
      size="min(760px, 94vw)"
      destroy-on-close
      @close="selected = null"
    >
      <template v-if="selected">
        <p class="remote-text">
          {{ time(selected.time) }} UTC · {{ selected.channel }} · ID {{ selected.id }} ·
          {{ t('记录编号') }} {{ selected.recordId }}
        </p>
        <p class="remote-text">
          {{ selected.user || '—' }} · {{ selected.address || '—' }} · {{ t('会话') }}
          {{ selected.session || '—' }}
        </p>
        <p class="remote-text">{{ t(selected.meaning) }}</p>
        <p v-if="selected.reason" class="remote-text">
          {{ t('失败原因') }}: {{ t(selected.reason) }} · Status {{ selected.status }} / SubStatus
          {{ selected.subStatus }}
        </p>
        <el-collapse class="log-extra">
          <el-collapse-item :title="t('事件原文')" name="message">
            <pre>{{ selected.detail || t('消息资源不可用，请查看结构化字段。') }}</pre>
          </el-collapse-item>
          <el-collapse-item :title="t('结构化字段')" name="fields">
            <el-table :data="selected.data || []" max-height="250"
              ><el-table-column prop="name" :label="t('字段')" min-width="160" /><el-table-column
                prop="value"
                :label="t('值')"
                min-width="180"
                show-overflow-tooltip
            /></el-table>
          </el-collapse-item>
          <el-collapse-item title="XML" name="xml">
            <pre>{{ selected.xml || '—' }}</pre>
          </el-collapse-item>
        </el-collapse>
        <p v-if="selected.limited" class="wt-hint">{{ t('原文或 XML 过长，已截断。') }}</p>
      </template>
    </el-drawer>
  </el-card>
</template>

<style scoped>
.remote-text,
pre {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
</style>
