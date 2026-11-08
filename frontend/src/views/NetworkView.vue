<script setup>
import { computed, ref } from 'vue'
import { t } from '../i18n/index.js'
import { api } from '../api/index.js'

const emit = defineEmits(['checked'])
const target = ref('')
const port = ref('')
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
  if (mode !== 'listener' && !host) {
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
      basic.value = await api.checkNetwork(host, portNumber)
      basicTarget.value = host
      emit('checked', {
        dns: basic.value.dns,
        ping: basic.value.ping,
        tcp: basic.value.tcp,
      })
    } else if (mode === 'listener') {
      listener.value = null
      listener.value = { port: portNumber, status: await api.checkLocalListener(portNumber) }
    } else {
      trace.value = null
      trace.value = await api.traceNetwork(host)
    }
  } catch (reason) {
    const message = typeof reason === 'string' ? reason : reason?.message
    const expected = [
      '请输入有效域名或 IP 地址。',
      '请输入 1 到 65535 之间的端口号。',
      '已有网络检查正在运行。',
    ]
    error.value = expected.includes(message) ? message : '检查未能完成，请稍后重试。'
  } finally {
    busy.value = ''
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
          <dt>{{ basic.port ? 'TCP ' + basic.port : 'TCP' }}</dt>
          <dd>
            {{
              basic.tcp !== 'skipped'
                ? t(tcpText[basic.tcp])
                : t(basic.port ? '未测试：没有可用的目标地址。' : '未填写端口，已跳过。')
            }}
          </dd>
        </div>
      </dl>
      <p v-if="basic?.address" class="wt-hint network-address">
        {{ t('Ping 和 TCP 使用的目标 IP：{0}', [basic.address]) }}
      </p>
      <p class="wt-hint">
        {{
          t('检查结果以当前电脑为起点。要确认外网能否访问 VPS，请在另一台电脑测试 VPS 的公网地址。')
        }}
      </p>
    </el-card>
    <el-card shadow="never" class="wt-card">
      <el-collapse v-model="advanced">
        <el-collapse-item :title="t('查看本机监听')" name="listener">
          <p class="wt-hint">
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
          <p class="wt-hint">
            {{
              t(
                'MTR 会多次检查沿途节点，最多 24 跳，最久约 50 秒。中间节点不回应时，请结合后续节点和目标的结果判断。',
              )
            }}
          </p>
          <el-button
            :loading="busy === 'trace'"
            :disabled="!!busy || !target.trim()"
            @click="run('trace')"
            >{{ t('运行 MTR') }}</el-button
          >
          <template v-if="trace">
            <p v-if="trace.address" class="network-address">
              {{ t('MTR 目标 IP：{0}', [trace.address]) }}
            </p>
            <p v-if="trace.status === 'complete'" role="status">
              {{ t(trace.reached ? '目标已回应。' : '本次追踪未收到目标回应。') }}
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
        v-if="error && errorMode !== 'basic'"
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
