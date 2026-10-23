package rdp

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Session struct {
	ID     uint32 `json:"id"`
	Remote bool   `json:"remote"`
	Known  bool   `json:"known"`
}

// Query the process session, not SESSIONNAME (which can be stale or inherited).
func CurrentSession() Session {
	var s Session
	if windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &s.ID) != nil || s.ID == 0 {
		return s
	}
	dll := windows.NewLazySystemDLL("wtsapi32.dll")
	var buffer *uint16
	var size uint32
	r, _, _ := dll.NewProc("WTSQuerySessionInformationW").Call(0, uintptr(s.ID), 16, uintptr(unsafe.Pointer(&buffer)), uintptr(unsafe.Pointer(&size)))
	if r == 0 || buffer == nil {
		return s
	}
	defer dll.NewProc("WTSFreeMemory").Call(uintptr(unsafe.Pointer(buffer)))
	if size < 2 {
		return s
	}
	protocol := *buffer
	s.Known = protocol == 0 || protocol == 2
	s.Remote = protocol == 2
	return s
}

func RequireConsole() error {
	s := CurrentSession()
	if !s.Known || s.Remote {
		return fmt.Errorf("此操作会影响远程连接，仅允许在已确认的本地控制台执行")
	}
	return nil
}
