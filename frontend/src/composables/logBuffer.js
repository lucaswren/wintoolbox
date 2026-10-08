export const MAX_LOG_LINES = 500

export function appendBoundedLog(lines, message) {
  lines.push(message)
  if (lines.length > MAX_LOG_LINES) lines.splice(0, lines.length - MAX_LOG_LINES)
}

export function logLevel(message) {
  if (/失败|异常|错误/.test(message)) return 'error'
  if (/未知|超时|提示|稍候/.test(message)) return 'warning'
  return 'info'
}
