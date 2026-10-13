package defender

import "testing"

func TestRealtimeValueRejectsUnknownAndMalformedOutput(t *testing.T) {
	for _, tc := range []struct {
		text         string
		value, known bool
	}{
		{"RT=True\r\n", true, true},
		{"RT=False\r\n", false, true},
		{"RT=1", true, true},
		{"RT=0", false, true},
		{"RT=?", false, false},
		{"RT=", false, false},
		{"RT=invalid", false, false},
		{"error", false, false},
	} {
		value, known := parseRealtimeValue(tc.text)
		if value != tc.value || known != tc.known {
			t.Fatalf("%q: value=%v known=%v", tc.text, value, known)
		}
	}
}
