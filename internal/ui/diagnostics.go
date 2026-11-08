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

func (a *App) CheckNetwork(target string, port int) (netcheck.Basic, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return netcheck.Basic{}, errors.New("已有网络检查正在运行。")
	}
	ctx, cancel := context.WithTimeout(a.context(), 10*time.Second)
	defer cancel()
	return netcheck.Check(ctx, target, port)
}

func (a *App) TraceNetwork(target string) (netcheck.Route, error) {
	select {
	case networkProbeSlot <- struct{}{}:
		defer func() { <-networkProbeSlot }()
	default:
		return netcheck.Route{}, errors.New("已有网络检查正在运行。")
	}
	ctx, cancel := context.WithTimeout(a.context(), 50*time.Second)
	defer cancel()
	return netcheck.Trace(ctx, target)
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
