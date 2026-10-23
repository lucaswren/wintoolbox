package account

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func validateAccountName(name string) error {
	if name == "" || len(utf16.Encode([]rune(name))) > 20 || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") || strings.Trim(name, ". ") == "" {
		return fmt.Errorf("新用户名必须为 1–20 个字符，不能以空白或句点结尾")
	}
	for _, r := range name {
		if unicode.IsControl(r) || strings.ContainsRune(`"/\[]:;|=,+*?<>@`, r) {
			return fmt.Errorf("新用户名包含不允许的字符")
		}
	}
	return nil
}

// Rename changes only the local SAM account name, preserving the SID and profile path.
func Rename(username, newName string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("请选择用户")
	}
	if err := validateAccountName(newName); err != nil {
		return err
	}
	if username == newName {
		return fmt.Errorf("新用户名与原用户名相同")
	}
	// NetUserGetInfo with a nil server limits this operation to local accounts.
	if _, err := Get(username); err != nil {
		return err
	}
	auto := GetAutoLogonStatus()
	if auto.Unknown || auto.Enabled {
		return fmt.Errorf("请先关闭自动登录并刷新状态，再修改用户名")
	}
	machine, err := os.Hostname()
	if err != nil {
		return err
	}
	sid, _, _, err := windows.LookupSID("", machine+`\`+username)
	if err != nil {
		return fmt.Errorf("读取账户标识失败: %w", err)
	}
	oldPtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return err
	}
	newPtr, err := windows.UTF16PtrFromString(newName)
	if err != nil {
		return err
	}
	payload := userInfo0{Name: newPtr}
	code, _, _ := procNetUserSetInfo.Call(0, uintptr(unsafe.Pointer(oldPtr)), 0, uintptr(unsafe.Pointer(&payload)), 0)
	if code == 2224 || code == 2223 || code == 1379 {
		return fmt.Errorf("新用户名已被使用")
	}
	if code != 0 {
		return fmt.Errorf("%s", mapNetError(code))
	}
	actual, _, _, err := windows.LookupSID("", machine+`\`+newName)
	if err != nil || actual.String() != sid.String() {
		return fmt.Errorf("用户名可能已修改，但账户标识复查失败，请刷新账户列表")
	}
	return nil
}
