import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import { resolveLocale, detectLocale, formatMessage } from '../src/i18n/locale.js'
import { locale, setLocale, useSystemLocale, t } from '../src/i18n/index.js'
import { MENUS, NTP_PRESETS } from '../src/constants.js'

test('display languages map by script and region, with English fallback', () => {
  for (const tag of ['zh-TW', 'zh-HK', 'zh-MO', 'zh-Hant', 'zh-Hant-CN', 'zh_HK']) {
    assert.equal(resolveLocale(tag), 'zh-Hant', tag)
  }
  for (const tag of ['zh-CN', 'zh-SG', 'zh-Hans', 'zh-Hans-TW', 'zh']) {
    assert.equal(resolveLocale(tag), 'zh-Hans', tag)
  }
  for (const tag of ['en-US', 'fr-FR', 'ja-JP', 'zh-XX', '', null]) {
    assert.equal(resolveLocale(tag), 'en', tag)
  }
  assert.equal(detectLocale(['fr-FR', 'zh-Hans']), 'en')
  assert.equal(detectLocale([]), 'en')
})

test('catalogs have identical keys and interpolation placeholders', () => {
  const catalogs = ['en', 'zh-Hans', 'zh-Hant'].map((name) =>
    JSON.parse(fs.readFileSync(new URL(`../src/i18n/${name}.json`, import.meta.url))),
  )
  const placeholders = (value) => [...value.matchAll(/\{\d+\}/g)].map((m) => m[0]).sort()
  for (const catalog of catalogs) {
    assert.deepEqual(Object.keys(catalog).sort(), Object.keys(catalogs[0]).sort())
    for (const [key, value] of Object.entries(catalog)) {
      assert.ok(value.trim(), key)
      assert.deepEqual(placeholders(value), placeholders(key), key)
      assert.ok(
        !/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/.test(value),
        key,
      )
    }
  }
  for (const value of Object.values(catalogs[0])) assert.ok(!/[\u3400-\u9fff]/.test(value), value)
})

test('manual choice persists, overrides Windows detection and updates navigation', async () => {
  const previousStorage = globalThis.localStorage
  const previousDocument = globalThis.document
  const saved = new Map()
  globalThis.localStorage = {
    getItem: (key) => saved.get(key),
    setItem: (key, value) => saved.set(key, value),
  }
  globalThis.document = { documentElement: {} }
  try {
    setLocale('zh-Hant')
    assert.equal(t('本机概览'), '本機概覽')
    assert.equal(MENUS[0].title, '本機概覽')
    assert.equal(saved.get('wintoolbox.language'), 'zh-Hant')
    assert.equal(document.documentElement.lang, 'zh-Hant')
    useSystemLocale('en')
    assert.equal(locale.value, 'zh-Hant')
    const restarted = await import('../src/i18n/index.js?restart')
    assert.equal(restarted.locale.value, 'zh-Hant')
    setLocale('en')
    assert.equal(MENUS[0].title, 'Overview')
    assert.equal(NTP_PRESETS[1].label, 'Alibaba Cloud')
    setLocale('unsupported')
    assert.equal(locale.value, 'en')
    setLocale('zh-Hans')
    assert.equal(MENUS[0].title, '本机概览')
  } finally {
    globalThis.localStorage = previousStorage
    globalThis.document = previousDocument
    setLocale('en')
  }
})

test('language selection works with blocked storage and system choice applies only before override', async () => {
  const previousStorage = globalThis.localStorage
  globalThis.localStorage = {
    getItem() {
      throw new Error('blocked')
    },
    setItem() {
      throw new Error('blocked')
    },
  }
  try {
    const fresh = await import('../src/i18n/index.js?blocked')
    fresh.useSystemLocale('zh-Hant')
    assert.equal(fresh.locale.value, 'zh-Hant')
    fresh.setLocale('zh-Hans')
    fresh.useSystemLocale('en')
    assert.equal(fresh.locale.value, 'zh-Hans')
  } finally {
    globalThis.localStorage = previousStorage
  }
})

test('formatted diagnostics translate without changing backend tokens or OS output', () => {
  setLocale('en')
  assert.equal(
    t('阈值=10 · 锁定时间=30 · 复位窗口=30'),
    'Threshold=10 · Lockout=30 · Reset window=30',
  )
  assert.equal(t('最短长度=12 · 复杂度=已开启'), 'Minimum length=12 · Complexity=Enabled')
  assert.equal(
    t('读取本地账户失败: 用户名不能为空'),
    'Failed to read local accounts: Username cannot be empty',
  )
  assert.equal(t('已下载并校验 v{0}', ['1.2.3']), 'Downloaded and verified v1.2.3')
  assert.equal(
    t('下载不完整: 期望 10 字节，实际 5'),
    'Incomplete download: expected 10 bytes, received 5',
  )
  assert.equal(
    t('The system cannot find the file specified.'),
    'The system cannot find the file specified.',
  )
  assert.equal(formatMessage('{0} / {1}', [0]), '0 / {1}')
  setLocale('zh-Hant')
  assert.equal(t('SMBv1 状态未知'), 'SMBv1 狀態未知')
})

test('legacy regional preferences migrate to script identifiers', async () => {
  const previous = globalThis.localStorage
  try {
    for (const [old, value] of [
      ['zh-CN', 'zh-Hans'],
      ['zh-TW', 'zh-Hant'],
    ]) {
      const saved = new Map([['wintoolbox.language', old]])
      globalThis.localStorage = {
        getItem: (key) => saved.get(key),
        setItem: (key, value) => saved.set(key, value),
      }
      const fresh = await import('../src/i18n/index.js?legacy=' + old)
      assert.equal(fresh.locale.value, value)
      assert.equal(saved.get('wintoolbox.language'), value)
    }
  } finally {
    globalThis.localStorage = previous
  }
})
