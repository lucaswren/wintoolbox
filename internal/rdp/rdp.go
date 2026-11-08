package rdp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/windows/registry"

	"wintoolbox/internal/win/port"
	"wintoolbox/internal/win/syscmd"
)

const (
	rdpKeyPath = `SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp`
	tsKeyPath  = `SYSTEM\CurrentControlSet\Control\Terminal Server`
	fwRuleName = "WinToolbox-RDP"
)

// Status holds current Remote Desktop settings.
type Status struct {
	Enabled    bool
	Port       uint32
	NLA        bool
	NLAUnknown bool
}

// GetStatus reads whether RDP is enabled, the listening port, and NLA.
func GetStatus() (Status, error) {
	port, err := GetPort()
	if err != nil {
		return Status{}, err
	}
	enabled, err := IsEnabled()
	if err != nil {
		return Status{}, err
	}
	nla, nlaErr := IsNLAEnabled()
	if nlaErr != nil {
		return Status{Enabled: enabled, Port: port, NLA: false, NLAUnknown: true}, nil
	}
	return Status{Enabled: enabled, Port: port, NLA: nla, NLAUnknown: false}, nil
}

// GetPort returns the current RDP TCP port.
func GetPort() (uint32, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, rdpKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return 0, fmt.Errorf("读取 RDP 端口失败: %w", err)
	}
	defer k.Close()

	port, _, err := k.GetIntegerValue("PortNumber")
	if err != nil {
		return 0, fmt.Errorf("读取 PortNumber 失败: %w", err)
	}
	return uint32(port), nil
}

// IsEnabled reports whether Remote Desktop connections are allowed.
func IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, tsKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("读取远程桌面开关失败: %w", err)
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("fDenyTSConnections")
	if err != nil {
		return false, fmt.Errorf("读取 fDenyTSConnections 失败: %w", err)
	}
	return v == 0, nil
}

func setDenyTSConnections(deny uint32) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, tsKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开远程桌面注册表失败: %w", err)
	}
	defer k.Close()
	if err := k.SetDWordValue("fDenyTSConnections", deny); err != nil {
		return fmt.Errorf("写入远程桌面开关失败: %w", err)
	}
	return nil
}

// SetEnabled enables or disables Remote Desktop.
// Enable: firewall rule first, then registry (rollback registry if rule fails after write).
// Disable: registry first, then disable custom rule.
func SetEnabled(enabled bool) error {
	if enabled {
		p, err := GetPort()
		if err != nil {
			return fmt.Errorf("读取端口失败，未开启远程桌面: %w", err)
		}
		if err := ensureCustomRuleState(p, true); err != nil {
			return fmt.Errorf("防火墙同步失败，未开启远程桌面: %w", err)
		}
		if err := setDenyTSConnections(0); err != nil {
			_, _ = syscmd.Run("netsh", "advfirewall", "firewall", "set", "rule",
				"name="+fwRuleName, "new", "enable=no")
			return err
		}
		scheduleBuiltinRDPGroup(true)
		return nil
	}

	if err := setDenyTSConnections(1); err != nil {
		return err
	}
	_, _ = syscmd.Run("netsh", "advfirewall", "firewall", "set", "rule",
		"name="+fwRuleName, "new", "enable=no")
	scheduleBuiltinRDPGroup(false)
	return nil
}

// ensureCustomRuleState creates/updates WinToolbox-RDP via netsh and sets enable yes/no.
func ensureCustomRuleState(p uint32, enabled bool) error {
	if err := ValidatePort(p); err != nil {
		return err
	}
	portStr := strconv.FormatUint(uint64(p), 10)
	en := "no"
	if enabled {
		en = "yes"
	}

	_, err := syscmd.Run("netsh", "advfirewall", "firewall", "set", "rule",
		"name="+fwRuleName, "new",
		"enable="+en,
		"dir=in",
		"action=allow",
		"protocol=TCP",
		"localport="+portStr,
		"profile=any",
	)
	if err == nil {
		if enabled {
			if !firewallAllowsPort(portStr) {
				return fmt.Errorf("防火墙规则校验失败，未确认端口 %s 已放行", portStr)
			}
		}
		return nil
	}

	out, addErr := syscmd.Run("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+fwRuleName,
		"dir=in",
		"action=allow",
		"protocol=TCP",
		"localport="+portStr,
		"profile=any",
		"enable="+en,
		"description=WinToolbox RDP custom port firewall rule",
	)
	if addErr != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = addErr.Error()
		}
		return fmt.Errorf("创建防火墙规则失败: %s", msg)
	}
	if enabled && !firewallAllowsPort(portStr) {
		return fmt.Errorf("防火墙规则校验失败，未确认端口 %s 已放行", portStr)
	}
	return nil
}

