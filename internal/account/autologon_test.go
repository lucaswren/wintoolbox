package account

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

type fakeStringValue struct {
	value string
	err   error
}

func (f fakeStringValue) GetStringValue(string) (string, uint32, error) {
	return f.value, registry.SZ, f.err
}

func TestAutoLogonReadErrorsAreUnknown(t *testing.T) {
	for _, tc := range []struct {
		reader           fakeStringValue
		enabled, unknown bool
	}{
		{fakeStringValue{err: registry.ErrNotExist}, false, false},
		{fakeStringValue{err: registry.ErrUnexpectedType}, false, true},
		{fakeStringValue{value: "1"}, true, false},
		{fakeStringValue{value: "0"}, false, false},
	} {
		st := autoLogonStatus(tc.reader)
		if st.Enabled != tc.enabled || st.Unknown != tc.unknown {
			t.Fatalf("reader %+v: %+v", tc.reader, st)
		}
	}
}
