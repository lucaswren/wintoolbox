package harden

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"

	"wintoolbox/internal/win/syscmd"
)

const (
	lsaKeyPath   = `SYSTEM\CurrentControlSet\Control\Lsa`
	smb1KeyPath  = `SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters`
	winRMService = "WinRM"
	winRMBlockV4 = "WinToolbox-Block-WinRM-5985"
	winRMBlockV6 = "WinToolbox-Block-WinRM-5986"
)

// Status aggregates SMBv1 / WinRM / anonymous enumeration hardening state.
type Status struct {
	Smb1Disabled     bool
	Smb1Unknown      bool
	Smb1Detail       string
	WinRMHardened    bool
	WinRMUnknown     bool
	WinRMDetail      string
	AnonymousOK      bool
	AnonymousUnknown bool
	AnonymousDetail  string
}

// DisableSMBv1 turns off SMB1 protocol for the SMB server (and best-effort optional feature).
func DisableSMBv1() error {
	out, err := syscmd.RunPS(`
$ErrorActionPreference='Stop'
try {
  Set-SmbServerConfiguration -EnableSMB1Protocol $false -Force -ErrorAction Stop
  Write-Output 'OK'
} catch {
  Write-Output ('ERR:' + $_.Exception.Message)
  exit 1
}
`)
	psOK := err == nil && strings.Contains(out, "OK")
	psMsg := strings.TrimSpace(out)
	if psMsg == "" && err != nil {
		psMsg = err.Error()
	}

	// Registry fallback / reinforcement.
	k, kerr := registry.OpenKey(registry.LOCAL_MACHINE, smb1KeyPath, registry.SET_VALUE)
	if kerr == nil {
		_ = k.SetDWordValue("SMB1", 0)
		k.Close()
	}

	// Best-effort disable optional feature (may require reboot; ignore failure).
	_, _ = syscmd.RunPS(`
$ErrorActionPreference='SilentlyContinue'
Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -NoRestart -ErrorAction SilentlyContinue | Out-Null
`)

	disabled, unknown, detail := smb1State()
	if disabled {
		return nil
	}
	if !psOK {
		return fmt.Errorf("禁用 SMBv1 失败: %s；当前: %s", psMsg, detail)
	}
	if unknown {
		return fmt.Errorf("已提交禁用 SMBv1，但无法复查（%s）。可能需重启后生效", detail)
	}
	return fmt.Errorf("已提交禁用 SMBv1，但复查仍启用（可能被组策略覆盖）")
}

// HardenWinRM disables the WinRM service and blocks inbound 5985/5986.
func HardenWinRM() error {
	var errs []string

	_, _ = syscmd.Run("sc", "stop", winRMService)
	out, err := syscmd.Run("sc", "config", winRMService, "start=", "disabled")
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		// Chinese locale may need different spacing; try alternate.
		out2, err2 := syscmd.Run("sc", "config", winRMService, "start=disabled")
		if err2 != nil {
			msg2 := strings.TrimSpace(out2)
			if msg2 == "" {
				msg2 = err2.Error()
			}
			errs = append(errs, "服务: "+msg+"; "+msg2)
		}
	}

	if err := ensureBlockTCP(winRMBlockV4, 5985); err != nil {
		errs = append(errs, "5985: "+err.Error())
	}
	if err := ensureBlockTCP(winRMBlockV6, 5986); err != nil {
		errs = append(errs, "5986: "+err.Error())
	}

	hardened, unknown, detail := winrmState()
	if hardened {
		return nil
	}
	if len(errs) > 0 {
		return fmt.Errorf("限制 WinRM 失败: %s；当前: %s", strings.Join(errs, "; "), detail)
	}
	if unknown {
		return fmt.Errorf("已提交限制 WinRM，但无法复查: %s", detail)
	}
	return fmt.Errorf("已提交限制 WinRM，但复查未完全生效: %s", detail)
}

func ensureBlockTCP(name string, port uint32) error {
	portStr := strconv.FormatUint(uint64(port), 10)
	ps := fmt.Sprintf(`
$ErrorActionPreference='Stop'
$name='%s'
$port=%s
$r = Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue
if ($r) {
  Set-NetFirewallRule -Name $name -Direction Inbound -Action Block -Enabled True -Profile Any -ErrorAction Stop | Out-Null
  $r | Get-NetFirewallPortFilter | Set-NetFirewallPortFilter -Protocol TCP -LocalPort $port -ErrorAction Stop | Out-Null
} else {
  New-NetFirewallRule -DisplayName $name -Name $name -Direction Inbound -Action Block -Protocol TCP -LocalPort $port -Profile Any -Enabled True -Description 'WinToolbox block WinRM port' | Out-Null
}
Write-Output 'OK'
`, name, portStr)
	out, err := syscmd.RunPS(ps)
	if err == nil && strings.Contains(out, "OK") {
		return nil
	}
	psMsg := strings.TrimSpace(out)
	if psMsg == "" && err != nil {
		psMsg = err.Error()
	}
	_, _ = syscmd.Run("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
	out2, err2 := syscmd.Run("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+name,
		"dir=in",
		"action=block",
		"protocol=TCP",
		"localport="+portStr,
		"profile=domain,private,public",
		"enable=yes",
	)
	if err2 != nil {
		msg2 := strings.TrimSpace(out2)
		if msg2 == "" {
			msg2 = err2.Error()
		}
		return fmt.Errorf("PowerShell: %s; netsh: %s", psMsg, msg2)
	}
	return nil
}

func anonymousState() (ok, unknown bool, detail string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, lsaKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, true, "无法读取 LSA 匿名限制"
	}
	defer k.Close()

	ra, raOK := readDword(k, "RestrictAnonymous")
	ras, rasOK := readDword(k, "RestrictAnonymousSAM")

	parts := []string{}
	if raOK {
		parts = append(parts, fmt.Sprintf("RestrictAnonymous=%d", ra))
	} else {
		parts = append(parts, "RestrictAnonymous=未设置")
	}
	if rasOK {
		parts = append(parts, fmt.Sprintf("RestrictAnonymousSAM=%d", ras))
	} else {
		parts = append(parts, "RestrictAnonymousSAM=未设置")
	}
	detail = strings.Join(parts, " · ")

	// Hardened when RestrictAnonymous >= 1 and RestrictAnonymousSAM == 1.
	if raOK && rasOK && ra >= 1 && ras >= 1 {
		return true, false, detail
	}
	if !raOK && !rasOK {
		return false, true, detail
	}
	return false, false, detail
}

func readDword(k registry.Key, name string) (uint64, bool) {
	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		return 0, false
	}
	return v, true
}

// RestrictAnonymous sets RestrictAnonymous=1 and RestrictAnonymousSAM=1.
func RestrictAnonymous() error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, lsaKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开 LSA 注册表失败: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("RestrictAnonymous", 1); err != nil {
		return fmt.Errorf("写入 RestrictAnonymous 失败: %w", err)
	}
	if err := k.SetDWordValue("RestrictAnonymousSAM", 1); err != nil {
		return fmt.Errorf("写入 RestrictAnonymousSAM 失败: %w", err)
	}

	ok, unknown, detail := anonymousState()
	if ok {
		return nil
	}
	if unknown {
		return fmt.Errorf("已写入匿名限制，但无法复查: %s", detail)
	}
	return fmt.Errorf("已写入匿名限制，但复查未完全生效: %s", detail)
}
