import { t } from '../i18n/index.js'
export function createRunner({ busy, appendLog, refresh, getModule, operation }) {
  return async function run(
    fn,
    successMsg,
    { confirm, confirmType = 'warning', skipRefresh, title } = {},
  ) {
    if (busy.value) {
      ElMessage.info(t('上一操作仍在进行，请稍候'))
      return false
    }
    const module = getModule?.()
    busy.value = true
    let attempted = false
    let result
    let error = ''
    let verificationError = ''
    let timer
    let started = Date.now()
    const update = (phase, detail = '') => {
      if (operation)
        Object.assign(operation, {
          phase,
          module,
          title: title || confirm || successMsg || t('系统操作'),
          detail,
          elapsed: Math.floor((Date.now() - started) / 1000),
        })
    }
    try {
      if (confirm) {
        try {
          await ElMessageBox.confirm(confirm, t('请确认'), {
            type: confirmType,
            confirmButtonText: t('确定'),
            cancelButtonText: t('取消'),
            closeOnClickModal: false,
            closeOnPressEscape: true,
            distinguishCancelAndClose: true,
          })
        } catch {
          return false
        }
      }
      started = Date.now()
      attempted = true
      update('running')
      appendLog(t('开始执行：') + (title || confirm || successMsg || t('系统操作')), module)
      timer = setInterval(() => {
        if (operation) operation.elapsed++
      }, 1000)
      try {
        result = await fn()
      } catch (e) {
        error = t(e?.message || String(e))
      }
      if (!skipRefresh) {
        update('verifying')
        try {
          const snapshot = await refresh(false, module)
          if (snapshot && stateIsUnknown(snapshot, module)) verificationError = t('部分状态未知')
          if (snapshot?.warnings?.length)
            verificationError = snapshot.warnings.map((value) => t(value)).join('; ')
        } catch (e) {
          verificationError = t(e?.message || String(e))
        }
      }
      if (error) {
        const detail =
          t('操作可能已部分生效，请查看当前状态。') +
          ' ' +
          error +
          (verificationError ? ' ' + t('状态复查失败：') + verificationError : '')
        update('failed', detail)
        appendLog(t('失败: ') + detail, module)
        return false
      }
      if (verificationError) {
        const detail = t('操作已返回，但状态复查不完整，请手动刷新。') + ' ' + verificationError
        update('warning', detail)
        appendLog(detail, module)
      } else {
        update('completed', successMsg || t('操作完成'))
        appendLog(successMsg || t('操作完成'), module)
      }
      return result == null ? true : result
    } finally {
      clearInterval(timer)
      busy.value = false
      if (attempted) {
        const elapsed = Math.floor((Date.now() - started) / 1000)
        if (operation) operation.elapsed = elapsed
        appendLog(t('耗时 {0} 秒', [elapsed]), module)
      }
    }
  }
}

export function powerDelayLabel(seconds) {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return t('60 秒后')
  if (n === 0) return t('立即')
  return t('{0} 秒后', [n])
}

export function normalizePort(value, fallback = 0) {
  const n = Number(value)
  if (!Number.isFinite(n)) return fallback
  const p = Math.floor(n)
  if (p < 1 || p > 65535) return fallback
  return p
}

function stateIsUnknown(snapshot, module) {
  const fields = {
    account: [
      'lockoutUnknown',
      'guestUnknown',
      'autoLogonUnknown',
      'passwordUnknown',
      'passwordComplexityUnknown',
    ],
    rdp: ['rdpNLAUnknown'],
    firewall: ['riskPortsUnknown'],
    harden: ['smb1Unknown', 'winrmUnknown', 'anonymousUnknown'],
    defender: ['defenderUnknown'],
    time: ['timeUnknown'],
    update: ['updateUnknown'],
  }
  if ((fields[module] || []).some((key) => snapshot[key] === true)) return true
  if (module === 'firewall')
    return (
      snapshot.pingState === 'unknown' ||
      ['firewallDomain', 'firewallPrivate', 'firewallPublic'].some(
        (key) => snapshot[key] && !['开', '关'].includes(snapshot[key]),
      )
    )
  return module === 'rdp' && snapshot.rdpAvailable === false
}
