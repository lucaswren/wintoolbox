<script setup>
import { computed, onMounted, ref } from 'vue'
import { locale, t } from '../i18n/index.js'
import { api } from '../api/index.js'

const props = defineProps({
  overview: { type: Object, default: null },
  networkSummary: { type: Object, default: null },
})
defineEmits(['open-logs'])

const health = ref(null)
const loading = ref(false)
const failed = ref(false)
const showSummary = ref(false)
const copying = ref(false)
const copyMessage = ref('')
const collectedAt = ref(0)

const disks = computed(() =>
  [...(props.overview?.diskSpace || [])].sort((a, b) => Number(b.system) - Number(a.system)),
)
const serviceNames = {
  TermService: '远程桌面',
  W32Time: 'Windows 时间服务',
  wuauserv: 'Windows 更新',
  MpsSvc: 'Windows 防火墙服务',
  WinDefend: 'Microsoft Defender',
  Spooler: '打印服务',
}
const serviceState = {
  running: '运行中',
  stopped: '已停止',
  missing: '本机未安装',
  paused: '已暂停',
  startpending: '正在启动',
  stoppending: '正在停止',
  continuepending: '状态切换中',
  pausepending: '状态切换中',
}

function localTime(time) {
  const date = new Date(time || NaN)
  return Number.isFinite(date.getTime()) ? date.toLocaleString(locale.value) : t('未知')
}

const uptime = computed(() => {
  const boot = Date.parse(health.value?.bootTime)
  if (!Number.isFinite(boot) || boot > collectedAt.value) return ''
  const minutes = Math.floor((collectedAt.value - boot) / 60000)
  const days = Math.floor(minutes / 1440)
  const hours = Math.floor((minutes % 1440) / 60)
  return days ? t('{0} 天 {1} 小时', [days, hours]) : t('{0} 小时 {1} 分钟', [hours, minutes % 60])
})

function diskText(disk) {
  return t('剩余 {0} / {1} GiB', [disk.freeGiB.toFixed(1), disk.totalGiB.toFixed(1)])
}

function diskLow(disk) {
  return disk.freeGiB < 2 || disk.freeGiB / disk.totalGiB < 0.1
}

function usedPercent(disk) {
  return Math.max(0, Math.min(100, Math.round((1 - disk.freeGiB / disk.totalGiB) * 100)))
}

const eventNames = {
  41: '检测到非正常关机后的启动',
  6008: '记录到上次关机异常',
  1001: '记录到系统停止错误',
}
const recentEvents = computed(() =>
  (health.value?.recentEvents || []).map((event) =>
    t('{0} · {1}（事件 {2}）', [
      localTime(event.time),
      t(eventNames[event.id] || '系统事件'),
      event.id,
    ]),
  ),
)

const networkLabels = {
  dns: {
    resolved: '已找到地址',
    literal: '已跳过',
    'not-found': '未找到地址',
    timeout: '超时',
    unavailable: '暂不可用',
  },
  ping: { reply: '有回应', 'no-reply': '无回应', unavailable: '暂不可用', skipped: '已跳过' },
  tcp: {
    connected: '可连通',
    unreachable: '网络不可达',
    refused: '拒绝连接',
    timeout: '超时',
    unavailable: '暂不可用',
    skipped: '已跳过',
  },
}

const summary = computed(() => {
  const os = props.overview
  const network = props.networkSummary
  const networkLine = network
    ? ['dns', 'ping', 'tcp']
        .map((key) => key.toUpperCase() + ' ' + t(networkLabels[key][network[key]] || '未知'))
        .join(' · ')
    : t('尚未运行网络检查。')
  const eventsNote = health.value?.eventsPartial ? t('部分日志无法读取。') : ''
  return [
    t('故障摘要'),
    t('Windows 版本：{0}', [
      os?.osName ? [os.osName, os.osBuild, os.arch].filter(Boolean).join(' · ') : t('未知'),
    ]),
    t('上次开机：{0}', [localTime(health.value?.bootTime)]),
    t('磁盘空间'),
    ...(disks.value.length
      ? disks.value.map((disk) => disk.drive + ' ' + diskText(disk))
      : [t('未知')]),
    t('网络检查') + ' · ' + networkLine,
    t('近 7 天的异常记录'),
    ...(recentEvents.value.length
      ? recentEvents.value
      : [health.value && !health.value.eventsPartial ? t('未找到所检查的异常事件。') : t('未知')]),
    eventsNote,
    failed.value && health.value ? t('本次读取失败，显示上次结果。') : '',
  ]
    .filter(Boolean)
    .join('\n')
})

