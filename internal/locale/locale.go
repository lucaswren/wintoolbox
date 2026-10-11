// Package locale selects one of the supported application languages.
package locale

const (
	English            = "en"
	SimplifiedChinese  = "zh-CN"
	TraditionalChinese = "zh-TW"
)

// FromLanguageID maps Windows UI language IDs, with English as the fallback.
func FromLanguageID(id uint16) string {
	if id&0x3ff != 0x04 {
		return English
	}
	switch id >> 10 {
	case 1, 3, 5: // Taiwan, Hong Kong, Macao
		return TraditionalChinese
	case 2, 4: // Mainland China, Singapore
		return SimplifiedChinese
	default:
		return English
	}
}

// Text selects translated native setup text before WebView2 is available.
func Text(en, simplified, traditional string) string {
	switch System() {
	case SimplifiedChinese:
		return simplified
	case TraditionalChinese:
		return traditional
	default:
		return en
	}
}
