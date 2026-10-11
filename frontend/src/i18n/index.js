import { ref } from 'vue'
import en from './en.json' with { type: 'json' }
import zhCN from './zh-CN.json' with { type: 'json' }
import zhTW from './zh-TW.json' with { type: 'json' }
import { detectLocale, formatMessage, SUPPORTED_LOCALES } from './locale.js'

const storageKey = 'wintoolbox.language'
function savedLocale() {
  try {
    const saved = globalThis.localStorage?.getItem(storageKey)
    return SUPPORTED_LOCALES.includes(saved) ? saved : null
  } catch {
    return null
  }
}
const explicitLocale = ref(savedLocale())
export const locale = ref(explicitLocale.value || detectLocale(globalThis.navigator?.languages))
const catalogs = { en, 'zh-CN': zhCN, 'zh-TW': zhTW }
const escapeRegex = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const patterns = Object.keys(en)
  .filter((key) => /\{\d+\}/.test(key))
  .sort((a, b) => b.replace(/\{\d+\}/g, '').length - a.replace(/\{\d+\}/g, '').length)
  .map((key) => ({
    key,
    indexes: [...key.matchAll(/\{(\d+)\}/g)].map((match) => Number(match[1])),
    regex: new RegExp(
      '^' +
        key
          .split(/(\{\d+\})/)
          .map((part) => (/^\{\d+\}$/.test(part) ? '([\\s\\S]*?)' : escapeRegex(part)))
          .join('') +
        '$',
    ),
  }))
const prefixes = Object.keys(en)
  .filter((key) => /[:：] $/.test(key))
  .sort((a, b) => b.length - a.length)

function translate(key, depth = 0) {
  const catalog = catalogs[locale.value]
  if (Object.hasOwn(catalog, key)) return catalog[key]
  if (depth >= 5) return key
  for (const pattern of patterns) {
    const match = pattern.regex.exec(key)
    if (!match) continue
    const params = []
    pattern.indexes.forEach((index, i) => {
      params[index] = translate(match[i + 1], depth + 1)
    })
    return formatMessage(catalog[pattern.key] ?? en[pattern.key], params)
  }
  for (const prefix of prefixes) {
    if (key.startsWith(prefix))
      return catalog[prefix] + translate(key.slice(prefix.length), depth + 1)
  }
  // Domain summaries join separately generated fields. Keep OS output unchanged.
  const parts = key.split(/( · |; )/)
  if (parts.length > 1) return parts.map((part) => translate(part, depth + 1)).join('')
  return key
}

function updateDocument() {
  if (globalThis.document) document.documentElement.lang = locale.value
}
export function setLocale(value) {
  if (!SUPPORTED_LOCALES.includes(value)) return
  explicitLocale.value = value
  locale.value = value
  try {
    globalThis.localStorage?.setItem(storageKey, value)
  } catch {
    // Language selection still works when storage is unavailable.
  }
  updateDocument()
}
export function useSystemLocale(value) {
  if (!explicitLocale.value && SUPPORTED_LOCALES.includes(value)) {
    locale.value = value
    updateDocument()
  }
}
export function t(key, params) {
  if (typeof key !== 'string') return key
  if (params) return formatMessage(catalogs[locale.value][key] ?? en[key] ?? key, params)
  return translate(key)
}
updateDocument()
