<script setup>
import { computed, ref } from 'vue'
import { t } from '../i18n/index.js'
import { api } from '../api/index.js'

const emit = defineEmits(['checked'])
const target = ref('')
const port = ref('')
const tcpCount = ref(4)
const mtrRounds = ref(4)
const source = ref('')
const sources = ref(null)
const loadingSources = ref(false)
const sourceError = ref(false)
const stopping = ref(false)
const egress = ref(null)
const busy = ref('')
const error = ref('')
const errorMode = ref('')
const basic = ref(null)
const basicTarget = ref('')
const trace = ref(null)
const listener = ref(null)
const advanced = ref([])

const dnsText = {
  resolved: '已找到目标地址。',
  literal: '已输入 IP 地址，跳过域名解析。',
  'not-found': '没有找到地址，请检查域名拼写。',
  timeout: '域名解析超时，请稍后重试。',
  unavailable: '暂时无法解析域名，请检查本机网络或 DNS 设置。',
}
const tcpText = {
  connected: '端口可连通。',
  refused: '对方拒绝连接，请确认服务已启动且端口正确。',
  timeout: '连接超时，请检查服务、防火墙或云安全组。',
  unreachable: '当前电脑无法到达此地址。若目标是 IPv6，请确认本机有 IPv6 网络连接。',
  unavailable: '无法建立连接，请检查本机网络和目标地址。',
}
const listenerText = {
  listening: '有程序正在本机监听该端口。可在另一台电脑测试外部连通性。',
  'loopback-only': '该端口只监听本机回环地址，其他电脑无法直接连接。',
  'not-listening': '没有发现程序在本机监听该 TCP 端口。',
  unavailable: '暂时无法读取本机监听状态。',
}
const egressText = {
  'skipped-family': '所选源 IP 属于另一协议，已跳过。',
  timeout: '公网出口查询超时。',
  unreachable: '当前线路无法到达查询端点。',
  unavailable: '暂未取得公网地址。',
}
const noPing = computed(() =>
  basic.value?.tcp === 'connected'
    ? t('未收到 Ping 回应，但 TCP 端口可连通。目标可能关闭了 Ping 回应。')
    : t('没有收到 Ping 回应。可以用 TCP 端口测试确认具体服务能否连接。'),
)

async function run(mode) {
  if (busy.value) return
  error.value = ''
  errorMode.value = mode
  const host = target.value.trim()
  if ((mode === 'basic' || mode === 'trace') && !host) {
    error.value = '请输入有效域名或 IP 地址。'
    return
  }
  const portText = port.value.trim()
  const portNumber = Number(portText)
  if (
    (mode === 'listener' || (mode === 'basic' && portText)) &&
    (!/^\d{1,5}$/.test(portText) || portNumber < 1 || portNumber > 65535)
  ) {
    error.value = '请输入 1 到 65535 之间的端口号。'
    return
  }
  busy.value = mode
  try {
    if (mode === 'basic') {
      basic.value = null
      basic.value = await api.checkNetwork(host, portNumber, tcpCount.value, source.value)
      basicTarget.value = host
      emit('checked', {
        dns: basic.value.dns,
        ping: basic.value.ping,
        tcp: basic.value.tcp,
      })
    } else if (mode === 'listener') {
      listener.value = null
      listener.value = { port: portNumber, status: await api.checkLocalListener(portNumber) }
    } else if (mode === 'egress') {
      egress.value = null
      egress.value = await api.checkPublicEgress(source.value)
    } else {
      trace.value = null
      trace.value = await api.traceNetwork(host, mtrRounds.value, source.value)
    }
  } catch (reason) {
    const message = typeof reason === 'string' ? reason : reason?.message
    const expected = [
      '请输入有效域名或 IP 地址。',
      '请输入 1 到 65535 之间的端口号。',
      '已有网络检查正在运行。',
      '请选择本机已配置的源 IP。',
      '目标没有与所选源 IP 匹配的地址。',
      'TCPing 次数须为 1 到 10。',
      'MTR 轮数须为 1 到 10。',
    ]
    error.value = expected.includes(message) ? message : '检查未能完成，请稍后重试。'
  } finally {
    busy.value = ''
    stopping.value = false
  }
}

