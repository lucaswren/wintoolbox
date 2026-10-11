<script setup>
import { t } from '../i18n/index.js'

import { computed } from 'vue'

const props = defineProps({
  status: { type: Object, default: null },
  rules: { type: Array, default: () => [] },
  form: { type: Object, required: true },
  busy: { type: Boolean, default: false },
})

defineEmits([
  'allow',
  'remove',
  'remove-row',
  'enable-all',
  'disable-all',
  'disable-ping',
  'enable-ping',
  'clear-allow-all',
  'block-risk',
  'unblock-risk',
])

const profiles = computed(() => [
  { key: 'domain', label: t('域'), value: props.status?.firewallDomain || t('未知') },
  { key: 'private', label: t('专用'), value: props.status?.firewallPrivate || t('未知') },
  { key: 'public', label: t('公用'), value: props.status?.firewallPublic || t('未知') },
])

function profileTagType(value) {
  if (value === '开') return 'success'
  if (value === '关') return 'danger'
  return 'info'
}

function profileLabel(value) {
  if (value === '开') return t('已开启')
  if (value === '关') return t('已关闭')
  return t(value || '未知')
}

const overall = computed(() => {
  if (props.status?.firewallAllOn) {
    return { text: t('全部开启'), type: 'success', desc: t('域 / 专用 / 公用配置文件均已启用') }
  }
  if (props.status?.firewallAllOff) {
    return { text: t('全部关闭'), type: 'danger', desc: t('域 / 专用 / 公用配置文件均已关闭') }
  }
  const vals = profiles.value.map((p) => p.value)
  if (vals.every((v) => v !== '开' && v !== '关')) {
    return { text: t('状态未知'), type: 'info', desc: t('未能可靠读取防火墙配置文件状态') }
  }
  return { text: t('部分开启'), type: 'warning', desc: t('部分配置文件已开启，部分已关闭或未知') }
})

const pingState = computed(() => props.status?.pingState || 'enabled')
const pingStateInfo = computed(() => {
  switch (pingState.value) {
    case 'blocked':
      return { text: t('已禁用 ping'), type: 'info', desc: t('IPv4 和 IPv6 均已阻止 Echo 请求') }
    case 'partial':
      return { text: t('部分禁用'), type: 'warning', desc: t('IPv4 / IPv6 中仅部分协议被阻止') }
    case 'unknown':
      return { text: t('状态未知'), type: 'info', desc: t('禁 ping 状态读取超时或失败') }
    default:
      return {
        text: t('未检测到工具拦截'),
        type: 'info',
        desc: t('未检测到本工具的 Echo 拦截规则；实际响应取决于其他规则和网络配置'),
      }
  }
})
const canDisablePing = computed(
  () => !props.busy && pingState.value !== 'blocked' && pingState.value !== 'unknown',
)
const canEnablePing = computed(
  () => !props.busy && pingState.value !== 'enabled' && pingState.value !== 'unknown',
)

const canEnableAll = computed(
  () =>
    !props.busy &&
    !props.status?.firewallAllOn &&
    profiles.value.some((p) => ['开', '关'].includes(p.value)),
)
const canDisableAll = computed(
  () =>
    !props.busy &&
    !props.status?.firewallAllOff &&
    profiles.value.some((p) => ['开', '关'].includes(p.value)),
)

const riskInfo = computed(() => {
  if (props.status?.riskPortsUnknown) {
    return { text: t('状态未知'), type: 'info' }
  }
  if (props.status?.riskPortsBlocked) {
    return { text: t('已配置全部拦截规则'), type: 'success' }
  }
  if (props.status?.riskPortsPartial) {
    return { text: t('部分拦截'), type: 'warning' }
  }
  return { text: t('未启用'), type: 'info' }
})
const riskRows = computed(() =>
  [135, 139, 445, 5985, 5986].map((port) => ({
    port,
    blocked: props.status?.riskPorts?.find((row) => row.port === port)?.blocked,
    unknown:
      props.status?.riskPortsUnknown || !props.status?.riskPorts?.some((row) => row.port === port),
  })),
)
const canBlockRisk = computed(
  () => !props.busy && !props.status?.riskPortsUnknown && !props.status?.riskPortsBlocked,
)
const canUnblockRisk = computed(
  () =>
    !props.busy &&
    !props.status?.riskPortsUnknown &&
    (props.status?.riskPortsBlocked || props.status?.riskPortsPartial),
)
</script>

