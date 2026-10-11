export const MAX_LOG_LINES = 500

export function appendBoundedLog(lines, message) {
  lines.push(message)
  if (lines.length > MAX_LOG_LINES) lines.splice(0, lines.length - MAX_LOG_LINES)
}

export function logLevel(message) {
  if (/失败|失敗|异常|異常|错误|錯誤|failed|failure|error/i.test(message)) return 'error'
  if (/未知|超时|超時|提示|稍候|unknown|timed out|notice|please wait/i.test(message))
    return 'warning'
  return 'info'
}
