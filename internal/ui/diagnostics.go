package ui

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"wintoolbox/internal/netcheck"
	"wintoolbox/internal/sysinfo"
)

var networkProbeSlot = make(chan struct{}, 1)

func (a *App) CheckNetwork(target string, port, count int, sourceIP string) (netcheck.Basic, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return netcheck.Basic{}, errors.New("已有网络检查正在运行。")
	}
	source, err := netcheck.ParseSource(sourceIP)
	if err != nil {
		return netcheck.Basic{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 40*time.Second)
	defer cancel()
	return netcheck.Check(ctx, target, port, count, source)
}

func (a *App) TraceNetwork(target string, rounds int, sourceIP string) (netcheck.Route, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return netcheck.Route{}, errors.New("已有网络检查正在运行。")
	}
	ctx, cancel := context.WithTimeout(a.context(), 120*time.Second)
	a.traceMu.Lock()
	a.traceCancel = cancel
	a.traceMu.Unlock()
	defer func() {
		cancel()
		a.traceMu.Lock()
		a.traceCancel = nil
		a.traceMu.Unlock()
	}()
	source, err := netcheck.ParseSource(sourceIP)
	if err != nil {
		return netcheck.Route{}, err
	}
	return netcheck.Trace(ctx, target, rounds, source)
}

func (a *App) StopNetworkTrace() bool {
	a.traceMu.Lock()
	defer a.traceMu.Unlock()
	if a.traceCancel == nil {
		return false
	}
	a.traceCancel()
	return true
}

func (a *App) GetNetworkSources() ([]netcheck.LocalSource, error) {
	return netcheck.LocalSources()
}

func (a *App) CheckPublicEgress(sourceIP string) ([]netcheck.Egress, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return nil, errors.New("已有网络检查正在运行。")
	}
	source, err := netcheck.ParseSource(sourceIP)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 10*time.Second)
	defer cancel()
	return netcheck.LookupEgress(ctx, source), nil
}

func (a *App) CheckLocalListener(port int) (string, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return "", errors.New("已有网络检查正在运行。")
	}
	return netcheck.CheckLocalListener(a.context(), port)
}

func (a *App) GetSystemHealth() (sysinfo.Health, error) {
	return sysinfo.ReadHealth(a.context())
}

func (a *App) CopyHealthSummary(text string) error {
	if a.ctx == nil || len(text) == 0 || len(text) > 16*1024 {
		return errors.New("无法复制故障摘要。")
	}
	return runtime.ClipboardSetText(a.ctx, text)
}
