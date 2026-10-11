package webview2rt

import (
	"errors"
	"fmt"
	"time"
	"wintoolbox/internal/locale"

	"golang.org/x/sys/windows"
)

const mutexName = `Global\WinToolbox_WebView2Setup`

// withSetupLock runs fn while holding a machine-wide mutex so only one
// download/install runs at a time. Other instances wait, then re-check.
func withSetupLock(fn func() error) error {
	namePtr, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return err
	}
	handle, err := windows.CreateMutex(nil, false, namePtr)
	if handle == 0 {
		if err != nil {
			return fmt.Errorf(locale.Text("Failed to create the installation lock: %w", "创建安装锁失败: %w", "建立安裝鎖定失敗: %w"), err)
		}
		return errors.New(locale.Text("Failed to create the installation lock", "创建安装锁失败", "建立安裝鎖定失敗"))
	}
	defer windows.CloseHandle(handle)

	alreadyRunning := err == windows.ERROR_ALREADY_EXISTS
	timeout := 30 * time.Second
	if alreadyRunning {
		timeout = 15 * time.Minute
	}

	waitResult, waitErr := windows.WaitForSingleObject(handle, uint32(timeout.Milliseconds()))
	if waitErr != nil {
		return fmt.Errorf(locale.Text("Failed to acquire the installation lock: %w", "获取安装锁失败: %w", "取得安裝鎖定失敗: %w"), waitErr)
	}
	switch waitResult {
	case windows.WAIT_OBJECT_0, windows.WAIT_ABANDONED:
		defer windows.ReleaseMutex(handle)
		if IsInstalled() {
			return nil
		}
		return fn()
	case uint32(windows.WAIT_TIMEOUT):
		if IsInstalled() {
			return nil
		}
		if alreadyRunning {
			return errors.New(locale.Text("Timed out waiting for another WinToolbox instance to install WebView2. Try again later.", "等待其他 WinToolbox 完成 WebView2 安装超时，请稍后重试", "等待其他 WinToolbox 完成 WebView2 安裝逾時，請稍後重試"))
		}
		return errors.New(locale.Text("Unable to acquire the WebView2 installation lock. Try again later.", "无法获取 WebView2 安装锁，请稍后重试", "無法取得 WebView2 安裝鎖定，請稍後重試"))
	default:
		return fmt.Errorf(locale.Text("Failed to wait for the WebView2 installation lock (code %d)", "等待 WebView2 安装锁失败（代码 %d）", "等待 WebView2 安裝鎖定失敗（代碼 %d）"), waitResult)
	}
}
