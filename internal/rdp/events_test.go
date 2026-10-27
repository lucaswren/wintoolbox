package rdp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoginClassificationAndDuplicateChannels(t *testing.T) {
	events := []ServerEvent{
		{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4624, LogonType: "3", RecordID: "1"},
		{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4624, LogonType: "10", RecordID: "2"},
		{Channel: authChannel, Provider: "Microsoft-Windows-TerminalServices-RemoteConnectionManager", ID: 1149, RecordID: "1"},
		{Channel: sessionChannel, Provider: "Microsoft-Windows-TerminalServices-LocalSessionManager", ID: 21, RecordID: "1", Address: "LOCAL"},
		{Channel: "Security", Provider: "Other", ID: 4624, LogonType: "10", RecordID: "3"},
		{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4778, Session: "Console", RecordID: "4"},
		{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4778, Session: "RDP-Tcp#2", RecordID: "5"},
		{Channel: sessionChannel, Provider: "Microsoft-Windows-TerminalServices-LocalSessionManager", ID: 21, RecordID: "6"},
	}
	events = append(events, events[1])
	r := AnalyzeLogins(ServerEvents{Events: events})
	if r.Summary.Total != 8 || r.Summary.Logins != 1 || r.Summary.Reconnects != 1 {
		t.Fatalf("misclassified or merged unrelated channel IDs: %+v", r.Summary)
	}
	if !reflect.DeepEqual(r, AnalyzeLogins(r)) {
		t.Fatal("analysis is not idempotent")
	}
}

func TestLoginFailureWindowsAndReasons(t *testing.T) {
	start := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	var events []ServerEvent
	for i := 0; i < 5; i++ {
		events = append(events, ServerEvent{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4625, LogonType: "3", RecordID: fmt.Sprint(i), Address: "::ffff:192.0.2.1", Status: "0xc000006d", SubStatus: "0xC000006A", Time: start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano)})
	}
	events = append(events, ServerEvent{Channel: "Security", Provider: "Microsoft-Windows-Security-Auditing", ID: 4625, LogonType: "10", RecordID: "last", Address: "192.0.2.1", Time: start.Add(30 * time.Minute).Format(time.RFC3339Nano)})
	for i := 0; i < 7; i++ {
		e := events[0]
		e.RecordID = fmt.Sprint("missing", i)
		e.Address = "-"
		events = append(events, e)
	}
	r := AnalyzeLogins(ServerEvents{Events: events})
	if len(r.Findings) != 1 || r.Findings[0].Count != 5 || r.Findings[0].Address != "192.0.2.1" {
		t.Fatalf("not a sliding window or grouped missing sources: %+v", r.Findings)
	}
	if r.Summary.Failures != 1 || r.Summary.NetworkFailures != 12 || len(r.Sources) != 1 || r.Sources[0].NetworkFailures != 5 {
		t.Fatalf("network failures counted as RDP: %+v", r)
	}
	for _, e := range r.Events {
		if e.SubStatus != "" && e.Reason != "密码不正确" {
			t.Fatal(e.Reason)
		}
	}
	if failureReason(ServerEvent{Status: "0xc0000234", SubStatus: "0x0"}) != "账户已锁定" {
		t.Fatal("zero substatus must use status")
	}
}

func TestLoginSortUsesActualTime(t *testing.T) {
	r := AnalyzeLogins(ServerEvents{Events: []ServerEvent{{RecordID: "old", Time: "2026-10-27T01:00:00Z"}, {RecordID: "new", Time: "2026-10-27T09:00:00.1+08:00"}}})
	if r.Events[0].RecordID != "new" {
		t.Fatal("time zones/fractional timestamps sorted as text")
	}
	for _, args := range [][2]int{{0, 100}, {91, 100}, {7, 49}, {7, 1001}} {
		if _, err := ReadLoginReport(context.Background(), args[0], args[1]); err == nil {
			t.Fatal("invalid bounds accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadLoginReport(ctx, 7, 100); err == nil {
		t.Fatal("cancelled read continued")
	}
}

func TestLoginCollectorBoundaries(t *testing.T) {
	fixture := `$days=7;$limit=2
function Get-WinEvent {
 param($ListLog,$LogName,$FilterXPath,$MaxEvents,$ErrorAction,$ErrorVariable)
 if($testCase -eq 'all-denied' -or ($testCase -eq 'partial-denied' -and ($ListLog -eq 'Security' -or $LogName -eq 'Security'))){throw 'Injected access denied'}
 if($ListLog){return [pscustomobject]@{IsEnabled=($testCase -ne 'disabled')}}
 if($testCase -eq 'empty'){return}
 if($testCase -eq 'userdata' -and $LogName -ne 'Microsoft-Windows-TerminalServices-LocalSessionManager/Operational'){return}
 if($LogName -eq 'Security' -and $FilterXPath -notlike "*LogonType*10*3*"){throw 'Security filter is missing logon type restrictions'}
 foreach($i in 1..3){
  $event=[pscustomobject]@{RecordId=('900719925474099'+$i);TimeCreated=([datetime]'2026-10-27T01:00:00Z').AddSeconds($i);ProviderName='Microsoft-Windows-Security-Auditing';Id=4625;Message='Fixture message'}
  if($testCase -eq 'message-unavailable'){$event.PSObject.Properties.Remove('Message');$event | Add-Member ScriptProperty Message {throw 'Message resource missing'}}
  if($testCase -eq 'userdata'){
   $event.Id=21;$event.ProviderName='Microsoft-Windows-TerminalServices-LocalSessionManager'
   $event | Add-Member ScriptMethod ToXml {return '<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><UserData><EventXML><User>DOMAIN\alice</User><Address>192.0.2.1</Address><SessionID>22</SessionID></EventXML></UserData></Event>'}
  } else {
   $event | Add-Member ScriptMethod ToXml {return '<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><EventData><Data Name="TargetUserName">alice</Data><Data Name="TargetDomainName">DOMAIN</Data><Data Name="IpAddress">192.0.2.1</Data><Data Name="LogonType">3</Data><Data Name="SubStatus">0xc000006a</Data></EventData></Event>'}
  }
  $event
 }
}
`
	for _, scenario := range []string{"empty", "truncated", "partial-denied", "all-denied", "disabled", "message-unavailable", "userdata"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$testCase='"+scenario+"';"+fixture+eventsScript)
			out, err := cmd.CombinedOutput()
			if scenario == "all-denied" || scenario == "disabled" {
				if err == nil {
					t.Fatal("unavailable channels accepted", string(out))
				}
				return
			}
			if err != nil {
				t.Fatalf("%v %s", err, out)
			}
			var r ServerEvents
			if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &r); err != nil {
				t.Fatalf("%v %s", err, out)
			}
			if scenario == "empty" {
				if len(r.Events) != 0 || r.Truncated {
					t.Fatal(r)
				}
				return
			}
			if len(r.Events) != 2 || !r.Truncated || r.Events[0].User != "DOMAIN\\alice" || r.Events[0].RecordID != "9007199254740992" || len(r.Events[0].Data) == 0 {
				t.Fatalf("lost original fields: %+v", r)
			}
			if (scenario == "message-unavailable" || scenario == "partial-denied") && len(r.Warnings) == 0 {
				t.Fatal("missing partial-read warning")
			}
			if scenario == "partial-denied" && !r.Partial {
				t.Fatal("partial channel coverage must be explicit")
			}
		})
	}
}
