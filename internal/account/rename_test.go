package account

import "testing"

func TestValidateAccountName(t *testing.T) {
	for _, name := range []string{"Alice", "新用户", "first last", "O'Neil", "abcdefghijklmnopqrst"} {
		if err := validateAccountName(name); err != nil {
			t.Errorf("valid %q: %v", name, err)
		}
	}
	for _, name := range []string{"", " ", "...", "user.", " user", "user ", "a/b", `a\b`, "a@b", "a\x00b", "a\nb", "abcdefghijklmnopqrstu", "😀😀😀😀😀😀😀😀😀😀a"} {
		if err := validateAccountName(name); err == nil {
			t.Errorf("accepted invalid %q", name)
		}
	}
}
