package eventlog

import (
	"strings"
	"testing"
)

func TestRepeatedClockAndLogClearNoticesKeepAllEvidence(t *testing.T) {
	r := Analyze(Report{Warnings: []string{"Collector notice", "Collector notice"}, Events: []Event{
		{RecordID: "1", Provider: "Microsoft-Windows-Kernel-General", ID: 1},
		{RecordID: "2", Provider: "Microsoft-Windows-Kernel-General", ID: 1},
		{RecordID: "3", Provider: "Microsoft-Windows-Eventlog", ID: 104},
	}})
	if len(r.Events) != 3 || r.Summary.Total != 3 || len(r.Warnings) != 4 {
		t.Fatalf("lost evidence or repeated notices: %+v", r)
	}
	again := Analyze(r)
	if len(again.Warnings) != len(r.Warnings) || len(again.Events) != 3 {
		t.Fatal("reanalyzing a report duplicates notices or drops events")
	}
}

func TestRebootEvidenceIsProviderSpecificAndNotRootCause(t *testing.T) {
	r := Analyze(Report{Truncated: true, Events: []Event{
		{RecordID: "1", Time: "2026-10-23T01:00:00Z", Provider: "Microsoft-Windows-Kernel-Power", ID: 41, Data: []Field{{Name: "BugcheckCode", Value: "159"}}},
		{RecordID: "2", Time: "2026-10-23T01:00:10Z", Provider: "EventLog", ID: 6008},
		{RecordID: "3", Time: "2026-10-23T01:00:20Z", Provider: "Microsoft-Windows-WER-SystemErrorReporting", ID: 1001},
		{RecordID: "4", Time: "2026-10-22T23:00:00Z", Provider: "User32", ID: 1074},
		{RecordID: "5", Time: "2026-10-23T01:00:30Z", Provider: "UnrelatedProvider", ID: 41},
	}})
	if r.Summary.Unexpected != 2 || r.Summary.Bugchecks != 1 || r.Summary.Requests != 1 {
		t.Fatalf("%+v", r.Summary)
	}
	if len(r.Findings) != 4 {
		t.Fatal(r.Findings)
	}
	for _, f := range r.Findings {
		if f.RecordID == "1" && (f.StopCode != "0x0000009F" || !strings.Contains(f.Conclusion, "不能确定")) {
			t.Fatal(f)
		}
		if f.RecordID == "4" && len(f.Evidence) != 1 {
			t.Fatal("unrelated distant events correlated", f)
		}
	}
	if len(r.Warnings) < 2 {
		t.Fatal("truncation and counting limitations missing")
	}
}
func TestMissingCrashCodeDoesNotImplyPowerFailure(t *testing.T) {
	r := Analyze(Report{Events: []Event{{RecordID: "1", Time: "bad-time", Provider: "Microsoft-Windows-Kernel-Power", ID: 41}}})
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Conclusion, "不能确定") || r.Findings[0].StopCode != "" {
		t.Fatal(r)
	}
	if c, _ := classify(Event{Provider: "Application Error", ID: 1001}); c == "bugcheck" {
		t.Fatal("unrelated event ID classified as bugcheck")
	}
	if len(Analyze(Report{}).Events) != 0 {
		t.Fatal("empty report")
	}
}
