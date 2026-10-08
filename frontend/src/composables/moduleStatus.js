const prefixes = {
  overview: ['overview'],
  account: ['accounts', 'lockout', 'guest', 'autoLogon', 'password'],
  rdp: ['rdp'],
  firewall: ['firewall', 'ping', 'riskPorts'],
  harden: ['smb1', 'winrm', 'anonymous'],
  defender: ['defender'],
  time: ['time', 'ntp'],
  update: ['update'],
}

export function mergeModuleStatus(previous, incoming, module) {
  const selected = prefixes[module]
  if (!selected) throw new Error('未知状态模块: ' + module)
  const result = { ...previous, warnings: incoming.warnings || [] }
  for (const [key, value] of Object.entries(incoming)) {
    if (selected.some((prefix) => key === prefix || key.startsWith(prefix))) result[key] = value
  }
  return result
}
