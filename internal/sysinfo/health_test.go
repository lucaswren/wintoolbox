package sysinfo

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestReadHealth(t *testing.T) {
	if os.Getenv("WINTOOLBOX_NETWORK_INTEGRATION") != "1" {
		t.Skip("requires Windows PowerShell and system log access")
	}
	health, err := ReadHealth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := time.Parse(time.RFC3339Nano, health.BootTime); err != nil {
		t.Fatalf("unexpected boot time: %q (%v)", health.BootTime, err)
	}
	if len(health.Services) != 6 {
		t.Fatalf("unexpected services: %+v", health.Services)
	}
	for _, event := range health.RecentEvents {
		if _, err := time.Parse(time.RFC3339Nano, event.Time); err != nil {
			t.Fatalf("unexpected event: %+v (%v)", event, err)
		}
	}
}
