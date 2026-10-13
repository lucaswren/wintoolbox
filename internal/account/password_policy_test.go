package account

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestPasswordComplexityINFEncodings(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		flag := "0"
		if enabled {
			flag = "1"
		}
		text := "[Unicode]\r\nUnicode=yes\r\n[System Access]\r\nPasswordComplexity = " + flag + "\r\n"
		inputs := [][]byte{[]byte(text), []byte("\ufeff" + text)}
		for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			units := append([]uint16{0xfeff}, utf16.Encode([]rune(text))...)
			data := make([]byte, len(units)*2)
			for i, unit := range units {
				order.PutUint16(data[i*2:], unit)
			}
			inputs = append(inputs, data)
		}
		for _, data := range inputs {
			if got, known := parsePasswordComplexity(data); !known || got != enabled {
				t.Fatalf("encoding %x: enabled=%v known=%v", data[:4], got, known)
			}
		}
	}
	for _, data := range [][]byte{[]byte("PasswordComplexityExtra=1"), []byte("PasswordComplexity=invalid"), {0xff, 0xfe, 0x50}} {
		if _, known := parsePasswordComplexity(data); known {
			t.Fatalf("invalid INF accepted: %q", data)
		}
	}
}
