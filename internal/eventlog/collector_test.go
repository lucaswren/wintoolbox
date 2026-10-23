package eventlog

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPowerShellCollectorBoundaries(t *testing.T) {
	fixture := `$days=7; $limit=2
function Get-WinEvent {
 param($ListLog,$LogName,$FilterXPath,$MaxEvents,$ErrorAction,$ErrorVariable)
 if($testCase -eq 'denied'){throw 'Injected event log access denied'}
 if($ListLog){return [pscustomobject]@{IsEnabled=($testCase -ne 'disabled')}}
 if($testCase -eq 'empty'){return}
 foreach($i in 1..3){
  $event=[pscustomobject]@{RecordId=('900719925474099'+$i);TimeCreated=[datetime]'2026-10-23T01:00:00Z';ProviderName='Microsoft-Windows-Kernel-Power';Id=41;Level=1;MachineName='FIXTURE';Message='Unexpected shutdown fixture'}
  if($testCase -eq 'message-unavailable'){$event.PSObject.Properties.Remove('Message');$event | Add-Member ScriptProperty Message {throw 'Message resource missing'}}
  $event | Add-Member ScriptMethod ToXml {return '<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><EventData><Data Name="BugcheckCode">159</Data></EventData></Event>'}
  $event
 }
}
`
	for _, scenario := range []string{"empty", "truncated", "denied", "disabled", "message-unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$testCase='"+scenario+"';"+fixture+collector)
			out, err := cmd.CombinedOutput()
			if scenario == "denied" || scenario == "disabled" {
				if err == nil {
					t.Fatal("provider failure accepted", string(out))
				}
				return
			}
			if err != nil {
				t.Fatalf("%v %s", err, out)
			}
			var report Report
			if err = json.Unmarshal([]byte(strings.TrimSpace(string(out))), &report); err != nil {
				t.Fatalf("%v %s", err, out)
			}
			if scenario == "empty" && (len(report.Events) != 0 || report.Truncated) {
				t.Fatal(report)
			}
			if scenario == "message-unavailable" && (len(report.Events) != 2 || len(report.Warnings) == 0) {
				t.Fatal("message error discarded event evidence", report)
			}
			if scenario == "truncated" && (!report.Truncated || len(report.Events) != 2 || report.Events[0].Data[0].Name != "BugcheckCode" || report.Events[0].RecordID != "9007199254740991") {
				t.Fatal(report)
			}
		})
	}
}
