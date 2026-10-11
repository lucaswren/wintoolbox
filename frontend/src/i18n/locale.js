export const SUPPORTED_LOCALES = ['en', 'zh-Hans', 'zh-Hant']

// Language names are fixed autonyms; never translate them with the active locale.
export const LANGUAGE_OPTIONS = Object.freeze([
  { value: 'en', label: 'English' },
  { value: 'zh-Hant', label: '繁體中文' },
  { value: 'zh-Hans', label: '简体中文' },
])

export function resolveLocale(language) {
  const tag = String(language || '')
    .replaceAll('_', '-')
    .toLowerCase()
  if (/^zh-(hant|tw|hk|mo)(-|$)/.test(tag)) return 'zh-Hant'
  if (tag === 'zh' || /^zh-(hans|cn|sg)(-|$)/.test(tag)) return 'zh-Hans'
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
