package defender

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"

	"wintoolbox/internal/win/snapshot"
	"wintoolbox/internal/win/syscmd"
)

const (
	policyKeyPath = `SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`
	backupKeyPath = `SOFTWARE\WinToolbox\DefenderBackup`
)

// Status describes Windows Defender realtime protection.
type Status struct {
	Disabled bool
	Unknown  bool
	Detail   string
}

// GetStatus reports whether realtime monitoring appears disabled.
func GetStatus() Status { return GetStatusContext(context.Background()) }

func GetStatusContext(ctx context.Context) Status {
	rtOff, hasPref := readRealtimeDisabledContext(ctx)
	policyOff, hasPolicy := readPolicyDisabled()

	parts := []string{}
	if hasPref {
		parts = append(parts, fmt.Sprintf("实时防护关闭=%v", rtOff))
	} else {
		parts = append(parts, "实时防护状态=未知")
	}
	if hasPolicy {
		parts = append(parts, fmt.Sprintf("策略禁用=%v", policyOff))
	}

	unknown := !hasPref
	disabled := hasPref && rtOff
	detail := strings.Join(parts, " · ")
	return Status{Disabled: disabled, Unknown: unknown, Detail: detail}
}

// Disable turns off Defender realtime protection.
func Disable() error {
	if err := saveSnapshotIfNeeded(); err != nil {
		return fmt.Errorf("保存防病毒配置备份失败，已中止关闭: %w", err)
	}

	out, err := syscmd.RunPS(`
$ErrorActionPreference='Stop'
try {
  Set-MpPreference -DisableRealtimeMonitoring $true -ErrorAction Stop
  Write-Output 'OK'
} catch {
  Write-Output ('ERR:' + $_.Exception.Message)
  exit 1
}
`)
	if err != nil || !strings.Contains(out, "OK") {
		msg := strings.TrimSpace(out)
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if looksLikeTamper(msg) {
			return fmt.Errorf("关闭实时防护失败（可能已开启篡改防护）。请先在 Windows 安全中心关闭「篡改防护」后重试:\n%s", msg)
		}
		if perr := setPolicyDisabled(true); perr != nil {
			return fmt.Errorf("关闭实时防护失败: %s; 策略回退: %v", msg, perr)
		}
		if rtOff, ok := readRealtimeDisabled(); ok && rtOff {
			// Policy fallback worked — treat as success.
			return nil
		}
		if rtOff, ok := readRealtimeDisabled(); ok && !rtOff {
			return fmt.Errorf("已写入策略尝试关闭实时防护，但实时防护仍在运行。请关闭「篡改防护」后重试，或在 Windows 安全中心手动关闭。原始错误: %s", msg)
		}
		return fmt.Errorf("命令关闭失败，已写入策略作为回退（效果可能需注销/重启后生效）: %s", msg)
	}
	if err := setPolicyDisabled(true); err != nil {
		// Preference succeeded; policy is best-effort.
		_ = err
	}
	if rtOff, ok := readRealtimeDisabled(); !ok || !rtOff {
		return fmt.Errorf("已提交关闭实时防护，但无法确认已关闭（可能被篡改防护或组策略覆盖）")
	}
	return nil
}

// Enable turns realtime protection back on (UI "恢复" always means force-on).
func Enable() error {
	if _, _, err := loadSnapshot(); err != nil {
		return fmt.Errorf("读取防病毒配置备份失败: %w", err)
	}

	// Clear WinToolbox disable policy so preference can take effect.
	if err := setPolicyDisabled(false); err != nil {
		return fmt.Errorf("清除防病毒禁用策略失败: %w", err)
	}

	out, err := syscmd.RunPS(`
$ErrorActionPreference='Stop'
try {
  Set-MpPreference -DisableRealtimeMonitoring $false -ErrorAction Stop
  Write-Output 'OK'
} catch {
  Write-Output ('ERR:' + $_.Exception.Message)
  exit 1
}
`)
	if err != nil || !strings.Contains(out, "OK") {
		msg := strings.TrimSpace(out)
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if looksLikeTamper(msg) {
			return fmt.Errorf("恢复实时防护失败（可能已开启篡改防护）。请在 Windows 安全中心手动开启:\n%s", msg)
		}
		return fmt.Errorf("恢复实时防护失败: %s", msg)
	}

	if rtOff, ok := readRealtimeDisabled(); !ok || rtOff {
		return fmt.Errorf("已提交启用实时防护，但无法确认已开启")
	}
	return clearBackupIfPresent()
}

