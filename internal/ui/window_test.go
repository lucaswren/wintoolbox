package ui

import "testing"

func TestDefaultWindowFitsRDPWorkArea(t *testing.T) {
	for _, size := range [][2]int{{1920, 1040}, {1366, 728}, {1024, 728}, {800, 560}} {
		w, h, mw, mh := fitWindow(size[0], size[1])
		if w > size[0] || h > size[1] || mw > w || mh > h || w <= 0 || h <= 0 {
			t.Fatalf("%v -> %d %d %d %d", size, w, h, mw, mh)
		}
	}
	for _, dpi := range []int{120, 144, 192} {
		w, h, _, _ := fitWindow(1920*96/dpi, 1040*96/dpi)
		if w*dpi/96 > 1920 || h*dpi/96 > 1040 {
			t.Fatalf("DPI %d window exceeds work area", dpi)
		}
	}
}
