package ui

import (
	winapi "golang.org/x/sys/windows"
	"unsafe"
)

func fitWindow(workWidth, workHeight int) (width, height, minWidth, minHeight int) {
	width = min(1320, max(1, workWidth-32))
	height = min(840, max(1, workHeight-32))
	return width, height, min(960, width), min(600, height)
}
func initialWindow() (int, int, int, int) {
	var rect struct{ Left, Top, Right, Bottom int32 }
	user32 := winapi.NewLazySystemDLL("user32.dll")
	r, _, _ := user32.NewProc("SystemParametersInfoW").Call(0x30, 0, uintptr(unsafe.Pointer(&rect)), 0)
	if r == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return 1280, 800, 960, 600
	}
	dpi := uintptr(96)
	proc := user32.NewProc("GetDpiForSystem")
	if proc.Find() == nil {
		if value, _, _ := proc.Call(); value >= 96 {
			dpi = value
		}
	}
	return fitWindow(int(rect.Right-rect.Left)*96/int(dpi), int(rect.Bottom-rect.Top)*96/int(dpi))
}
