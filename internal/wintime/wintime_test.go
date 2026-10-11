package wintime

import "testing"

func TestNTPResponseMustContainSuccessfulSample(t *testing.T) {
	for _, tc := range []struct {
		text  string
		valid bool
	}{
		{"12:00:01, +00.0012345s", true},
		{"12:00:01, -00.0012345s", true},
		{"12:00:01, error: 0x800705B4", false},
		{"tracking server", false},
		{"", false},
	} {
		if validNTPResponse(tc.text) != tc.valid {
			t.Fatalf("unexpected sample result: %s", tc.text)
		}
	}
}