async function refresh() {
  if (loading.value) return
  loading.value = true
  failed.value = false
  try {
    health.value = await api.getSystemHealth()
    collectedAt.value = Date.now()
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

async function copy() {
  if (copying.value) return
  copying.value = true
  copyMessage.value = ''
  try {
    await api.copyHealthSummary(summary.value)
    copyMessage.value = '已复制故障摘要。'
  } catch {
    copyMessage.value = '复制失败，可手动选择上方文字复制。'
  } finally {
    copying.value = false
  }
}

function preview() {
  copyMessage.value = ''
  showSummary.value = true
}

onMounted(refresh)
defineExpose({ refresh })
</script>

<template>
  <el-card shadow="never" :header="t('系统检查')" class="wt-card">
    <div class="wt-actions">
      <el-button :disabled="loading" @click="preview">{{ t('预览故障摘要') }}</el-button>
      <el-button @click="$emit('open-logs')">{{ t('打开系统日志') }}</el-button>
    </div>
    <p v-if="loading" role="status">{{ t('正在读取启动、服务和日志状态…') }}</p>
    <p v-else-if="failed" role="status">
      {{
        t(
          health
            ? '本次读取失败，显示上次结果。'
            : '暂时无法读取启动、服务和日志状态，请刷新重试。',
        )
      }}
    </p>
    <p v-if="health">
      {{ t('上次开机：{0}', [localTime(health.bootTime)]) }}
      <span v-if="uptime"> · {{ t('已运行 {0}', [uptime]) }}</span>
    </p>
    <h3>{{ t('磁盘空间') }}</h3>
    <p v-if="!disks.length" class="wt-hint">{{ t('暂无可显示的本地磁盘容量。') }}</p>
    <div v-for="disk in disks" :key="disk.drive" class="health-disk">
      <div class="health-disk-label">
        <strong
          >{{ disk.drive }} <span v-if="disk.system">{{ t('系统盘') }}</span></strong
        >
        <span>{{ diskText(disk) }} · {{ t('已用 {0}%', [usedPercent(disk)]) }}</span>
        <el-tag v-if="diskLow(disk)" type="warning">{{ t('空间偏低') }}</el-tag>
      </div>
      <el-progress
        :percentage="usedPercent(disk)"
        :show-text="false"
        :status="diskLow(disk) ? 'warning' : undefined"
        :aria-label="disk.drive + ' ' + diskText(disk)"
      />
    </div>

    <el-collapse v-if="health">
      <el-collapse-item :title="t('常用服务状态')" name="services">
        <p class="wt-hint">{{ t('有些服务只在需要时运行，停止也可能是正常状态。') }}</p>
        <dl class="health-services">
          <div v-for="service in health.services" :key="service.name">
            <dt>{{ t(serviceNames[service.name] || '未知') }}</dt>
            <dd>{{ t(serviceState[service.state] || '未知') }}</dd>
          </div>
        </dl>
      </el-collapse-item>
      <el-collapse-item :title="t('近 7 天的异常记录')" name="events">
        <p v-if="health.eventsPartial" role="status">{{ t('部分日志无法读取。') }}</p>
        <p v-if="!recentEvents.length && !health.eventsPartial">
          {{ t('未找到所检查的异常事件。') }}
        </p>
        <ul v-if="recentEvents.length">
          <li v-for="entry in recentEvents" :key="entry">{{ entry }}</li>
        </ul>
        <p class="wt-hint">{{ t('这里只显示每类异常最近一条。完整记录可在系统日志中查看。') }}</p>
      </el-collapse-item>
    </el-collapse>
  </el-card>

  <el-dialog v-model="showSummary" :title="t('故障摘要')" width="min(620px, 92vw)" append-to-body>
    <p>{{ t('摘要省略用户名、主机名、目标地址和 IP。分享前可先核对内容。') }}</p>
    <el-input
      :model-value="summary"
      type="textarea"
      :rows="9"
      readonly
      :aria-label="t('故障摘要')"
    />
    <p v-if="copyMessage" role="status">{{ t(copyMessage) }}</p>
    <template #footer>
      <el-button @click="showSummary = false">{{ t('取消') }}</el-button>
      <el-button type="primary" :loading="copying" @click="copy">{{ t('复制故障摘要') }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.health-disk {
  margin: 8px 0 16px;
}
.health-disk-label {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.health-services {
  margin: 0;
}
.health-services div {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.health-services dd {
  margin: 0;
}
</style>
