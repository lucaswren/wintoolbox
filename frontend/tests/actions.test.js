import test from 'node:test'
import assert from 'node:assert/strict'
import { createRunner } from '../src/composables/actionRunner.js'
import { mergeModuleStatus } from '../src/composables/moduleStatus.js'
import { appendBoundedLog, MAX_LOG_LINES } from '../src/composables/logBuffer.js'
import { useAppState } from '../src/composables/useAppState.js'

globalThis.ElMessage = { info() {}, success() {}, warning() {}, error() {} }
globalThis.ElMessageBox = { async confirm() {} }

test('partial failure triggers state verification and releases busy', async () => {
  const busy = { value: false }
  let refreshed = 0
  const run = createRunner({
    busy,
    appendLog() {},
    async refresh() {
      refreshed++
    },
  })
  assert.equal(
    await run(async () => {
      throw new Error('service restart failed')
    }, 'success'),
    false,
  )
  assert.equal(refreshed, 1)
  assert.equal(busy.value, false)
})
test('cancelled confirmation performs no write or refresh', async () => {
  globalThis.ElMessageBox.confirm = async () => {
    throw new Error('cancel')
  }
  let calls = 0
  const run = createRunner({
    busy: { value: false },
    appendLog() {},
    async refresh() {
      calls++
    },
  })
  await run(
    async () => {
      calls++
    },
    'success',
    { confirm: 'confirm' },
  )
  assert.equal(calls, 0)
  globalThis.ElMessageBox.confirm = async () => {}
})
test('module refresh preserves unrelated state and unknowns', () => {
  const previous = { rdpPort: 3389, updateDisabled: true }
  const next = mergeModuleStatus(
    previous,
    { rdpPort: 0, updateDisabled: false, updateUnknown: true },
    'update',
  )
  assert.equal(next.rdpPort, 3389)
  assert.equal(next.updateUnknown, true)
})
test('long sessions retain the newest log entries', () => {
  const lines = []
  for (let i = 0; i < MAX_LOG_LINES + 50; i++) appendBoundedLog(lines, String(i))
  assert.equal(lines.length, MAX_LOG_LINES)
  assert.equal(lines[0], '50')
})

test('operation verifies its original module after navigation', async () => {
  let active = 'rdp'
  let checked
  const run = createRunner({
    busy: { value: false },
    appendLog() {},
    getModule: () => active,
    async refresh(_showLog, module) {
      checked = module
    },
  })
  await run(async () => {
    active = 'account'
    throw new Error('partial failure')
  }, 'success')
  assert.equal(checked, 'rdp')
})

test('successful status refresh exposes collection time to the header', async () => {
  const previousWindow = globalThis.window
  globalThis.window = {
    go: { ui: { App: { GetStatus: async () => ({ accounts: [], warnings: [] }) } } },
  }
  try {
    const app = useAppState({ appendLog() {} })
    await app.refresh()
    assert.match(app.updatedAt.all, /\d{2}:\d{2}:\d{2}/)
    assert.equal(app.loading.value, false)
  } finally {
    globalThis.window = previousWindow
  }
})