async function loadSources() {
  if (loadingSources.value) return
  loadingSources.value = true
  sourceError.value = false
  try {
    sources.value = await api.getNetworkSources()
  } catch {
    sourceError.value = true
  } finally {
    loadingSources.value = false
  }
}

function onAdvancedChange(open) {
  if (open.includes('mtr') && sources.value === null) loadSources()
}

async function stopTrace() {
  if (busy.value !== 'trace' || stopping.value) return
  stopping.value = true
  try {
    if (!(await api.stopNetworkTrace())) stopping.value = false
  } catch {
    stopping.value = false
  }
}
</script>

<template>
  <div data-view="network">
    <p class="page-desc">{{ t('检查从这台电脑到目标的连接。输入域名或 IP 地址即可开始。') }}</p>
    <el-card shadow="never" class="wt-card">
      <form class="network-form" @submit.prevent="run('basic')">
        <label>
          <span>{{ t('目标地址') }}</span>
          <el-input
            v-model="target"
            :aria-label="t('目标地址')"
            placeholder="example.com"
            maxlength="255"
            :disabled="!!busy"
          />
        </label>
        <label>
          <span>{{ t('TCP 端口（选填）') }}</span>
          <el-input
            v-model="port"
            :aria-label="t('TCP 端口（选填）')"
            placeholder="443"
            inputmode="numeric"
            :disabled="!!busy"
          />
        </label>
        <el-button
          native-type="submit"
          type="primary"
          :loading="busy === 'basic'"
          :disabled="!!busy"
          >{{ t('开始检查') }}</el-button
        >
      </form>
      <div v-if="port.trim()" class="network-options">
        <label>
          <span>{{ t('TCPing 次数') }}</span>
          <el-input-number
            v-model="tcpCount"
            :aria-label="t('TCPing 次数')"
            :min="1"
            :max="10"
            :step="1"
            :precision="0"
            :value-on-clear="4"
            :disabled="!!busy"
            controls-position="right"
          />
        </label>
      </div>
      <p class="wt-hint">
        {{ t('例如网站可填 443，远程桌面可填 3389。留空时只检查 DNS 和 Ping。') }}
      </p>
      <el-alert
        v-if="error && errorMode === 'basic'"
        :title="t(error)"
        type="warning"
        :closable="false"
        show-icon
      />
      <p v-if="basic" class="network-address">
        <strong>{{ t('检查结果：{0}', [basicTarget]) }}</strong>
      </p>
      <dl v-if="basic" class="network-results" aria-live="polite">
        <div>
          <dt>{{ t('DNS 解析') }}</dt>
          <dd>
            {{ t(dnsText[basic.dns]) }}
            <span v-if="basic.addresses?.length" class="network-address">{{
              basic.addresses.join(' · ')
            }}</span>
          </dd>
        </div>
        <div>
          <dt>Ping</dt>
          <dd>
            <template v-if="basic.ping === 'reply'">{{
              t('收到 {0}/{1} 次回应，平均 {2} ms，未回应 {3}%。', [
                basic.received,
                basic.sent,
                basic.averageMs,
                Math.round((1 - basic.received / basic.sent) * 100),
              ])
            }}</template>
            <template v-else-if="basic.ping === 'no-reply'">{{ noPing }}</template>
            <template v-else-if="basic.ping === 'unavailable'">{{
              t('本机暂时无法执行 Ping 检查。')
            }}</template>
            <template v-else>{{ t('未测试：没有可用的目标地址。') }}</template>
          </dd>
        </div>
        <div>
          <dt>{{ basic.port ? 'TCPing ' + basic.port : 'TCPing' }}</dt>
          <dd>
            {{
              basic.tcp !== 'skipped'
                ? t(tcpText[basic.tcp])
                : t(basic.port ? '未测试：没有可用的目标地址。' : '未填写端口，已跳过。')
            }}
            <div v-if="basic.tcpStats?.attempts">
              {{
                t('连接成功 {0}/{1} 次，成功率 {2}%。', [
                  basic.tcpStats.successes,
                  basic.tcpStats.attempts,
                  Math.round((100 * basic.tcpStats.successes) / basic.tcpStats.attempts),
                ])
              }}
            </div>
            <div v-if="basic.tcpStats?.successes">
              {{
                t('成功连接耗时（最短 / 平均 / 最长）：{0} / {1} / {2} ms', [
                  basic.tcpStats.minMs.toFixed(1),
                  basic.tcpStats.averageMs.toFixed(1),
                  basic.tcpStats.maxMs.toFixed(1),
                ])
              }}
            </div>
          </dd>
        </div>
      </dl>
      <p v-if="basic?.source" class="wt-hint network-address">
        {{ t('指定源 IP：{0}', [basic.source]) }}
      </p>
      <p v-if="basic?.address" class="wt-hint network-address">
        {{ t('Ping 和 TCP 使用的目标 IP：{0}', [basic.address]) }}
      </p>
      <p class="wt-hint">
        {{
          t('检查结果以当前电脑为起点。要确认外网能否访问 VPS，请在另一台电脑测试 VPS 的公网地址。')
        }}
      </p>
    </el-card>
    <el-card shadow="never" class="wt-card" :header="t('公网出口 IP')">
      <p class="wt-hint network-action-hint">
        {{
          t(
            '点击后向 ipify 发送 HTTPS 请求，分别查看对端看到的 IPv4 和 IPv6 地址。可在高级诊断指定本机源 IP。',
          )
        }}
      </p>
      <el-button :loading="busy === 'egress'" :disabled="!!busy" @click="run('egress')">{{
        t('查询公网出口')
      }}</el-button>
      <table v-if="egress" class="egress-table" aria-live="polite">
        <thead>
          <tr>
            <th>{{ t('协议') }}</th>
            <th>{{ t('本机源 IP') }}</th>
            <th>{{ t('实际公网出口 IP') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in egress" :key="row.family">
            <th scope="row">IPv{{ row.family }}</th>
            <td>{{ row.source || '—' }}</td>
            <td>
              <div v-if="row.publicIp">{{ row.publicIp }}</div>
              <div v-else>{{ t(egressText[row.status] || '暂未取得公网地址。') }}</div>
              <small>{{ row.endpoint }}</small>
            </td>
          </tr>
        </tbody>
      </table>
      <el-alert
        v-if="error && errorMode === 'egress'"
        :title="t(error)"
        type="warning"
        :closable="false"
        show-icon
      />
    </el-card>
    <el-card shadow="never" class="wt-card">
      <el-collapse v-model="advanced" @change="onAdvancedChange">
        <el-collapse-item :title="t('查看本机监听')" name="listener">
          <p class="wt-hint network-action-hint">
            {{ t('在上方填写 TCP 端口，然后检查这台电脑有没有程序在等待连接。') }}
          </p>
          <el-button :loading="busy === 'listener'" :disabled="!!busy" @click="run('listener')">{{
            t('检查本机端口')
          }}</el-button>
          <p v-if="listener" role="status">
            TCP {{ listener.port }} · {{ t(listenerText[listener.status]) }}
          </p>
        </el-collapse-item>
        <el-collapse-item :title="t('高级诊断')" name="mtr">
          <p class="wt-hint network-action-hint">
            {{
              t(
                'MTR 每轮最多检查 24 跳，最多约 2 分钟。中间节点未回应时，请结合后续节点和目标的结果判断。',
              )
            }}
          </p>
          <p class="wt-hint">
            {{ t('源 IP 用于 Ping、TCPing、MTR 和公网出口查询。自动时由 Windows 选择。') }}
          </p>
          <div class="network-options">
            <label class="source-option">
              <span>{{ t('本机源 IP') }}</span>
              <el-select
                v-model="source"
                :aria-label="t('本机源 IP')"
                :loading="loadingSources"
                :disabled="!!busy"
                class="source-select"
                :empty-values="[null, undefined]"
              >
                <el-option :label="t('自动选择')" value="" />
                <el-option
                  v-for="item in sources || []"
                  :key="item.address"
                  :label="item.address + ' · ' + item.interface"
                  :value="item.address"
                />
              </el-select>
            </label>
            <el-button :disabled="!!busy || loadingSources" @click="loadSources">{{
              t('刷新源 IP')
            }}</el-button>
            <label>
              <span>{{ t('MTR 轮数') }}</span>
              <el-input-number
                v-model="mtrRounds"
                :aria-label="t('MTR 轮数')"
                :min="1"
                :max="10"
                :step="1"
                :precision="0"
                :value-on-clear="4"
                :disabled="!!busy"
                controls-position="right"
              />
            </label>
          </div>
          <p v-if="sourceError" role="status">{{ t('无法读取本机源地址，请刷新重试。') }}</p>
          <div class="wt-actions">
            <el-button
              :loading="busy === 'trace'"
              :disabled="!!busy || !target.trim()"
              @click="run('trace')"
              >{{ t('运行 MTR') }}</el-button
            >
            <el-button v-if="busy === 'trace'" :loading="stopping" @click="stopTrace">{{
              t('停止 MTR')
            }}</el-button>
          </div>
          <template v-if="trace">
            <p v-if="trace.source" class="wt-hint network-address">
              {{ t('指定源 IP：{0}', [trace.source]) }}
            </p>
            <p v-if="trace.address || trace.status === 'stopped'" class="wt-hint">
              {{ t('已完成 {0}/{1} 轮。', [trace.roundsCompleted, trace.rounds]) }}
            </p>
            <p v-if="trace.address" class="network-address">
              {{ t('MTR 目标 IP：{0}', [trace.address]) }}
            </p>
            <p v-if="trace.status === 'complete'" role="status">
              {{ t(trace.reached ? '目标已回应。' : '本次追踪未收到目标回应。') }}
            </p>
            <p v-else-if="trace.status === 'stopped'" role="status">
              {{ t('已停止 MTR，保留已取得的结果。') }}
            </p>
            <p v-else role="status">
              {{
                t(
                  dnsText[trace.status] ||
                    (trace.hops?.length
                      ? 'MTR 提前结束，下方是已取得的结果。'
                      : '无法开始 MTR。请检查当前电脑能否访问该地址。'),
                )
              }}
            </p>
            <p
              v-if="trace.reached && trace.hops?.some((hop) => hop.received === 0)"
              class="wt-hint"
            >
              {{ t('中间节点可能限制探测回应，仍可正常转发数据。') }}
            </p>
            <el-table v-if="trace.hops?.length" :data="trace.hops" size="small">
              <el-table-column prop="number" :label="t('跳数')" width="60" />
              <el-table-column :label="t('回应地址')" min-width="130">
                <template #default="{ row }"
                  ><span class="network-address">{{
                    row.addresses?.join(' · ') || '—'
                  }}</span></template
                >
              </el-table-column>
              <el-table-column :label="t('回应 / 发送')" min-width="150">
                <template #default="{ row }">
                  {{ row.received }} / {{ row.sent }} ·
                  {{ t('{0}% 未回应', [Math.round((1 - row.received / row.sent) * 100)]) }}
                </template>
              </el-table-column>
              <el-table-column :label="t('平均延迟')" min-width="90">
                <template #default="{ row }">{{
                  row.received ? row.averageMs + ' ms' : '—'
                }}</template>
              </el-table-column>
            </el-table>
          </template>
        </el-collapse-item>
      </el-collapse>
      <el-alert
        v-if="error && (errorMode === 'listener' || errorMode === 'trace')"
        :title="t(error)"
        type="warning"
        :closable="false"
        show-icon
      />
    </el-card>
  </div>
