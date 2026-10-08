package firewall

import "testing"

func TestMissingProbeResultsStayUnknown(t *testing.T) {
	if st := riskStatus(nil); !st.Unknown || st.None {
		t.Fatal("failed probe was interpreted as disabled")
	}
	if st := riskStatus(map[uint32]bool{135: true}); !st.Unknown {
		t.Fatal("incomplete probe was interpreted as complete")
	}
	complete := map[uint32]bool{135: true, 139: false, 445: false, 5985: false, 5986: false}
	if st := riskStatus(complete); st.Unknown || !st.Partial {
		t.Fatal("known partial state lost")
	}
	if mode := (PingBlockStatus{Unknown: true, IPv4: true}).Mode(); mode != "unknown" {
		t.Fatal(mode)
	}
}
