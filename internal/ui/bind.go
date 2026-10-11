package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"wintoolbox/internal/account"
	"wintoolbox/internal/defender"
	"wintoolbox/internal/firewall"
	"wintoolbox/internal/harden"
	"wintoolbox/internal/locale"
	"wintoolbox/internal/power"
	"wintoolbox/internal/rdp"
	"wintoolbox/internal/selfupdate"
	"wintoolbox/internal/update"
	"wintoolbox/internal/win/port"
	"wintoolbox/internal/wintime"
)

// App exposes WinToolbox operations to the Wails frontend (window.go.ui.App).
type App struct {
	ctx context.Context
}

var resourceLocks sync.Map

func lockResource(resource string) func() {
	value, _ := resourceLocks.LoadOrStore(resource, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// NewApp creates the bound application API.
func NewApp() *App { return &App{} }

// Startup is called by Wails when the app starts.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Name:     AppName,
		Version:  AppVersion,
		Language: locale.System(),
	}
}

func (a *App) GetStatus(refresh bool) (Status, error) {
	return loadStatus(a.context(), "", refresh)
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) GetModuleStatus(module string, refresh bool) (Status, error) {
	switch module {
	case "overview", "account", "rdp", "firewall", "harden", "defender", "time", "update":
	default:
		return Status{}, fmt.Errorf("不支持的状态模块: %s", module)
	}
	return loadStatus(a.context(), module, refresh)
}

func (a *App) GetOverviewDetail() (OverviewDetail, error) {
	return LoadOverviewDetail()
}

func (a *App) GetTimeZones() ([]wintime.ZoneOption, error) {
	return LoadTimeZones()
}

func (a *App) GetFirewallRules() ([]firewall.RuleInfo, error) {
	return LoadFirewallRules()
}

func (a *App) ChangeAccountPassword(user, pass1, pass2 string) error {
	defer lockResource("account")()

	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("请选择用户")
	}
	if pass1 == "" || pass2 == "" {
		return fmt.Errorf("密码不能为空")
	}
	if pass1 != pass2 {
		return fmt.Errorf("两次输入的密码不一致")
	}
	return account.SetPassword(user, pass1)
}

func (a *App) SetAccountEnabled(user string, enabled bool) error {
	defer lockResource("account")()

	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("请选择用户")
	}
	return account.SetEnabled(user, enabled)
}

func (a *App) SetAccountAdmin(user string, admin bool) error {
	defer lockResource("account")()

	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("请选择用户")
	}
	return account.SetAdmin(user, admin)
}

func (a *App) DisableAccountLockout() error {
	defer lockResource("account")()

	return account.DisableLockout()
}

func (a *App) EnableAccountLockout() error {
	defer lockResource("account")()

	return account.EnableLockout()
}

func (a *App) SetAccountLockoutPolicy(threshold, durationMin, windowMin int) error {
	defer lockResource("account")()

	if threshold < 0 || durationMin < 0 || windowMin < 0 {
		return fmt.Errorf("锁定策略参数无效")
	}
	return account.SetLockoutPolicy(uint32(threshold), uint32(durationMin), uint32(windowMin))
}

func (a *App) DisableGuestAccount() error {
	defer lockResource("account")()

	return account.DisableGuest()
}

func (a *App) DisableAutoLogon() error {
	defer lockResource("account")()

	return account.DisableAutoLogon()
}

func (a *App) EnablePasswordPolicy(minLen int) error {
	defer lockResource("account")()

	if minLen < 0 || minLen > 128 {
		return fmt.Errorf("密码最短长度无效（0–128）")
	}
	return account.EnablePasswordPolicy(minLen)
}

func (a *App) ChangeRdpPort(portNum uint32) error {
	defer lockResource("network")()

	if err := port.ValidTCP(portNum); err != nil {
		return err
	}
	if err := rdp.ValidatePort(portNum); err != nil {
		return err
	}
	return rdp.SetPort(portNum)
}

func (a *App) ToggleRdp() error {
	defer lockResource("network")()

	st, err := rdp.GetStatus()
	if err != nil {
		return err
	}
	return rdp.SetEnabled(!st.Enabled)
}

func (a *App) SetRdpNLA(enabled bool) error {
	defer lockResource("network")()

	return rdp.SetNLA(enabled)
}

func (a *App) ClearRdpHistory() (string, error) {
	defer lockResource("history")()

	return rdp.ClearConnectionHistory()
}

func (a *App) ClearRdpHistoryByKind(kind string) (string, error) {
	defer lockResource("history")()

	return rdp.ClearConnectionHistoryByKind(kind)
}

