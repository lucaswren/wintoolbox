import test from 'node:test'
import assert from 'node:assert/strict'
import { api } from '../src/api/index.js'
import { useRdpSafety } from '../src/composables/useRdpSafety.js'

test('RDP safety refresh coalesces requests and retains pending state on errors', async () => {
  const original = api.getRdpSafetyStatus
  try {
    const safety = useRdpSafety()
    let resolve,
      calls = 0
    api.getRdpSafetyStatus = () => {
      calls++
      return new Promise((done) => {
        resolve = done
      })
    }
    const first = safety.refresh()
    assert.equal(safety.refresh(), first)
    assert.equal(calls, 1)
    assert.equal(safety.loading.value, true)
    resolve({ pending: true, transaction: 'transaction-one', canConfirm: true })
    await first
    assert.equal(safety.status.value.transaction, 'transaction-one')
    api.getRdpSafetyStatus = async () => {
      throw Error('provider unavailable')
    }
    await safety.refresh()
    assert.equal(safety.status.value.pending, true)
    assert.equal(safety.status.value.transaction, 'transaction-one')
    assert.equal(safety.status.value.canConfirm, false)
    assert.equal(safety.error.value, 'provider unavailable')
    assert.equal(safety.loading.value, false)
    api.getRdpSafetyStatus = async () => ({ pending: false, phase: 'rolled-back' })
    await safety.refresh()
    assert.equal(safety.status.value.pending, false)
    assert.equal(safety.error.value, '')
  } finally {
    api.getRdpSafetyStatus = original
  }
})
