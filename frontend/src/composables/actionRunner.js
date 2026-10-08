export function createRunner({ busy, appendLog, refresh, getModule }) {
  return async function run(
    fn,
    successMsg,
    { confirm, confirmType = 'warning', skipRefresh } = {},
  ) {
    if (busy.value) {
      appendLog('上一操作仍在进行，请稍候…')
      ElMessage.info('上一操作仍在进行，请稍候')
      return false
    }
    // Hold busy across confirm + action to prevent stacked dangerous ops.
    const module = getModule?.()
    busy.value = true
    let attempted = false
    let succeeded = false
    try {
      if (confirm) {
        try {
          await ElMessageBox.confirm(confirm, '请确认', {
            type: confirmType,
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            closeOnClickModal: false,
            closeOnPressEscape: true,
            distinguishCancelAndClose: true,
          })
        } catch {
          return false
        }
      }
      attempted = true
      const result = await fn()
      succeeded = true
      if (successMsg) {
        appendLog(successMsg, module)
        ElMessage.success(successMsg)
      }
      return result == null ? true : result
    } catch (e) {
      const msg = e?.message || String(e)
      appendLog('失败: ' + msg, module)
      ElMessage.error(msg)
      return false
    } finally {
      if (attempted && !skipRefresh) {
        try {
          await refresh(false, module)
        } catch {
          ElMessage.warning(
            succeeded ? '操作成功，但状态复查失败' : '操作可能已部分生效，状态复查失败，请手动刷新',
          )
        }
      }
      busy.value = false
    }
  }
}

export function powerDelayLabel(seconds) {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return '60 秒后'
  if (n === 0) return '立即'
  return `${n} 秒后`
}

export function normalizePort(value, fallback = 0) {
  const n = Number(value)
  if (!Number.isFinite(n)) return fallback
  const p = Math.floor(n)
  if (p < 1 || p > 65535) return fallback
  return p
}
