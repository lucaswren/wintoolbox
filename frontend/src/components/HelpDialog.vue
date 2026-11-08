<script setup>
import { computed, ref } from 'vue'
import { locale, t } from '../i18n/index.js'
import { api } from '../api/index.js'

const visible = defineModel({ type: Boolean, default: false })
const emit = defineEmits(['navigate'])
const opening = ref(false)
const openFailed = ref(false)

const items = computed(() => [
  {
    key: 'overview',
    title: t('系统信息'),
    detail: t('查看 Windows 版本、磁盘余量和开机时间。'),
  },
  {
    key: 'network',
    title: t('网络检查'),
    detail: t('输入域名或 IP，检查解析、延迟或端口连接。'),
  },
  {
    key: 'systemlogs',
    title: t('系统日志'),
    detail: t('查看最近的关机、重启和异常记录。'),
  },
])

function navigate(key) {
  visible.value = false
  emit('navigate', key)
}

async function openDocumentation() {
  if (opening.value) return
  opening.value = true
  openFailed.value = false
  try {
    await api.openDocumentation(locale.value)
  } catch {
    openFailed.value = true
  } finally {
    opening.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('新手指引')"
    width="min(540px, 92vw)"
    top="min(7vh, 42px)"
    append-to-body
    @open="openFailed = false"
  >
    <div class="help-content">
      <p>{{ t('第一次用 WinToolBox？先从下面的查看和检查功能开始。') }}</p>
      <ul class="help-list">
        <li v-for="item in items" :key="item.key">
          <el-button link type="primary" @click="navigate(item.key)">{{ item.title }}</el-button>
          <span>{{ item.detail }}</span>
        </li>
      </ul>
      <p>{{ t('修改系统设置前，请阅读确认提示。操作步骤和注意事项可在文档中查看。') }}</p>
      <p v-if="openFailed" role="status">
        {{ t('未能打开浏览器。请在浏览器中访问 WinToolBox 项目的文档目录。') }}
      </p>
    </div>
    <template #footer>
      <el-button @click="visible = false">{{ t('知道了') }}</el-button>
      <el-button type="primary" :loading="opening" @click="openDocumentation">
        {{ t('在浏览器打开文档') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.help-content {
  max-height: 55vh;
  overflow: auto;
  line-height: 1.55;
}
.help-content p {
  margin: 0 0 12px;
}
.help-content p:last-child {
  margin-bottom: 0;
}
.help-list {
  display: grid;
  gap: 10px;
  padding: 0;
  margin: 0 0 14px;
  list-style: none;
}
.help-list li {
  display: grid;
  justify-items: start;
  gap: 2px;
}
</style>
