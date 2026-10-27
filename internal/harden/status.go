package harden

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"wintoolbox/internal/win/syscmd"
)

// Fit inside the UI's 15 second collector deadline, including command queueing.
const statusBudget = 12 * time.Second

//go:embed winrm-status.ps1
var winRMStatusScript string

var runStatusPS = syscmd.RunPSQuickContext
var readSMB1Registry = smb1RegistryState
var readWinRMService = winRMServiceState
var readAnonymousStatus = anonymousState

func GetStatus() Status { return GetStatusContext(context.Background()) }

// GetStatusContext keeps local results even when a PowerShell provider times out.
// Two parallel PowerShell reads replace the sequential per-rule/fallback chain.
func GetStatusContext(parent context.Context) Status {
	ctx, cancel := context.WithTimeout(parent, statusBudget)
	defer cancel()
	var result Status
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		result.Smb1Disabled, result.Smb1Unknown, result.Smb1Detail = smb1StateContext(ctx)
	}()
	go func() {
		defer wg.Done()
		result.WinRMHardened, result.WinRMUnknown, result.WinRMDetail = winrmStateContext(ctx)
	}()
	go func() {
		defer wg.Done()
		result.AnonymousOK, result.AnonymousUnknown, result.AnonymousDetail = readAnonymousStatus()
	}()
	wg.Wait()
	return result
}

func smb1State() (bool, bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), statusBudget)
	defer cancel()
	return smb1StateContext(ctx)
}

func smb1StateContext(ctx context.Context) (disabled, unknown bool, detail string) {
	out, err := runStatusPS(ctx, `$ErrorActionPreference='Stop'; $c=Get-SmbServerConfiguration; Write-Output ('SMB1=' + $c.EnableSMB1Protocol)`)
	if err == nil {
		switch strings.TrimSpace(out) {
		case "SMB1=False", "SMB1=false":
			return true, false, "SMBv1 协议已禁用"
		case "SMB1=True", "SMB1=true":
			return false, false, "SMBv1 协议仍启用"
		}
	}
	return readSMB1Registry()
}

func smb1RegistryState() (bool, bool, string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, smb1KeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, true, "SMBv1 状态未知"
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("SMB1")
	if err != nil {
		return false, true, "SMBv1 注册表未设置（状态未知）"
	}
	if v == 0 {
		return true, false, "SMBv1 注册表已禁用"
	}
	return false, false, "SMBv1 注册表仍启用"
}

func winrmState() (bool, bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), statusBudget)
	defer cancel()
	return winrmStateContext(ctx)
}

func winRMServiceState() (disabled, stopped bool, err error) {
	handle, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false, false, err
	}
	defer windows.CloseServiceHandle(handle)
	name, err := windows.UTF16PtrFromString(winRMService)
	if err != nil {
		return false, false, err
	}
	service, err := windows.OpenService(handle, name, windows.SERVICE_QUERY_CONFIG|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return false, false, err
	}
	s := &mgr.Service{Name: winRMService, Handle: service}
	defer s.Close()
	cfg, err := s.Config()
	if err != nil {
		return false, false, err
	}
	state, err := s.Query()
	return cfg.StartType == mgr.StartDisabled, state.State == svc.Stopped, err
}

func winrmStateContext(ctx context.Context) (hardened, unknown bool, detail string) {
	disabled, stopped, serviceErr := readWinRMService()
	service := "服务=未知"
	if serviceErr == nil {
		switch {
		case disabled:
			service = "服务=已禁用"
		case stopped:
			service = "服务=已停止"
		default:
			service = "服务=运行中"
		}
	}
	out, err := runStatusPS(ctx, winRMStatusScript)
	var blocks struct {
		Port5985 *bool `json:"port5985"`
		Port5986 *bool `json:"port5986"`
	}
	if err == nil {
		err = json.Unmarshal([]byte(strings.TrimSpace(out)), &blocks)
		if err == nil && (blocks.Port5985 == nil || blocks.Port5986 == nil) {
			err = fmt.Errorf("防火墙查询结果不完整")
		}
	}
	if err != nil {
		return false, true, service + " · 防火墙=未知（" + err.Error() + "）"
	}
	firewall := "防火墙=未拦截"
	if *blocks.Port5985 && *blocks.Port5986 {
		firewall = "防火墙=已拦 5985/5986"
	} else if *blocks.Port5985 || *blocks.Port5986 {
		firewall = "防火墙=部分拦截"
	}
	return serviceErr == nil && disabled && *blocks.Port5985 && *blocks.Port5986,
		serviceErr != nil, service + " · " + firewall
}