type defenderSnap struct {
	hadPref   bool
	rtOff     bool
	hadPolicy bool
	policyOff bool
}

func readRealtimeDisabled() (disabled bool, ok bool) {
	return readRealtimeDisabledContext(context.Background())
}

func readRealtimeDisabledContext(ctx context.Context) (disabled bool, ok bool) {
	out, err := syscmd.RunPSQuickContext(ctx, `
try {
  $p = Get-MpPreference -ErrorAction Stop
  Write-Output ('RT=' + [string]$p.DisableRealtimeMonitoring)
} catch {
  Write-Output 'RT=?'
}
`)
	if err != nil {
		return false, false
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "RT=") {
			val := strings.TrimPrefix(line, "RT=")
			if val == "?" {
				return false, false
			}
			return strings.EqualFold(val, "True") || val == "1", true
		}
	}
	return false, false
}

func readPolicyDisabled() (disabled bool, ok bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, policyKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("DisableRealtimeMonitoring")
	if err != nil {
		return false, false
	}
	return v == 1, true
}

func setPolicyDisabled(disabled bool) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, policyKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if disabled {
		return k.SetDWordValue("DisableRealtimeMonitoring", 1)
	}
	if err := k.DeleteValue("DisableRealtimeMonitoring"); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

func saveSnapshotIfNeeded() error {
	if backupExists() {
		return nil
	}
	rtOff, ok := readRealtimeDisabled()
	if !ok {
		return fmt.Errorf("无法读取实时防护，已中止以避免保存未知配置")
	}
	policy, present, err := snapshot.ReadOptional(policyKeyPath, "DisableRealtimeMonitoring")
	if err != nil {
		return err
	}
	values := map[string]uint32{"HadPref": 1, "DisableRealtimeMonitoring": 0, "HadPolicy": 0, "PolicyDisableRealtimeMonitoring": policy}
	if rtOff {
		values["DisableRealtimeMonitoring"] = 1
	}
	if present {
		values["HadPolicy"] = 1
	}
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, backupKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return snapshot.Commit(k, values)
}

func backupExists() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, backupKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("Saved")
	return err == nil && v == 1
}

func loadSnapshot() (defenderSnap, bool, error) {
	fields := []string{"HadPref", "HadPolicy"}
	v, exists, err := snapshot.Load(backupKeyPath, fields)
	if err != nil || !exists {
		return defenderSnap{}, exists, err
	}
	if v["HadPref"] == 1 {
		fields = append(fields, "DisableRealtimeMonitoring")
	}
	if v["HadPolicy"] == 1 {
		fields = append(fields, "PolicyDisableRealtimeMonitoring")
	}
	v, _, err = snapshot.Load(backupKeyPath, fields)
	if err != nil {
		return defenderSnap{}, true, err
	}
	return defenderSnap{hadPref: v["HadPref"] == 1, rtOff: v["DisableRealtimeMonitoring"] == 1, hadPolicy: v["HadPolicy"] == 1, policyOff: v["PolicyDisableRealtimeMonitoring"] == 1}, true, nil
}

func clearBackup() error {
	return registry.DeleteKey(registry.LOCAL_MACHINE, backupKeyPath)
}

func looksLikeTamper(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(msg, "篡改") ||
		strings.Contains(lower, "tamper") ||
		strings.Contains(lower, "access is denied") ||
		strings.Contains(msg, "拒绝访问") ||
		strings.Contains(lower, "0x80070005")
}

func clearBackupIfPresent() error {
	err := clearBackup()
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
