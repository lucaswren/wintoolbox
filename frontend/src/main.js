import { createApp } from 'vue'
import App from './App.vue'
import { api } from './api/index.js'
import { useSystemLocale } from './i18n/index.js'
import './styles.css'
// Programmatic ElMessage / ElMessageBox need explicit styles (not covered by SFC auto-import).
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'

async function start() {
  try {
    useSystemLocale((await api.getAppInfo())?.language)
  } catch {
    // Browser development uses navigator.languages when the native bridge is absent.
  }
  createApp(App).mount('#app')
}
start()
