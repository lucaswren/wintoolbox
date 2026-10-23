<script setup>
import { onMounted, ref } from 'vue'
import { api } from '../api/index.js'
import { t } from '../i18n/index.js'
import { useRdpSafety } from '../composables/useRdpSafety.js'

const props = defineProps({
  busy: { type: Boolean, default: false },
  section: { type: String, default: 'access' },
})
const { status, error, loading, refresh } = useRdpSafety()
const sources = ref('')
const working = ref(false)
const actionError = ref('')
const eventsRead = ref(false)
const events = ref([])
const eventWarnings = ref([])
const eventError = ref('')
const eventsLoading = ref(false)
onMounted(async () => {
  if (props.section !== 'access') return
  await refresh()
  sources.value = status.value?.addresses?.join('\n') || ''
})
async function apply() {
  working.value = true
  actionError.value = ''
  try {
    await ElMessageBox.confirm(
      t(
        '请确认已填入连接服务器时使用的公网 IP，并保留 VPS 控制台。保存后五分钟内重新连接并确认，否则自动恢复原设置。',
      ),
      t('请确认'),
      { type: 'warning', closeOnClickModal: false },
    )
    await api.applyRdpWhitelist(sources.value)
    await refresh()
    ElMessage.warning(t('配置已提交，等待重新连接确认'))
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      actionError.value = e?.message || String(e)
      ElMessage.error(t('保存失败，请查看诊断详情。'))
    }
    await refresh()
  } finally {
    working.value = false
  }
}
async function readEvents() {
  eventsLoading.value = true
  eventError.value = ''
  try {
    const result = await api.getRdpServerEvents()
    eventsRead.value = true
    events.value = result.events || []
    eventWarnings.value = result.warnings || []
  } catch (e) {
    eventError.value = e?.message || String(e)
  } finally {
    eventsLoading.value = false
  }
}
function eventLabel(id) {
  const labels = {
    21: '登录成功',
    22: '桌面已启动',
    23: '已注销',
    24: '连接已断开',
    25: '重新连接成功',
    39: '连接已断开',
    40: '连接已断开',
    1149: '身份验证成功',
    4624: '登录成功',
    4625: '登录失败',
    4634: '已注销',
    4778: '重新连接成功',
    4779: '连接已断开',
  }
  return labels[id] ? t(labels[id]) : t('其他事件')
}
</script>

<template>
  <el-card v-if="section === 'access'" class="wt-card" shadow="never">
    <div class="wt-inline-row">
      <span v-if="!status || error">{{ t('连接状态未确认') }}</span>
      <span v-else-if="status.listening">{{ t('本机正在监听端口 {0}', [status.port]) }}</span>
      <span v-else>{{ t('未检测到远程桌面监听，请先检查服务。') }}</span>
      <el-button :loading="loading" :disabled="working" @click="refresh">{{
        t('检查连接状态')
      }}</el-button>
    </div>
    <el-alert
      v-if="error || actionError"
      class="wt-actions--mt"
      type="error"
      :closable="false"
      :title="t('操作未完成，请展开诊断详情查看原因。')"
    />
    <p>{{ t('允许连接的 IP（白名单）') }}</p>
    <el-input
      v-model="sources"
      type="textarea"
      :rows="3"
      :disabled="busy || working || status?.pending"
      :aria-label="t('允许连接的 IP（白名单）')"
      :placeholder="t('每行一个公网 IP 或网段，例如 203.0.113.8')"
    />
    <p class="wt-hint">
      {{ t('只允许这些地址连接。使用 VPN 或动态 IP 时，请先确认实际出口地址。') }}
    </p>
    <div class="wt-actions wt-actions--mt">
      <el-button
        type="primary"
        :loading="working"
        :disabled="
          busy ||
          loading ||
          !status ||
          !!error ||
          status.pending ||
          !status.session?.known ||
          !sources.trim()
        "
        @click="apply"
        >{{ t('保存访问限制') }}</el-button
      >
    </div>
    <p class="wt-hint">
      {{ t('保存后五分钟内重新连接并确认；未确认会自动恢复。请保留 VPS 控制台入口。') }}
    </p>
    <el-collapse class="wt-actions--mt">
      <el-collapse-item :title="t('诊断详情')" name="diagnostics">
        <p v-if="error" role="alert">{{ error }}</p>
        <p v-if="actionError" role="alert">{{ actionError }}</p>
        <p v-if="status">
          {{ t('当前会话') }}:
          {{
            status.session?.known ? (status.session.remote ? 'RDP' : t('本地控制台')) : t('未知')
          }}
          · ID {{ status.session?.id }} · TermService: {{ status.service }}
        </p>
        <p>{{ status?.profiles }}</p>
        <p class="wt-hint">{{ t('此处只检查本机。云安全组、网关和外网连接需要另行验证。') }}</p>
        <p v-for="warning in status?.warnings || []" :key="warning" class="wt-hint">
          {{ warning }}
        </p>
        <el-table :data="status?.rules || []" max-height="250" style="margin-top: 12px">
          <el-table-column prop="name" :label="t('名称')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="action" :label="t('操作')" width="85" />
          <el-table-column prop="protocol" :label="t('协议')" width="85" />
          <el-table-column prop="ports" :label="t('端口')" width="90" />
          <el-table-column
            prop="addresses"
            :label="t('来源地址')"
            min-width="150"
            show-overflow-tooltip
          />
          <el-table-column prop="source" :label="t('策略来源')" width="110" />
          <el-table-column prop="profile" :label="t('配置文件')" width="120" />
        </el-table>
      </el-collapse-item>
    </el-collapse>
  </el-card>
  <el-card v-else class="wt-card" shadow="never">
    <div class="wt-inline-row">
      <span>{{ t('最近七天的服务器登录和断连记录。') }}</span>
      <el-button :loading="eventsLoading" @click="readEvents">{{ t('读取记录') }}</el-button>
    </div>
    <el-alert
      v-if="eventError"
      class="wt-actions--mt"
      type="error"
      :closable="false"
      :title="
        eventsRead
          ? t('读取失败，下面保留上次结果；原因见日志说明。')
          : t('读取失败，请查看日志说明。')
      "
    />
    <el-table
      v-if="eventsRead"
      :data="events"
      max-height="450"
      style="margin-top: 12px"
      :empty-text="t('暂无记录；日志未启用或已清除时也可能为空。')"
    >
      <el-table-column type="expand">
        <template #default="{ row }">
          <p>ID {{ row.id }} · {{ t('会话') }} {{ row.session || '—' }} · {{ row.channel }}</p>
          <p>{{ row.detail }}</p>
        </template>
      </el-table-column>
      <el-table-column prop="time" :label="t('时间（UTC）')" min-width="185" />
      <el-table-column :label="t('事件')" min-width="130">
        <template #default="{ row }">{{ eventLabel(row.id) }}</template>
      </el-table-column>
      <el-table-column prop="user" :label="t('用户名')" min-width="110" show-overflow-tooltip />
      <el-table-column
        prop="address"
        :label="t('来源地址')"
        min-width="130"
        show-overflow-tooltip
      />
    </el-table>
    <el-collapse v-if="eventError || eventWarnings.length" class="wt-actions--mt">
      <el-collapse-item :title="t('日志说明')" name="event-details">
        <p v-if="eventError" role="alert">{{ eventError }}</p>
        <p v-for="warning in eventWarnings" :key="warning">{{ warning }}</p>
      </el-collapse-item>
    </el-collapse>
  </el-card>
</template>