var (
	builtinDesired atomic.Bool
	builtinSeq     atomic.Uint64
	builtinKick    = make(chan struct{}, 1)
	builtinOnce    sync.Once
)

func scheduleBuiltinRDPGroup(enabled bool) {
	builtinOnce.Do(func() {
		go builtinRDPGroupWorker()
	})
	builtinDesired.Store(enabled)
	builtinSeq.Add(1)
	select {
	case builtinKick <- struct{}{}:
	default:
	}
}

func builtinRDPGroupWorker() {
	for range builtinKick {
		for {
			seq := builtinSeq.Load()
			want := builtinDesired.Load()
			syncBuiltinRDPGroup(want)
			if builtinSeq.Load() == seq {
				break
			}
		}
	}
}

func syncBuiltinRDPGroup(enabled bool) {
	enableValue := "No"
	if enabled {
		enableValue = "Yes"
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = syscmd.RunQuick("netsh", "advfirewall", "firewall", "set", "rule",
			"group=远程桌面", "new", "enable="+enableValue)
	}()
	go func() {
		defer wg.Done()
		_, _ = syscmd.RunQuick("netsh", "advfirewall", "firewall", "set", "rule",
			"group=Remote Desktop", "new", "enable="+enableValue)
	}()
	wg.Wait()
}

// ValidatePort checks whether a port is usable for RDP.
func ValidatePort(p uint32) error {
	if err := port.ValidTCP(p); err != nil {
		return err
	}
	reserved := map[uint32]string{
		22:    "SSH",
		23:    "Telnet",
		25:    "SMTP",
		53:    "DNS",
		80:    "HTTP",
		110:   "POP3",
		135:   "RPC",
		139:   "NetBIOS",
		443:   "HTTPS",
		445:   "SMB",
		1433:  "SQL Server",
		3306:  "MySQL",
		5432:  "PostgreSQL",
		5985:  "WinRM",
		5986:  "WinRM HTTPS",
		27017: "MongoDB",
	}
	if name, ok := reserved[p]; ok {
		return fmt.Errorf("端口 %d 常被 %s 占用，建议换一个", p, name)
	}
	return nil
}

func firewallAllowsPort(portStr string) bool {
	out, err := syscmd.RunPS(fmt.Sprintf(
		`$r=Get-NetFirewallRule -Name '%s' -ErrorAction SilentlyContinue; if(-not $r){Write-Output 'NO'; exit 0}; $p=$r|Get-NetFirewallPortFilter; Write-Output ("ENABLED=" + $r.Enabled + ";PORT=" + $p.LocalPort + ";ACTION=" + $r.Action)`,
		fwRuleName,
	))
	if err == nil {
		enabled := strings.Contains(out, "ENABLED=True") || strings.Contains(out, "ENABLED=true")
		actionOK := strings.Contains(out, "ACTION=Allow") || strings.Contains(out, "ACTION=2") ||
			strings.Contains(strings.ToLower(out), "allow")
		if enabled && actionOK && portTokenExact(out, "PORT=", portStr) {
			return true
		}
	}

	out, err = syscmd.Run("netsh", "advfirewall", "firewall", "show", "rule", "name="+fwRuleName, "verbose")
	if err != nil || !strings.Contains(out, fwRuleName) {
		return false
	}
	allow := strings.Contains(strings.ToLower(out), "allow") || strings.Contains(out, "允许")
	return allow && netshLocalPortExact(out, portStr) && netshRuleEnabled(out)
}

func portTokenExact(line, prefix, portStr string) bool {
	idx := strings.Index(line, prefix+portStr)
	if idx < 0 {
		return false
	}
	end := idx + len(prefix) + len(portStr)
	if end < len(line) {
		c := line[end]
		if c >= '0' && c <= '9' {
			return false
		}
	}
	return true
}

func netshLocalPortExact(out, portStr string) bool {
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if !strings.Contains(lower, "localport") && !strings.Contains(line, "本地端口") {
			continue
		}
		if idx := strings.Index(line, ":"); idx >= 0 {
			val := strings.TrimSpace(line[idx+1:])
			for _, part := range strings.Split(val, ",") {
				part = strings.TrimSpace(part)
				if part == portStr {
					return true
				}
			}
		}
	}
	return false
}

func netshRuleEnabled(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(line, "已启用") {
			return strings.Contains(line, "是") || strings.Contains(strings.ToLower(line), "yes")
		}
		if strings.Contains(lower, "enabled") {
			return strings.Contains(lower, "yes") || strings.Contains(lower, "true")
		}
	}
	// Missing enabled line: do not treat as active.
	return false
}
