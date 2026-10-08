package update

import "testing"

func TestCorruptSnapshotRejectedBeforeRestore(t *testing.T) {
	for _, bad := range []map[string]uint32{{"HadSvc_wuauserv": 2}, {"Svc_wuauserv": 99}, {"Task_0": 99}} {
		if validateSnapshot(bad) == nil {
			t.Fatalf("accepted corrupt backup: %v", bad)
		}
	}
	if err := validateSnapshot(map[string]uint32{"HadNoAutoUpdate": 1, "NoAutoUpdate": 1, "Svc_wuauserv": 4, "Task_0": 2}); err != nil {
		t.Fatal(err)
	}
}
