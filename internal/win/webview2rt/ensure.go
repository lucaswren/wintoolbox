package webview2rt

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"wintoolbox/internal/locale"

	"wintoolbox/internal/win/syscmd"
)

// Ensure makes sure WebView2 Evergreen Runtime is installed.
// If missing, it downloads the official bootstrapper (with progress UI),
// installs silently, and blocks concurrent downloads via a named mutex.
// Callers should present errors to the user (Ensure itself does not MessageBox).
func Ensure() error {
	if IsInstalled() {
		return nil
	}

	return withSetupLock(func() error {
		if IsInstalled() {
			return nil
		}
		return downloadAndInstall()
	})
}

func downloadAndInstall() error {
	prog := openProgress("WinToolBox", locale.Text("Connecting to Microsoft to download WebView2…", "正在连接 Microsoft 下载 WebView2…", "正在連線 Microsoft 下載 WebView2…"))
	defer prog.Close()

	dest := installerPath()
	prog.SetStatus(locale.Text("Downloading the WebView2 runtime…", "正在下载 WebView2 运行时…", "正在下載 WebView2 執行階段…"))
	if err := downloadBootstrapper(dest, prog.SetProgress); err != nil {
		return fmt.Errorf(locale.Text("Failed to download WebView2: %w", "下载 WebView2 失败: %w", "下載 WebView2 失敗: %w"), err)
	}
	// Best-effort: remove Zone.Identifier ADS so the silent installer is less
	// likely to be blocked by SmartScreen/Defender.
	_ = syscmd.UnblockFile(dest)

	prog.SetStatus(locale.Text("Verifying the installer digital signature…", "正在校验安装包数字签名…", "正在驗證安裝程式數位簽章…"))
	if err := verifyMicrosoftSigned(dest); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf(locale.Text("WebView2 installer signature verification failed (file deleted): %w", "WebView2 安装包签名校验失败（已删除文件）: %w", "WebView2 安裝程式簽章驗證失敗（已刪除檔案）: %w"), err)
	}

	prog.SetMarquee(locale.Text("Installing the WebView2 runtime; please wait…\nKeep this window open during installation.", "正在安装 WebView2 运行时，请稍候…\n（安装期间请勿关闭本窗口）", "正在安裝 WebView2 執行階段，請稍候…\n（安裝期間請勿關閉本視窗）"))
	if err := runSilentInstall(dest); err != nil {
		return fmt.Errorf(locale.Text("Failed to install WebView2: %w", "安装 WebView2 失败: %w", "安裝 WebView2 失敗: %w"), err)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if IsInstalled() {
			prog.SetStatus(locale.Text("WebView2 installation completed", "WebView2 安装完成", "WebView2 安裝完成"))
			time.Sleep(400 * time.Millisecond)
			_ = os.Remove(dest)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	_ = os.Remove(dest)
	return errors.New(locale.Text("The installer finished, but WebView2 was not detected. Restart and try again, or install it manually.", "安装程序已结束，但仍未检测到 WebView2 运行时；请重启后再试或手动安装", "安裝程式已結束，但仍未偵測到 WebView2 執行階段；請重新啟動後再試或手動安裝"))
}

func runSilentInstall(installer string) error {
	cmd := exec.Command(installer, "/silent", "/install")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if msg == "" {
			msg = err.Error()
		}
		cmd2 := exec.Command(installer, "/install")
		if err2 := cmd2.Run(); err2 != nil {
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}