func (a *App) GetRdpHistory() ([]rdp.HistoryEntry, error) {
	return rdp.ListConnectionHistory()
}

func (a *App) DeleteRdpHistoryEntry(kind, host, username, detail, sid string) error {
	defer lockResource("history")()

	return rdp.DeleteHistoryEntry(kind, host, username, detail, sid)
}

func (a *App) AllowFirewallPort(portNum uint32) error {
	defer lockResource("network")()

	if err := port.ValidTCP(portNum); err != nil {
		return err
	}
	return firewall.AllowTCP(portNum)
}

func (a *App) RemoveFirewallPort(portNum uint32) error {
	defer lockResource("network")()

	if err := port.ValidTCP(portNum); err != nil {
		return err
	}
	return firewall.RemoveAllowTCP(portNum)
}

func (a *App) ClearFirewallAllowRules() error {
	defer lockResource("network")()

	return firewall.ClearAllowRules()
}

func (a *App) SetFirewallEnabled(enabled bool) error {
	defer lockResource("network")()

	return firewall.SetAllProfiles(enabled)
}

func (a *App) DisablePing() error {
	defer lockResource("network")()

	return firewall.DisablePing()
}

func (a *App) EnablePing() error {
	defer lockResource("network")()

	return firewall.EnablePing()
}

func (a *App) BlockRiskPorts() error {
	defer lockResource("network")()

	st, err := rdp.GetStatus()
	if err != nil {
		return fmt.Errorf("无法确认远程桌面端口，已中止拦截: %w", err)
	}
	for _, p := range firewall.RiskPortsList() {
		if st.Enabled && st.Port == p {
			return fmt.Errorf("当前远程桌面端口 %d 位于拦截预设内，已中止以避免断开连接", p)
		}
	}
	return firewall.BlockRiskPorts()
}

func (a *App) UnblockRiskPorts() error {
	defer lockResource("network")()

	return firewall.UnblockRiskPorts()
}

func (a *App) DisableSMBv1() error {
	defer lockResource("network")()

	return harden.DisableSMBv1()
}

func (a *App) HardenWinRM() error {
	defer lockResource("network")()

	return harden.HardenWinRM()
}

func (a *App) RestrictAnonymous() error {
	defer lockResource("network")()

	return harden.RestrictAnonymous()
}

func (a *App) ApplyTimeZone(id string) error {
	defer lockResource("time")()

	return wintime.SetTimeZone(id)
}

func (a *App) SaveNTPServer(server string) error {
	defer lockResource("time")()

	return wintime.SetNTPServer(server)
}

func (a *App) SyncNTP() error {
	defer lockResource("time")()
	return wintime.SyncNTP()
}

func (a *App) TestNTPServer(server string) (string, error) {
	return wintime.TestNTPServer(server)
}

func (a *App) LockPC() error {
	defer lockResource("power")()
	return power.Lock()
}

func (a *App) RestartPC(delay int) error {
	defer lockResource("power")()

	if delay < 0 || delay > power.MaxPowerDelaySeconds {
		return fmt.Errorf("延迟秒数无效（须为 0–%d）", power.MaxPowerDelaySeconds)
	}
	return power.Restart(delay)
}

func (a *App) ShutdownPC(delay int) error {
	defer lockResource("power")()

	if delay < 0 || delay > power.MaxPowerDelaySeconds {
		return fmt.Errorf("延迟秒数无效（须为 0–%d）", power.MaxPowerDelaySeconds)
	}
	return power.Shutdown(delay)
}

func (a *App) AbortPower() error {
	defer lockResource("power")()
	return power.Abort()
}

func (a *App) DisableUpdate() error {
	defer lockResource("update")()
	return update.Disable()
}

func (a *App) EnableUpdate() error {
	defer lockResource("update")()
	return update.Enable()
}

func (a *App) DisableDefender() error {
	defer lockResource("defender")()
	return defender.Disable()
}

func (a *App) EnableDefender() error {
	defer lockResource("defender")()
	return defender.Enable()
}

func (a *App) CheckAppUpdate() (selfupdate.Info, error) {
	return selfupdate.Check(AppVersion)
}

func (a *App) DownloadAppUpdate() (selfupdate.Info, error) {
	return selfupdate.Download(AppVersion)
}

func (a *App) GetAppUpdateInfo() selfupdate.Info {
	return selfupdate.Cached()
}

func (a *App) ApplyAppUpdate() error {
	defer lockResource("selfupdate")()

	return selfupdate.Apply()
}

func (a *App) CancelAppUpdateDownload() { selfupdate.CancelDownload() }
