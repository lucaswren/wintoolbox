package ui

import (
	"fmt"
	"testing"
	"wintoolbox/internal/rdp"
)

func TestWinRMHardeningCannotBlockCustomRdpPort(t *testing.T) {
	for _, port := range []uint32{5985, 5986, 47821} {
		writes := 0
		err := hardenWinRMChecked(func() (rdp.Status, error) { return rdp.Status{Enabled: true, Port: port}, nil }, func() error { writes++; return nil })
		if port == 47821 {
			if err != nil || writes != 1 {
				t.Fatal("unrelated RDP port should not prevent hardening", err, writes)
			}
		} else if err == nil || writes != 0 {
			t.Fatal("WinRM hardening could block RDP", port, err, writes)
		}
	}
	writes := 0
	err := hardenWinRMChecked(func() (rdp.Status, error) { return rdp.Status{}, fmt.Errorf("registry unavailable") }, func() error { writes++; return nil })
	if err == nil || writes != 0 {
		t.Fatal("unknown RDP state allowed a firewall mutation")
	}
}
