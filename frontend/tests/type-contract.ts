import { wailsCall } from '../src/api/wails.js'

// These deliberately-invalid calls ensure the generated contract stays active.
// @ts-expect-error method must exist on the Go bridge
wailsCall('TypoMethod')
// @ts-expect-error a boolean cannot be used for an RDP port
wailsCall('ChangeRdpPort', true)
// @ts-expect-error missing required delay
wailsCall('RestartPC')
async function verifyStatusContract() {
  const status = await wailsCall('GetModuleStatus', 'update', false)
  const known: boolean = status.updateUnknown
  return known
}
void verifyStatusContract
