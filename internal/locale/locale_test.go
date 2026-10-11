package locale

import "testing"

func TestFromLanguageID(t *testing.T) {
	for _, tc := range []struct {
		id   uint16
		want string
	}{
		{0x0409, English}, {0x0809, English}, {0x040c, English}, {0, English},
		{0x0804, SimplifiedChinese}, {0x1004, SimplifiedChinese},
		{0x0404, TraditionalChinese}, {0x0c04, TraditionalChinese}, {0x1404, TraditionalChinese},
	} {
		if got := FromLanguageID(tc.id); got != tc.want {
			t.Errorf("%04x: got %s, want %s", tc.id, got, tc.want)
		}
	}
}
