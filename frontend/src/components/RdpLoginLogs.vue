<script setup>
import { computed, ref } from 'vue'
import { api } from '../api/index.js'
import { locale, t } from '../i18n/index.js'

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
  <el-card class="wt-card remote-logins" shadow="never" data-testid="remote-login-logs">
    <div class="remote-toolbar">
      <label
        ><span>{{ t('时间范围') }}</span
        ><el-select v-model="days" :disabled="loading" :aria-label="t('时间范围')">
          <el-option
            v-for="n in [1, 7, 30, 90]"
            :key="n"
            :value="n"
            :label="t('最近 {0} 天', [n])"
          /> </el-select
      ></label>
      <label
        ><span>{{ t('条数上限') }}</span
        ><el-select v-model="limit" :disabled="loading" :aria-label="t('条数上限')">
          <el-option
            v-for="n in [100, 300, 1000]"
            :key="n"
            :value="n"
            :label="String(n)"
          /> </el-select
      ></label>
      <el-button type="primary" :disabled="busy" :loading="loading" @click="readEvents">{{
        t('读取记录')
      }}</el-button>
      <el-button :disabled="loading || !report" @click="exportReport">{{
        t('导出诊断报告')
      }}</el-button>
      <el-button :disabled="loading || (!report && !error)" @click="clearResults">{{
        t('清空结果')
      }}</el-button>
    </div>
    <p class="wt-hint">{{ t('清空结果仅清除当前显示，不删除 Windows 原始日志。') }}</p>
    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      :title="
        report ? t('读取失败，下面保留上次结果；原因见日志说明。') : t('读取失败，请查看日志说明。')
      "
    />
    <template v-if="report">
      <el-alert
        v-if="report.partial || report.truncated"
        type="warning"
        :closable="false"
        :title="t('日志可能不完整，请查看日志说明。')"
      />
      <p class="wt-hint">{{ t('采集时间（UTC）') }} {{ time(report.collectedAt) }}</p>
      <div class="remote-stats">
        <div v-for="[label, value] in stats" :key="label">
          <span>{{ t(label) }}</span
          ><strong>{{ value }}</strong>
        </div>
      </div>
      <p class="wt-hint">
        {{ t('网络认证失败（未确认 RDP）') }}: {{ report.summary?.networkFailures || 0 }} ·
        {{ t('重新连接成功') }}: {{ report.summary?.reconnects || 0 }}
      </p>
      <el-collapse>
        <el-collapse-item :title="t('登录分析与来源汇总')" name="analysis">
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
      </el-collapse>
      <div class="remote-toolbar remote-filters">
        <el-select
          v-model="category"
          clearable
          :placeholder="t('全部分类')"
          :aria-label="t('日志分类')"
        >
          <el-option v-for="(label, key) in labels" :key="key" :value="key" :label="t(label)" />
        </el-select>
        <el-input
          v-model="search"
          clearable
          :placeholder="t('搜索账户、来源 IP、事件或状态码')"
          :aria-label="t('搜索登录记录')"
        />
      </div>
      <el-table
        :data="rows"
        data-testid="login-event-table"
        max-height="450"
        :empty-text="t('暂无记录；日志未启用或已清除时也可能为空。')"
        @row-click="selected = $event"
      >
        <el-table-column :label="t('时间（UTC）')" min-width="185"
          ><template #default="{ row }">{{ time(row.time) }}</template></el-table-column
        >
        <el-table-column :label="t('事件')" min-width="170"
          ><template #default="{ row }">{{
            t(labels[row.category] || '其他事件')
          }}</template></el-table-column
        >
        <el-table-column prop="user" :label="t('用户名')" min-width="140" show-overflow-tooltip />
        <el-table-column
          prop="address"
          :label="t('来源地址')"
          min-width="140"
          show-overflow-tooltip
        />
        <el-table-column :label="t('失败原因')" min-width="180" show-overflow-tooltip
          ><template #default="{ row }">{{ t(row.reason || '—') }}</template></el-table-column
        >
        <el-table-column :label="t('操作')" width="88" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click.stop="selected = row">{{
              t('查看详情')
            }}</el-button></template
          ></el-table-column
        >
      </el-table>
    </template>
    <el-collapse v-if="error || report?.warnings?.length" class="wt-actions--mt">
      <el-collapse-item :title="t('日志说明')" name="notes">
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
        <h3>{{ t('结构化字段') }}</h3>
        <el-table :data="selected.data || []" max-height="250"
          ><el-table-column prop="name" :label="t('字段')" min-width="160" /><el-table-column
            prop="value"
            :label="t('值')"
            min-width="180"
            show-overflow-tooltip
        /></el-table>
        <h3>{{ t('事件原文') }}</h3>
        <pre>{{ selected.detail || t('消息资源不可用，请查看结构化字段。') }}</pre>
        <p v-if="selected.limited" class="wt-hint">{{ t('原文或 XML 过长，已截断。') }}</p>
        <h3>XML</h3>
        <pre>{{ selected.xml || '—' }}</pre>
      </template>
    </el-drawer>
  </el-card>
</template>

<style scoped>
.remote-toolbar {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}
.remote-toolbar label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 150px;
  max-width: 100%;
}
.remote-toolbar .el-button + .el-button {
  margin-left: 0;
}
.remote-filters {
  margin: 14px 0;
}
.remote-filters .el-select {
  width: 220px;
  max-width: 100%;
}
.remote-filters .el-input {
  width: 300px;
  max-width: 100%;
}
.remote-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: 12px;
  margin: 12px 0;
}
.remote-stats > div {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.remote-stats strong {
  font-size: 20px;
}
.remote-text,
pre {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
</style>
