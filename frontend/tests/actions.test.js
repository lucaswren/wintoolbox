import test from 'node:test'
import assert from 'node:assert/strict'
import { createRunner } from '../src/composables/actionRunner.js'
import { mergeModuleStatus } from '../src/composables/moduleStatus.js'
import { appendBoundedLog, MAX_LOG_LINES } from '../src/composables/logBuffer.js'
import { useAppState } from '../src/composables/useAppState.js'

globalThis.ElMessage = { info() {}, success() {}, warning() {}, error() {} }
globalThis.ElMessageBox = { async confirm() {} }

test('pending operations expose running and checking phases before success', async () => {
  const operation = {}
  const busy = { value: false }
  let finishWrite, finishCheck
  const write = new Promise((resolve) => {
    finishWrite = resolve
  })
  const check = new Promise((resolve) => {
    finishCheck = resolve
  })
  const run = createRunner({ busy, operation, appendLog() {}, refresh: () => check })
  const promise = run(() => write, 'done')
  assert.equal(operation.phase, 'running')
  assert.equal(busy.value, true)
  finishWrite()
  await new Promise((resolve) => setImmediate(resolve))
  assert.equal(operation.phase, 'verifying')
  assert.equal(busy.value, true)
  finishCheck()
  assert.equal(await promise, true)
  assert.equal(operation.phase, 'completed')
  assert.equal(operation.detail, 'done')
  assert.equal(busy.value, false)
})

test('unknown or failed checks remain a persistent warning instead of success', async () => {
  for (const refresh of [
    async () => ({ riskPortsUnknown: true }),
    async () => {
      throw Error('offline')
    },
  ]) {
    const operation = {}
    const run = createRunner({
      busy: { value: false },
      operation,
      getModule: () => 'firewall',
      appendLog() {},
      refresh,
    })
    assert.equal(await run(async () => {}, 'done'), true)
    assert.equal(operation.phase, 'warning')
    assert.notEqual(operation.detail, 'done')
  }
})

test('partial write failure remains visible after a successful state check', async () => {
  const operation = {}
  const run = createRunner({
    busy: { value: false },
    operation,
    appendLog() {},
    refresh: async () => ({}),
  })
  assert.equal(
    await run(async () => {
      throw Error('port 445 failed')
    }, 'done'),
    false,
  )
  assert.equal(operation.phase, 'failed')
  assert.match(operation.detail, /445/)
})

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

test('overlapping refreshes keep results from different modules', async () => {
  const previousWindow = globalThis.window
  const finish = {}
  globalThis.window = {
    go: {
      ui: {
        App: {
          GetModuleStatus: (module) =>
            new Promise((resolve) => {
              finish[module] = resolve
            }),
        },
      },
    },
  }
  try {
    const app = useAppState({ appendLog() {} })
    const rdp = app.refresh(false, false, false, 'rdp')
    const update = app.refresh(false, false, false, 'update')
    finish.update({ updateDisabled: true, warnings: [] })
    await update
    assert.equal(app.loading.value, true)
    finish.rdp({ rdpPort: 3390, warnings: [] })
    await rdp
    assert.equal(app.state.updateDisabled, true)
    assert.equal(app.state.rdpPort, 3390)
    assert.equal(app.loading.value, false)
  } finally {
    globalThis.window = previousWindow
  }
})

test('a full refresh preserves a newer module refresh', async () => {
  const previousWindow = globalThis.window
  let finishAll
  globalThis.window = {
    go: {
      ui: {
        App: {
          GetStatus: () =>
            new Promise((resolve) => {
              finishAll = resolve
            }),
          GetModuleStatus: async () => ({ rdpPort: 3390, warnings: [] }),
        },
      },
    },
  }
  try {
    const app = useAppState({ appendLog() {} })
    const all = app.refresh()
    await app.refresh(false, false, false, 'rdp')
    finishAll({ rdpPort: 3389, updateDisabled: true, accounts: [], warnings: [] })
    await all
    assert.equal(app.state.rdpPort, 3390)
    assert.equal(app.state.updateDisabled, true)
  } finally {
    globalThis.window = previousWindow
  }
})
