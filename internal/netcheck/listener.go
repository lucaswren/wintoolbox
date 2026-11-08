package netcheck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"wintoolbox/internal/win/syscmd"
)

func CheckLocalListener(ctx context.Context, port int) (string, error) {
	if port < 1 || port > 65535 {
		return "", errors.New("请输入 1 到 65535 之间的端口号。")
	}
	const script = `
$ErrorActionPreference = 'Stop'
$rows = @(Get-NetTCPConnection | Where-Object { $_.State -eq 'Listen' -and $_.LocalPort -eq %d })
if ($rows.Count -eq 0) { 'not-listening' }
elseif (@($rows | Where-Object { $_.LocalAddress -notlike '127.*' -and $_.LocalAddress -ne '::1' }).Count -eq 0) { 'loopback-only' }
else { 'listening' }
`
	out, err := syscmd.RunPSQuickContext(ctx, fmt.Sprintf(script, port))
	if err != nil {
		return "unavailable", nil
	}
	status := strings.TrimSpace(out)
	switch status {
	case "not-listening", "loopback-only", "listening":
		return status, nil
	default:
		return "unavailable", nil
	}
}
