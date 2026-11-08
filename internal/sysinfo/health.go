package sysinfo

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"wintoolbox/internal/win/syscmd"
)

//go:embed health.ps1
var healthScript string

type Health struct {
	BootTime      string          `json:"bootTime"`
	Services      []ServiceHealth `json:"services"`
	RecentEvents  []HealthEvent   `json:"recentEvents"`
	EventsPartial bool            `json:"eventsPartial"`
}

type ServiceHealth struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type HealthEvent struct {
	ID   int    `json:"id"`
	Time string `json:"time"`
}

func ReadHealth(ctx context.Context) (Health, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := syscmd.RunPSQuickContext(ctx, healthScript)
	if err != nil {
		return Health{}, err
	}
	var result Health
	err = json.Unmarshal([]byte(strings.TrimSpace(out)), &result)
	return result, err
}
