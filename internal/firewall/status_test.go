package firewall

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRemovalRequiresAllRulesKnownAbsent(t *testing.T) {
	if riskRulesAbsent(nil) {
		t.Fatal("unknown rules were treated as removed")
	}
	if riskRulesAbsent(map[uint32]bool{135: false}) {
		t.Fatal("partial absence was treated as removed")
	}
	all := map[uint32]bool{135: false, 139: false, 445: false, 5985: false, 5986: false}
	if !riskRulesAbsent(all) {
		t.Fatal("complete absence was not accepted")
	}
	all[445] = true
	if riskRulesAbsent(all) {
		t.Fatal("an existing disabled rule was treated as removed")
	}
	malformed := parseRiskResults(riskRuleName(135) + "=error")
	if len(malformed) != 0 {
		t.Fatal("malformed result became known state")
	}
}

func TestRiskScopeRejectsWrongDirectionProtocolAndProfiles(t *testing.T) {
	valid := "Direction: In\nProtocol: TCP\nProfiles: Domain,Private,Public\n"
	if !riskNetshScopeMatches(valid) {
		t.Fatal("valid scope rejected")
	}
	for _, invalid := range []string{strings.Replace(valid, "In", "Out", 1), strings.Replace(valid, "TCP", "UDP", 1), strings.Replace(valid, "Domain,Private,Public", "Public", 1), "Protocol: TCP"} {
		if riskNetshScopeMatches(invalid) {
			t.Fatal("invalid scope accepted")
		}
	}
}

func TestRiskScriptsParseWithoutExecutingWrites(t *testing.T) {
	for _, block := range []bool{true, false} {
		script := riskMutationScript(block)
		// Parsing is read-only; no firewall commands from the script are executed.
		escaped := strings.ReplaceAll(script, "'", "''")
		parse := "$e=$null; $t=$null; [System.Management.Automation.Language.Parser]::ParseInput('" + escaped + "',[ref]$t,[ref]$e) | Out-Null; if($e.Count){$e | Out-String; exit 1}"
		if out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", parse).CombinedOutput(); err != nil {
			t.Fatalf("invalid PowerShell: %s %v", out, err)
		}
	}
}

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
