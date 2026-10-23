<script setup>
import { t } from '../i18n/index.js'
import { ref } from 'vue'

import { FORM_LABEL_WIDTH } from '../constants.js'
import RdpSafetyPanel from '../components/RdpSafetyPanel.vue'
import RdpClientHistory from '../components/RdpClientHistory.vue'
import { useRdpSafety } from '../composables/useRdpSafety.js'

const { status: safety } = useRdpSafety()
const activeTab = ref('connection')

defineProps({
  status: { type: Object, default: null },
  form: { type: Object, required: true },
  busy: { type: Boolean, default: false },
  history: { type: Array, default: () => [] },
  historyLoading: { type: Boolean, default: false },
})

defineEmits([
  'save-port',
  'toggle',
  'set-nla',
  'clear-history',
  'clear-history-kind',
  'delete-history',
  'refresh-history',
])
</script>

<template>
  <div data-view="rdp">
    <el-alert
      v-if="status && status.rdpAvailable === false"
      type="warning"
      :closable="false"
      show-icon
      :title="t('无法读取远程桌面状态（可能未安装远程桌面服务或权限不足）')"
      class="wt-card"
    />

    <el-tabs
      v-model="activeTab"
      @tab-change="
        (name) => {
          if (name === 'client') $emit('refresh-history')
        }
      "
    >
      <el-tab-pane :label="t('连接设置')" name="connection">
        <el-card shadow="never" class="wt-card">
          <div class="wt-status-block">
            <div class="wt-status-line">
              <span class="wt-status-label">{{ t('远程桌面') }}</span>
              <template v-if="status?.rdpAvailable !== false">
                <el-tag :type="status?.rdpEnabled ? 'success' : 'info'" effect="dark">
                  {{ status?.rdpEnabled ? t('已开启') : t('已关闭') }}
                </el-tag>
                <span class="wt-status-sub">{{ t('配置端口') }} {{ status?.rdpPort || '—' }}</span>
              </template>
              <el-tag v-else type="danger" effect="dark">{{ t('不可用') }}</el-tag>
            </div>
            <div
              v-if="status?.rdpAvailable !== false"
              class="wt-status-line"
              style="margin-top: 8px"
            >
              <span class="wt-status-label">{{ t('网络级身份验证') }}</span>
              <el-tag
                :type="status?.rdpNLAUnknown ? 'info' : status?.rdpNLA ? 'success' : 'warning'"
                effect="dark"
              >
                {{
                  status?.rdpNLAUnknown
                    ? t('状态未知')
                    : status?.rdpNLA
                      ? t('已强制 NLA')
                      : t('未强制 NLA')
                }}
              </el-tag>
            </div>
          </div>

          <el-form :label-width="FORM_LABEL_WIDTH" class="wt-form-row">
            <el-form-item :label="t('端口')">
              <el-input-number
                v-model="form.rdpPort"
                :min="1024"
                :max="65535"
                controls-position="right"
                :disabled="busy || status?.rdpAvailable === false || safety?.pending"
              />
            </el-form-item>
            <el-form-item>
              <div class="wt-actions">
                <el-button
                  type="primary"
                  :disabled="
                    busy ||
                    status?.rdpAvailable === false ||
                    safety?.pending ||
                    !safety?.session?.known
                  "
                  @click="$emit('save-port')"
                  >{{ t('保存端口') }}</el-button
                >
                <el-button
                  type="warning"
                  :disabled="
                    busy ||
                    status?.rdpAvailable === false ||
                    safety?.pending ||
                    !safety?.session?.known ||
                    safety?.session?.remote
                  "
                  @click="$emit('toggle')"
                >
                  {{ status?.rdpEnabled ? t('关闭') : t('开启') }}
                </el-button>
                <el-button
                  type="success"
                  :disabled="
                    busy ||
                    safety?.pending ||
                    !safety?.session?.known ||
                    status?.rdpAvailable === false ||
                    status?.rdpNLAUnknown ||
                    status?.rdpNLA
                  "
                  @click="$emit('set-nla', true)"
                  >{{ t('强制 NLA') }}</el-button
                >
                <el-button
                  :disabled="
                    busy ||
                    safety?.pending ||
                    !safety?.session?.known ||
                    status?.rdpAvailable === false ||
                    status?.rdpNLAUnknown ||
                    !status?.rdpNLA
                  "
                  @click="$emit('set-nla', false)"
                  >{{ t('关闭 NLA') }}</el-button
                >
              </div>
            </el-form-item>
          </el-form>
          <p class="wt-hint">
            {{
              t(
                '强制 NLA 可在完成身份验证前拒绝连接，降低未认证爆破面。极旧客户端可能无法连接。NLA 状态未知时请先刷新。',
              )
            }}
          </p>
        </el-card>
      </el-tab-pane>
      <el-tab-pane :label="t('访问限制')" name="access">
        <RdpSafetyPanel :busy="busy" section="access" />
      </el-tab-pane>
      <el-tab-pane :label="t('登录记录')" name="events">
        <RdpSafetyPanel :busy="busy" section="events" />
      </el-tab-pane>
      <el-tab-pane :label="t('客户端记录')" name="client" lazy>
        <RdpClientHistory
          :history="history"
          :history-loading="historyLoading"
          :busy="busy"
          @clear-history="$emit('clear-history')"
          @clear-history-kind="(kind) => $emit('clear-history-kind', kind)"
          @delete-history="(row) => $emit('delete-history', row)"
          @refresh-history="$emit('refresh-history')"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>
