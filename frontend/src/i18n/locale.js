export const SUPPORTED_LOCALES = ['en', 'zh-CN', 'zh-TW']

export function resolveLocale(language) {
  const tag = String(language || '')
    .replaceAll('_', '-')
    .toLowerCase()
  if (/^zh-(hant|tw|hk|mo)(-|$)/.test(tag)) return 'zh-TW'
  if (tag === 'zh' || /^zh-(hans|cn|sg)(-|$)/.test(tag)) return 'zh-CN'
  return 'en'
}

export function detectLocale(languages = []) {
  return resolveLocale(languages[0])
}

export function formatMessage(message, params = []) {
  return message.replace(/\{(\d+)\}/g, (match, index) =>
    params[index] === undefined ? match : String(params[index]),
  )
}
