package locale

import "golang.org/x/sys/windows"

var getUserDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// System uses the user's Windows display language, not the machine locale.
func System() string {
	id, _, _ := getUserDefaultUILanguage.Call()
	return FromLanguageID(uint16(id))
}
