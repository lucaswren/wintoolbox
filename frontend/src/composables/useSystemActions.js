import { t } from '../i18n/index.js'
import { api } from '../api/index.js'
import { createRunner, normalizePort, powerDelayLabel } from './actionRunner.js'

export function useSystemActions({
  busy,
  operation,
  appendLog,
  refresh,
  form,
  state,
  ensureFirewallRules,
  ensureRdpHistory,
  getModule,
}) {
  const run = createRunner({ busy, appendLog, refresh, getModule, operation })

  function normalizePowerDelay() {
    const n = Number(form.powerDelay)
    if (!Number.isFinite(n) || n < 0) return 60
    const d = Math.floor(n)
    if (d > 604800) return 604800
    return d
  }

  return {
    onAccRename: async () => {
      const user = form.accUser
      const newName = form.accNewName.trim()
      const ok = await run(
        async () => {
          await api.renameAccount(user, newName)
          form.accUser = newName
          form.accNewName = ''
        },
        t('用户名已修改，请在下次登录时使用新名称'),
        {
          confirm: t(
            '确认将用户 [{0}] 改名为 [{1}]？用户文件夹路径保持不变；当前登录用户建议注销后重新登录。',
            [user, newName],
          ),
        },
      )
      return ok
    },
    onAccPass: async () => {
      if (!form.accPass1 || !form.accPass2) {
        const msg = t('密码不能为空')
        appendLog(t('失败: ') + msg)
        ElMessage.warning(msg)
        return
      }
      if (form.accPass1 !== form.accPass2) {
        const msg = t('两次输入的密码不一致')
        appendLog(t('失败: ') + msg)
        ElMessage.warning(msg)
        return
      }
      const ok = await run(
        () => api.changeAccountPassword(form.accUser, form.accPass1, form.accPass2),
        t('密码已修改'),
        { confirm: t('确认修改用户 [{0}] 的密码？', [form.accUser]) },
      )
      if (ok) {
        form.accPass1 = ''
        form.accPass2 = ''
      }
    },
    onAccEnabled: (enabled) => {
      const action = enabled ? t('启用') : t('禁用')
      run(() => api.setAccountEnabled(form.accUser, enabled), t('账户已') + action, {
        confirm: t('确认{0}用户 [{1}]？', [action, form.accUser]),
      })
    },
    onAccAdmin: (admin) => {
      const action = admin ? t('设为管理员') : t('取消管理员')
      run(() => api.setAccountAdmin(form.accUser, admin), t('已') + action, {
        confirm: t('确认对 [{0}] {1}？', [form.accUser, action]),
      })
    },
    onDisableLockout: () =>
      run(() => api.disableAccountLockout(), t('账户锁定已关闭'), {
        confirm: t('确认关闭账户锁定策略？将锁定阈值设为 0，并解锁当前已锁定的本地账户。'),
        confirmType: 'warning',
      }),
    onEnableLockout: () =>
      run(() => api.enableAccountLockout(), t('账户锁定已开启'), {
        confirm: t('确认开启账户锁定策略？将设置阈值=10、锁定时间=30 分钟、复位窗口=30 分钟。'),
        confirmType: 'warning',
      }),
    onSetLockoutPolicy: async (threshold, durationMin, windowMin) => {
      const ok = await run(
        () => api.setAccountLockoutPolicy(threshold, durationMin, windowMin),
        t('账户锁定策略已更新'),
        {
          confirm: t('确认应用锁定策略？阈值={0} · 锁定时间={1} 分钟 · 复位窗口={2} 分钟。', [
            threshold,
            durationMin,
            windowMin,
          ]),
          confirmType: 'warning',
        },
      )
      return ok
    },
    onDisableGuest: () =>
      run(() => api.disableGuestAccount(), t('来宾账户已禁用'), {
        confirm: t('确认禁用来宾（Guest）账户？'),
        confirmType: 'warning',
      }),
    onDisableAutoLogon: () =>
      run(() => api.disableAutoLogon(), t('自动登录已关闭'), {
        confirm: t('确认关闭自动登录？将关闭 AutoAdminLogon 并清除 DefaultPassword。'),
        confirmType: 'warning',
      }),
    onEnablePasswordPolicy: (minLen) =>
      run(() => api.enablePasswordPolicy(minLen), t('密码策略已更新'), {
        confirm: t('确认应用密码策略？最短长度={0}，并开启密码复杂度。', [minLen]),
        confirmType: 'warning',
      }),
    onRdpPort: () => {
      const port = normalizePort(form.rdpPort, 0)
      if (!port) {
        ElMessage.warning(t('请输入有效端口（1-65535）'))
        return
      }
      form.rdpPort = port
      run(() => api.changeRdpPort(port), t('配置已提交，等待重新连接确认'), {
        confirm: t(
          '将 RDP 端口改为 {0}，约十秒后重启远程桌面服务并断开会话。请先放行云安全组，五分钟内重新连接确认，否则恢复。',
          [port],
        ),
      })
    },
    onRdpToggle: async () => {
      if (!state.rdpAvailable) {
        ElMessage.warning(t('远程桌面状态不可用，无法操作'))
        return
      }
      const enabling = !state.rdpEnabled
      const action = enabling ? t('开启') : t('关闭')
      return run(() => api.toggleRdp(), t('远程桌面已') + action, {
        confirm: t('确认{0}远程桌面？', [action]),
      })
    },
    onRdpNLA: (enabled) => {
      if (!state.rdpAvailable) {
        ElMessage.warning(t('远程桌面状态不可用，无法操作'))
        return
      }
      if (state.rdpNLAUnknown) {
        ElMessage.warning(t('NLA 状态未知，请先刷新后再操作'))
        return
      }
      return run(() => api.setRdpNLA(enabled), t('配置已提交，等待重新连接确认'), {
        confirm: enabled
          ? t('确认强制 NLA？旧版客户端可能无法连接；五分钟内需重新连接确认，否则恢复。')
          : t('确认关闭 NLA？未认证连接面将增大；五分钟内需重新连接确认，否则恢复。'),
        confirmType: 'warning',
      })
    },
    onRdpClearHistory: () =>
      run(() => api.clearRdpHistory(), t('已清理 RDP 连接记录'), {
        confirm: t(
          '确认清理本机 RDP 客户端连接记录？将删除历史主机/MRU、已保存用户名提示、TERMSRV 凭据、Default.rdp / 最近 .rdp，以及位图缓存（不影响远程桌面服务开关与端口）。',
        ),
        confirmType: 'warning',
        skipRefresh: true,
      }).then((detail) => {
        // Refresh list even on failure so residual rows are visible.
        ensureRdpHistory?.()
        if (detail === false) return
        if (typeof detail === 'string' && detail && detail !== '已清理 RDP 连接记录') {
          appendLog(detail)
        }
      }),
    onRdpClearHistoryByKind: (kind) => {
      const labelMap = {
        mru: t('最近主机'),
        server: t('已保存主机'),
        credential: t('凭据'),
        file: t('文件'),
        cache: t('缓存'),
      }
      const label = labelMap[kind] || kind
      const successMsg = t('已清理类型：{0} 的 RDP 连接记录', [label])
      return run(() => api.clearRdpHistoryByKind(kind), successMsg, {
        confirm: t('确认清理本机 RDP 连接记录中“{0}”类型的内容？此操作不影响其它类型记录。', [
          label,
        ]),
        confirmType: 'warning',
        skipRefresh: true,
      }).then((detail) => {
        ensureRdpHistory?.()
        if (detail === false) return
        if (typeof detail === 'string' && detail && detail !== successMsg) {
          appendLog(detail)
        }
      })
    },
    onRdpDeleteHistory: (row) => {
      if (!row?.kind) return
      const label = row.host || row.detail || row.kind
      run(
        () =>
          api.deleteRdpHistoryEntry(
            row.kind,
            row.host || '',
            row.username || '',
            row.detail || '',
            row.sid || '',
          ),
        t('已删除记录: {0}', [label]),
        {
          confirm: t('确认删除该条连接记录？\n类型={0} · {1}', [row.kind, label]),
          confirmType: 'warning',
          skipRefresh: true,
        },
      ).then((ok) => {
        if (ok !== false) ensureRdpHistory?.()
      })
    },
    onFwAllow: () => {
      const port = normalizePort(form.fwPort, 0)
      if (!port) {
        ElMessage.warning(t('请输入有效端口（1-65535）'))
        return
      }
      form.fwPort = port
      run(() => api.allowFirewallPort(port), t('已放行 TCP {0}', [port]), {
        confirm: t('确认放行入站 TCP {0}？', [port]),
      }).then((ok) => ok && ensureFirewallRules())
    },
    onFwRemove: () => {
      const port = normalizePort(form.fwPort, 0)
      if (!port) {
        ElMessage.warning(t('请输入有效端口（1-65535）'))
        return
      }
      form.fwPort = port
      run(() => api.removeFirewallPort(port), t('已删除 TCP {0} 规则', [port]), {
        confirm: t('确认删除 TCP {0} 的放行规则？', [port]),
      }).then((ok) => ok && ensureFirewallRules())
    },
    onFwRemovePort: (port) => {
      const p = normalizePort(port, 0)
      if (!p) {
        ElMessage.warning(t('端口无效'))
        return
      }
      run(() => api.removeFirewallPort(p), t('已删除 TCP {0} 规则', [p]), {
        confirm: t('确认删除 TCP {0} 的放行规则？', [p]),
      }).then((ok) => ok && ensureFirewallRules())
    },
    onFwClearAllAllowRules: () =>
      run(() => api.clearFirewallAllowRules(), t('已删除全部放行规则'), {
        confirm: t(
          '确认删除所有 WinToolbox 放行规则（WinToolbox-Allow-*）？这不会影响远程桌面开关/端口，也不会影响禁 ping 规则。',
        ),
        confirmType: 'warning',
      }).then((ok) => ok && ensureFirewallRules()),
    onFwEnableAll: () =>
      run(() => api.setFirewallEnabled(true), t('配置已提交，等待重新连接确认'), {
        confirm: t('开启所有防火墙配置文件可能中断连接；五分钟内需重新连接确认，否则恢复。'),
      }),
    onFwDisableAll: () =>
      run(() => api.setFirewallEnabled(false), t('防火墙已全部关闭'), {
        confirm: t('确认关闭全部防火墙配置文件？关闭后本机将失去网络过滤保护。'),
        confirmType: 'warning',
      }),
    onFwDisablePing: () =>
      run(() => api.disablePing(), t('已禁用 ping'), {
        confirm: t('确认禁止外部通过 ping 探测本机？这会阻止入站 IPv4 / IPv6 ICMP Echo 请求。'),
        confirmType: 'warning',
      }),
    onFwEnablePing: () =>
      run(() => api.enablePing(), t('已恢复 ping'), {
        confirm: t('确认恢复本机对 ping 的响应？将删除 WinToolbox 创建的禁 ping 规则。'),
      }),
    onFwBlockRisk: () =>
      run(() => api.blockRiskPorts(), t('已拦截高危入站端口'), {
        title: t('拦高危入站端口'),
        confirm: t(
          '确认拦切入站 TCP 135 / 139 / 445 / 5985 / 5986？不含 3389。若本机依赖 SMB 共享或 WinRM，请谨慎操作。',
        ),
        confirmType: 'warning',
      }),
    onFwUnblockRisk: () =>
      run(() => api.unblockRiskPorts(), t('已移除高危端口拦截'), {
        title: t('移除高危拦截'),
        confirm: t('确认移除 WinToolbox 创建的高危端口拦截规则？'),
      }),
    onDisableSMBv1: () =>
      run(() => api.disableSMBv1(), t('SMBv1 已禁用'), {
        confirm: t('确认禁用 SMBv1？极旧设备将无法通过 SMB1 访问共享，可能需重启后完全生效。'),
        confirmType: 'warning',
      }),
    onHardenWinRM: () =>
      run(() => api.hardenWinRM(), t('WinRM 已限制'), {
        confirm: t(
          '确认禁用 WinRM 服务并拦切入站 5985/5986？远程 PowerShell / 部分自动化工具将不可用。',
        ),
        confirmType: 'warning',
      }),
    onRestrictAnonymous: () =>
      run(() => api.restrictAnonymous(), t('已限制匿名枚举'), {
        confirm: t('确认限制匿名枚举（RestrictAnonymous / RestrictAnonymousSAM）？'),
        confirmType: 'warning',
      }),
    onApplyTZ: () =>
      run(() => api.applyTimeZone(form.timeZone), t('时区已更新'), {
        confirm: t('确认修改时区？'),
      }),
    onSaveNTP: () => {
      const server = String(form.ntpServer || '').trim()
      if (!server) {
        ElMessage.warning(t('NTP 服务器不能为空'))
        return
      }
      form.ntpServer = server
      return run(() => api.saveNTPServer(server), t('NTP 已保存'), {
        confirm: t('确认保存 NTP 服务器为 {0}？', [server]),
      })
    },
    onSyncNTP: () =>
      run(() => api.syncNTP(), t('时间已同步'), { confirm: t('确认立即同步网络时间？') }),
    onTestNTP: async () => {
      const server = String(form.ntpServer || '').trim()
      if (!server) {
        ElMessage.warning(t('NTP 服务器不能为空'))
        return
      }
      form.ntpServer = server
      const detail = await run(() => api.testNTPServer(server), null, { skipRefresh: true })
      if (typeof detail === 'string' && detail) {
        appendLog(t('NTP 测试: ') + detail)
        ElMessage.success(t('NTP 测试完成'))
      }
      return detail
    },
    onLock: () => run(() => api.lockPC(), t('已锁定')),
    onRestart: () => {
      const d = normalizePowerDelay()
      const when = powerDelayLabel(d)
      run(() => api.restartPC(d), t('已计划{0}重启', [when]), {
        confirm: t('确认{0}重启这台计算机？系统将强制关闭应用，请先保存工作。', [when]),
        confirmType: 'warning',
        skipRefresh: true,
      })
    },
    onShutdown: () => {
      const d = normalizePowerDelay()
      const when = powerDelayLabel(d)
      run(() => api.shutdownPC(d), t('已计划{0}关机', [when]), {
        confirm: t('确认{0}关闭这台计算机？系统将强制关闭应用，请先保存工作。', [when]),
        confirmType: 'error',
        skipRefresh: true,
      })
    },
    onAbortPower: () => run(() => api.abortPower(), t('已取消关机/重启'), { skipRefresh: true }),
    onDisableUpdate: () =>
      run(() => api.disableUpdate(), t('更新已关闭'), {
        confirm: t('确认关闭系统更新？关闭后将不再自动获取安全补丁。'),
      }),
    onEnableUpdate: () =>
      run(() => api.enableUpdate(), t('更新配置已还原'), {
        confirm: t(
          '确认还原关闭前的更新配置？原配置已禁用时将保持禁用。无备份时使用建议开启配置。',
        ),
      }),
    onDisableDefender: () =>
      run(() => api.disableDefender(), t('实时防护已关闭'), {
        confirm: t('确认关闭 Windows Defender 实时防护？本机将暂时失去实时扫描保护。'),
        confirmType: 'warning',
      }),
    onEnableDefender: () =>
      run(() => api.enableDefender(), t('实时防护已启用'), {
        confirm: t('确认启用 Windows Defender 实时防护？这会开启防护，不会还原原来的关闭状态。'),
      }),
  }
}