<template>
  <div data-view="firewall">
    <p class="page-desc">
      {{
        t(
          '一键开关配置文件、禁用或恢复 ping、高危端口预设，并为入站 TCP 添加放行规则（规则名前缀 WinToolbox）',
        )
      }}
    </p>

    <el-card shadow="never" :header="t('配置文件')" class="wt-card">
      <div class="wt-status-block">
        <div class="wt-status-line">
          <span class="wt-status-label">{{ t('总状态') }}</span>
          <el-tag :type="overall.type" effect="dark">{{ overall.text }}</el-tag>
        </div>
        <p class="wt-detail">{{ overall.desc }}</p>
      </div>

      <div class="wt-stat-grid">
        <div v-for="p in profiles" :key="p.key" class="wt-stat-item">
          <span class="wt-stat-name">{{ p.label }}</span>
          <el-tag :type="profileTagType(p.value)" effect="plain" size="small">
            {{ profileLabel(p.value) }}
          </el-tag>
        </div>
      </div>

      <div class="wt-actions wt-actions--mt">
        <el-button type="success" :disabled="!canEnableAll" @click="$emit('enable-all')">{{
          t('全部开启')
        }}</el-button>
        <el-button type="danger" :disabled="!canDisableAll" @click="$emit('disable-all')">{{
          t('全部关闭')
        }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" :header="t('端口操作')" class="wt-card">
      <el-form inline class="wt-form-row">
        <el-form-item :label="t('TCP 端口')">
          <el-input-number v-model="form.fwPort" :min="1" :max="65535" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <div class="wt-actions">
            <el-button type="success" :disabled="busy" @click="$emit('allow')">{{
              t('放行')
            }}</el-button>
            <el-button type="danger" :disabled="busy" @click="$emit('remove')">{{
              t('按端口删除')
            }}</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" :header="t('Ping 操作')" class="wt-card">
      <p class="wt-detail">
        {{ t('当前状态：')
        }}<el-tag :type="pingStateInfo.type" effect="dark" size="small">
          {{ pingStateInfo.text }}
        </el-tag>
        <br />
        {{ pingStateInfo.desc }}
        <br />
        IPv4：{{
          pingState === 'unknown' ? t('未知') : status?.pingIPv4Blocked ? t('已阻止') : t('未阻止')
        }}；IPv6：{{
          pingState === 'unknown' ? t('未知') : status?.pingIPv6Blocked ? t('已阻止') : t('未阻止')
        }}
      </p>
      <div class="wt-actions wt-actions--mt">
        <el-button type="warning" :disabled="!canDisablePing" @click="$emit('disable-ping')">{{
          t('一键禁 ping')
        }}</el-button>
        <el-button type="success" :disabled="!canEnablePing" @click="$emit('enable-ping')">{{
          t('恢复 ping')
        }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" :header="t('高危端口预设')" class="wt-card">
      <div class="wt-status-block">
        <div class="wt-status-line">
          <span class="wt-status-label">{{ t('规则配置状态') }}</span>
          <el-tag :type="riskInfo.type" effect="dark">{{ riskInfo.text }}</el-tag>
          <span class="wt-status-sub">{{ t(status?.riskPortsDetail || '—') }}</span>
        </div>
      </div>
      <div class="wt-actions" style="margin-bottom: 10px">
        <el-tag
          v-for="row in riskRows"
          :key="row.port"
          :type="row.unknown ? 'info' : row.blocked ? 'success' : 'warning'"
          >TCP {{ row.port }} ·
          {{ row.unknown ? t('未知') : row.blocked ? t('已配置') : t('未配置') }}</el-tag
        >
      </div>
      <p class="wt-hint">
        {{
          t('拦截规则仅代表本地配置；防火墙配置文件关闭、域策略或其他网络设备可能影响实际拦截。')
        }}
      </p>
      <p class="wt-detail">
        {{
          t(
            '一键拦切入站 TCP：135（RPC）、139（NetBIOS）、445（SMB）、5985/5986（WinRM）。不含 3389，以免影响远程桌面。',
          )
        }}<template v-if="status?.riskPortsUnknown">
          <br />{{ t('状态未知时请先点击右上角「刷新」后再操作。') }}</template
        >
      </p>
      <div class="wt-actions wt-actions--mt">
        <el-button type="warning" :disabled="!canBlockRisk" @click="$emit('block-risk')">{{
          t('拦高危入站端口')
        }}</el-button>
        <el-button type="success" :disabled="!canUnblockRisk" @click="$emit('unblock-risk')">{{
          t('移除高危拦截')
        }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" :header="t('已创建规则')" class="wt-card">
      <div class="wt-actions wt-actions--mt" style="margin-bottom: 10px">
        <el-button
          type="danger"
          :disabled="busy || !rules.length"
          @click="$emit('clear-allow-all')"
          >{{ t('删除全部放行规则') }}</el-button
        >
      </div>
      <el-table
        :data="rules"
        stripe
        :empty-text="t('暂无规则')"
        highlight-current-row
        style="width: 100%"
      >
        <el-table-column prop="name" :label="t('规则名称')" min-width="240" />
        <el-table-column prop="port" :label="t('端口')" width="90" />
        <el-table-column :label="t('状态')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? t('开启') : t('关闭') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('操作')" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" :disabled="busy" @click="$emit('remove-row', row.port)">{{
              t('删除')
            }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
