package firewall

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"wintoolbox/internal/win/syscmd"
)

const riskRulePrefix = "WinToolbox-Block-Risk-"

// Default high-risk inbound TCP ports for one-click block preset.
// Intentionally excludes 3389 so RDP management remains usable.
var riskPorts = []uint32{135, 139, 445, 5985, 5986}

// RiskPortInfo is one port in the high-risk block preset.
type RiskPortInfo struct {
	Port    uint32 `json:"port"`
	Blocked bool   `json:"blocked"`
	Name    string `json:"name"`
}

// RiskBlockStatus summarizes the high-risk port block preset.
type RiskBlockStatus struct {
	Unknown    bool
	AllBlocked bool
	Partial    bool
	None       bool
	Detail     string
	Ports      []RiskPortInfo
}

func riskRuleName(port uint32) string {
	return riskRulePrefix + strconv.FormatUint(uint64(port), 10)
}

// BlockRiskPorts batches all writes and bounds write/fallback/check work together.
func BlockRiskPorts() error   { return changeRiskPorts(true) }
func UnblockRiskPorts() error { return changeRiskPorts(false) }

func changeRiskPorts(block bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, writeErr := syscmd.RunPSContext(ctx, riskMutationScript(block))
	var targets []uint32
	if block {
		st := GetRiskBlockStatusContext(ctx)
		if st.AllBlocked {
			return nil
		}
		for _, p := range riskPorts {
			blocked := false
			for _, info := range st.Ports {
				if info.Port == p {
					blocked = info.Blocked
				}
			}
			if !blocked {
				targets = append(targets, p)
			}
		}
	} else {
		present := probeRiskPresence(ctx)
		if riskRulesAbsent(present) {
			return nil
		}
		for _, p := range riskPorts {
			exists, known := present[p]
			if !known || exists {
				targets = append(targets, p)
			}
		}
	}
	for _, p := range targets {
		if ctx.Err() != nil {
			break
		}
		name, value := riskRuleName(p), strconv.FormatUint(uint64(p), 10)
		if !block {
			_, _ = syscmd.RunQuickContext(ctx, "netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
			continue
		}
		_, err := syscmd.RunQuickContext(ctx, "netsh", "advfirewall", "firewall", "set", "rule", "name="+name, "new", "dir=in", "action=block", "protocol=TCP", "localport="+value, "profile=any", "enable=yes")
		if err != nil {
			_, _ = syscmd.RunQuickContext(ctx, "netsh", "advfirewall", "firewall", "add", "rule", "name="+name, "dir=in", "action=block", "protocol=TCP", "localport="+value, "profile=any", "enable=yes")
		}
	}
	if block {
		st := GetRiskBlockStatusContext(ctx)
		if st.AllBlocked {
			return nil
		}
		return fmt.Errorf("已提交拦截，但复查未全部生效: %s; %v", st.Detail, riskFailure(writeErr, ctx.Err()))
	}
	if riskRulesAbsent(probeRiskPresence(ctx)) {
		return nil
	}
	return fmt.Errorf("已提交移除规则，但复查仍存在规则或状态未知: %v", riskFailure(writeErr, ctx.Err()))
}

func riskFailure(writeErr, timeoutErr error) error {
	if timeoutErr != nil {
		return timeoutErr
	}
	if writeErr != nil {
		return writeErr
	}
	return fmt.Errorf("请刷新后检查当前规则")
}

func riskRulesAbsent(present map[uint32]bool) bool {
	if len(present) != len(riskPorts) {
		return false
	}
	for _, exists := range present {
		if exists {
			return false
		}
	}
	return true
}

func probeRiskPresence(ctx context.Context) map[uint32]bool {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; $rules=@(Get-NetFirewallRule -ErrorAction Stop); foreach($name in @('%s')) { if($rules | Where-Object Name -eq $name){ Write-Output ($name+'=1') } else { Write-Output ($name+'=0') } }`, riskNames())
	out, err := syscmd.RunPSQuickContext(ctx, script)
	if err != nil {
		return nil
	}
	return parseRiskResults(out)
}

func parseRiskResults(text string) map[uint32]bool {
	out := make(map[uint32]bool)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		for _, p := range riskPorts {
			if line == riskRuleName(p)+"=0" {
				out[p] = false
			}
			if line == riskRuleName(p)+"=1" {
				out[p] = true
			}
		}
	}
	return out
}

func riskNames() string {
	names := make([]string, 0, len(riskPorts))
	for _, p := range riskPorts {
		names = append(names, riskRuleName(p))
	}
	return strings.Join(names, "','")
}

func riskMutationScript(block bool) string {
	if !block {
		return fmt.Sprintf(`$ErrorActionPreference='Stop'; Get-NetFirewallRule -ErrorAction Stop | Where-Object { $_.Name -in @('%s') } | Remove-NetFirewallRule -ErrorAction Stop`, riskNames())
	}
	ports := make([]string, 0, len(riskPorts))
	for _, p := range riskPorts {
		ports = append(ports, strconv.FormatUint(uint64(p), 10))
	}
	return fmt.Sprintf(`
$ErrorActionPreference='Stop'
$failures=@()
foreach($port in @(%s)) {
 $name='%s'+$port
 try {
  $r=Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue
  if($r) {
   Set-NetFirewallRule -Name $name -Direction Inbound -Action Block -Enabled True -Profile Any -LocalAddress Any -RemoteAddress Any -Program Any -Service Any -InterfaceType Any -ErrorAction Stop | Out-Null
   $r | Get-NetFirewallPortFilter | Set-NetFirewallPortFilter -Protocol TCP -LocalPort $port -RemotePort Any -ErrorAction Stop | Out-Null
  } else {
   New-NetFirewallRule -DisplayName $name -Name $name -Direction Inbound -Action Block -Protocol TCP -LocalPort $port -Profile Any -Enabled True -ErrorAction Stop | Out-Null
  }
 } catch { $failures += ($name+': '+$_.Exception.Message) }
}
if($failures.Count) { throw ($failures -join '; ') }
`, strings.Join(ports, ","), riskRulePrefix)
}

// GetRiskBlockStatus reports which preset high-risk ports are blocked by WinToolbox rules.
// Uses a single PowerShell probe for all ports to stay within LoadStatus timeout.
func GetRiskBlockStatus() RiskBlockStatus { return GetRiskBlockStatusContext(context.Background()) }

func GetRiskBlockStatusContext(ctx context.Context) RiskBlockStatus {
	return riskStatus(probeRiskBlocksContext(ctx))
}

func riskStatus(blockedMap map[uint32]bool) RiskBlockStatus {
	if len(blockedMap) != len(riskPorts) {
		return RiskBlockStatus{Unknown: true, Detail: "无法可靠读取高危端口规则"}
	}
	ports := make([]RiskPortInfo, 0, len(riskPorts))
	blocked := 0
	for _, p := range riskPorts {
		b := blockedMap[p]
		if b {
			blocked++
		}
		ports = append(ports, RiskPortInfo{
			Port:    p,
			Blocked: b,
			Name:    riskRuleName(p),
		})
	}
	st := RiskBlockStatus{Ports: ports}
	switch blocked {
	case 0:
		st.None = true
		st.Detail = "高危端口预设未启用"
	case len(riskPorts):
		st.AllBlocked = true
		st.Detail = fmt.Sprintf("已拦截 %d 个高危入站端口（135/139/445/5985/5986）", blocked)
	default:
		st.Partial = true
		st.Detail = fmt.Sprintf("部分拦截（%d/%d）", blocked, len(riskPorts))
	}
	return st
}

func probeRiskBlocks() map[uint32]bool { return probeRiskBlocksContext(context.Background()) }

func probeRiskBlocksContext(ctx context.Context) map[uint32]bool {
	out := make(map[uint32]bool, len(riskPorts))
	var names []string
	for _, p := range riskPorts {
		names = append(names, riskRuleName(p))
	}
	ps := fmt.Sprintf(`
$ErrorActionPreference='Stop'
$names=@('%s')
$rules=@(Get-NetFirewallRule -ErrorAction Stop)
foreach($name in $names){
  $r=@($rules | Where-Object Name -eq $name)
  if(-not $r){ Write-Output ($name + '=0'); continue }
  if($r.Count -ne 1){ throw ('duplicate rule: '+$name) }
  $r=$r[0]
  $p=$r|Get-NetFirewallPortFilter
  $en=($r.Enabled -eq $true -or [string]$r.Enabled -eq 'True')
  $blk=([string]$r.Action -eq 'Block' -or [int]$r.Action -eq 4)
  $local=[string]$p.LocalPort
  $want=$name.Substring($name.LastIndexOf('-')+1)
  $direction=([string]$r.Direction -eq 'Inbound' -or [int]$r.Direction -eq 1)
  $tcp=([string]$p.Protocol -eq 'TCP' -or [string]$p.Protocol -eq '6')
  $profiles=([string]$r.Profile -eq 'Any' -or [int]$r.Profile -eq 0)
  $portOk=$false
  foreach($part in ($local -split ',')){
    if($part.Trim() -eq $want){ $portOk=$true }
  }
  if($en -and $blk -and $portOk -and $direction -and $tcp -and $profiles){ Write-Output ($name + '=1') } else { Write-Output ($name + '=0') }
}
`, strings.Join(names, "','"))
	text, err := syscmd.RunPSQuickContext(ctx, ps)
	if err == nil && strings.TrimSpace(text) != "" {
		out = parseRiskResults(text)
		if len(out) == len(riskPorts) {
			return out
		}
	}

	// Fallback: parallel netsh probes (still bounded by RunQuick).
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(riskPorts))
	for _, p := range riskPorts {
		p := p
		go func() {
			defer wg.Done()
			blocked, known := readRiskBlockRuleNetshContext(ctx, p)
			mu.Lock()
			if known {
				out[p] = blocked
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func hasRiskBlockRule(port uint32) bool {
	return probeRiskBlocks()[port]
}

func hasRiskBlockRuleNetsh(port uint32) bool {
	blocked, _ := readRiskBlockRuleNetsh(port)
	return blocked
}

func readRiskBlockRuleNetsh(port uint32) (blocked, known bool) {
	return readRiskBlockRuleNetshContext(context.Background(), port)
}

func readRiskBlockRuleNetshContext(ctx context.Context, port uint32) (blocked, known bool) {
	name := riskRuleName(port)
	out, err := syscmd.RunQuickContext(ctx, "netsh", "advfirewall", "firewall", "show", "rule", "name="+name, "verbose")
	if err != nil {
		return false, false
	}
	if !strings.Contains(out, name) {
		return false, true
	}
	if !riskNetshScopeMatches(out) {
		return false, false // Unrecognized/incomplete scope must not confirm a block.
	}
	if !(strings.Contains(strings.ToLower(out), "block") || strings.Contains(out, "阻止")) {
		return false, true
	}
	return netshRuleEnabled(out) && netshRulePortMatches(out, strconv.FormatUint(uint64(port), 10)), true
}

func riskNetshScopeMatches(text string) bool {
	field := func(labels ...string) string {
		for _, line := range strings.Split(text, "\n") {
			parts := strings.SplitN(strings.ReplaceAll(line, "：", ":"), ":", 2)
			if len(parts) != 2 {
				continue
			}
			for _, label := range labels {
				if strings.EqualFold(strings.TrimSpace(parts[0]), label) {
					return strings.ToLower(strings.TrimSpace(parts[1]))
				}
			}
		}
		return ""
	}
	direction := field("Direction", "方向")
	protocol := field("Protocol", "协议", "通訊協定", "協定")
	profiles := field("Profiles", "配置文件", "設定檔")
	inbound := direction == "in" || direction == "inbound" || direction == "入站" || direction == "輸入"
	allProfiles := profiles == "any" || (strings.Contains(profiles, "domain") && strings.Contains(profiles, "private") && strings.Contains(profiles, "public")) || (strings.Contains(profiles, "域") && strings.Contains(profiles, "专用") && strings.Contains(profiles, "公用"))
	return inbound && (protocol == "tcp" || protocol == "6") && allProfiles
}

// RiskPortsList returns the preset port numbers for UI display.
func RiskPortsList() []uint32 {
	out := make([]uint32, len(riskPorts))
	copy(out, riskPorts)
	return out
}