</template>

<style scoped>
.network-form {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(7rem, 1fr) auto;
  align-items: end;
  gap: 12px;
}
.network-form label {
  min-width: 0;
  display: grid;
  gap: 6px;
}
.network-options {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 8px 12px;
  margin: 10px 0;
}
.network-options label {
  display: grid;
  gap: 4px;
  max-width: 100%;
}
.source-option {
  width: min(320px, 100%);
  min-width: 0;
}
.source-select {
  width: 100%;
}
.egress-table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  margin-top: 12px;
}
.egress-table :is(th, td) {
  text-align: left;
  vertical-align: top;
  padding: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  overflow-wrap: anywhere;
}
.egress-table th:first-child {
  width: 5.5rem;
}
.egress-table small {
  color: var(--el-text-color-secondary);
}
.network-results {
  margin: 16px 0 0;
}
.network-results > div {
  display: grid;
  grid-template-columns: 6rem minmax(0, 1fr);
  padding: 9px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.network-results dt {
  font-weight: 600;
}
.network-results dd {
  margin: 0;
}
.network-action-hint {
  margin-bottom: 8px;
}
.network-address {
  overflow-wrap: anywhere;
}
@media (max-width: 800px) {
  .network-form {
    grid-template-columns: minmax(0, 1fr);
  }
  .network-form > button {
    justify-self: start;
  }
}
</style>
