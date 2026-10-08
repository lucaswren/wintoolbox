package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// Apply stages a verified package in a restricted directory before handing off.
func Apply() error {
	operationMu.Lock()
	defer operationMu.Unlock()
	info := Cached()
	if !info.HasUpdate || !info.Downloaded || !info.Verified || info.AssetSHA256 == "" {
		return fmt.Errorf("请先下载并校验更新")
	}
	ok, verified := fileReadyVerified(info.DownloadPath, info.AssetSize, info.AssetSHA256)
	if !ok || !verified {
		return fmt.Errorf("更新包不存在或校验失败，请重新下载")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(downloadDir(), 0700); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(downloadDir(), "install-")
	if err != nil {
		return err
	}
	if err := restrictDirectory(dir); err != nil {
		return fmt.Errorf("保护更新暂存目录失败: %w", err)
	}
	staged := filepath.Join(dir, "package.exe")
	input, err := os.ReadFile(info.DownloadPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(staged, input, 0600); err != nil {
		return err
	}
	if ok, verified := fileReadyVerified(staged, info.AssetSize, info.AssetSHA256); !ok || !verified {
		return fmt.Errorf("暂存更新包校验失败")
	}
	script := filepath.Join(dir, "apply.ps1")
	logPath := filepath.Join(downloadDir(), "last-install.log")
	if err := os.WriteFile(script, []byte(updateScript(exe, staged, info.AssetSHA256, logPath, os.Getpid())), 0600); err != nil {
		return err
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新器失败: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return fmt.Errorf("更新器提前退出: %v；日志: %s", err, logPath)
	case <-time.After(800 * time.Millisecond):
	}
	go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }()
	return nil
}

func restrictDirectory(path string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

func updateScript(target, source, hash, logPath string, pid int) string {
	return fmt.Sprintf(`$ErrorActionPreference='Stop'
$target=%s
$source=%s
$expected=%s
$log=%s
$procId=%d
$new=$target+'.new'
$backup=$target+'.previous'
$replaced=$false
function Write-Log($message) { Add-Content -LiteralPath $log -Value ((Get-Date -Format o)+' '+$message) -Encoding UTF8 }
function Get-SHA256($path) {
  $stream=[System.IO.File]::OpenRead($path)
  $algorithm=[System.Security.Cryptography.SHA256]::Create()
  try { return ([BitConverter]::ToString($algorithm.ComputeHash($stream))).Replace('-','') }
  finally { $stream.Dispose(); $algorithm.Dispose() }
}
try {
  Write-Log 'Waiting for application exit'
  $deadline=(Get-Date).AddMinutes(2)
  while (Get-Process -Id $procId -ErrorAction SilentlyContinue) {
    if ((Get-Date) -gt $deadline) { throw 'Application exit timeout' }
    Start-Sleep -Milliseconds 250
  }
  if ((Get-SHA256 $source) -ne $expected) { throw 'Staged SHA256 mismatch' }
  Copy-Item -LiteralPath $source -Destination $new -Force
  if ((Get-SHA256 $new) -ne $expected) { throw 'Copied SHA256 mismatch' }
  [System.IO.File]::Replace($new,$target,$backup,$true)
  $replaced=$true
  Unblock-File -LiteralPath $target -ErrorAction SilentlyContinue
  $child=Start-Process -FilePath $target -WorkingDirectory (Split-Path $target) -PassThru -WindowStyle Hidden
  Start-Sleep -Seconds 5
  if ($child.HasExited) { throw ('Updated application exited: '+$child.ExitCode) }
  Write-Log 'Update started successfully; previous executable retained'
} catch {
  Write-Log ('Update failed: '+$_.Exception.Message)
  if ($replaced -and (Test-Path -LiteralPath $backup)) {
    if ($child -and -not $child.HasExited) { Stop-Process -Id $child.Id -Force }
    Copy-Item -LiteralPath $backup -Destination $target -Force
    Write-Log 'Restored previous executable'
    Start-Process -FilePath $target -WorkingDirectory (Split-Path $target) -WindowStyle Hidden
  }
  exit 1
} finally {
  Remove-Item -LiteralPath $new -Force -ErrorAction SilentlyContinue
}
`, psSingleQuote(target), psSingleQuote(source), psSingleQuote(hash), psSingleQuote(logPath), pid)
}
