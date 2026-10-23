package rdp

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
)

// Serialize the entire check/write sequence across app instances and RDP
// sessions. Windows mutex ownership is thread-affine, so pin the goroutine.
// Recovery uses its own script mutex and remains independent of app processes.
func LockNetworkMutation() (func(), error) {
	return lockNetworkMutationNamed(`Global\WinToolBox-NetworkUI-v1`)
}
func lockNetworkMutationNamed(mutexName string) (func(), error) {
	runtime.LockOSThread()
	name, _ := windows.UTF16PtrFromString(mutexName)
	h, err := windows.CreateMutex(nil, false, name)
	if h == 0 || (err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS)) {
		if h != 0 {
			windows.CloseHandle(h)
		}
		runtime.UnlockOSThread()
		return nil, err
	}
	result, err := windows.WaitForSingleObject(h, 15000)
	if err != nil || (result != windows.WAIT_OBJECT_0 && result != windows.WAIT_ABANDONED) {
		windows.CloseHandle(h)
		runtime.UnlockOSThread()
		if err != nil {
			return nil, fmt.Errorf("无法锁定网络配置: %w", err)
		}
		return nil, fmt.Errorf("其他实例正在修改网络配置，请稍后重试")
	}
	return func() { windows.ReleaseMutex(h); windows.CloseHandle(h); runtime.UnlockOSThread() }, nil
}
