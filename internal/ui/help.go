package ui

import (
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func documentationURL(language string) string {
	const base = "https://github.com/lucaswren/wintoolbox/blob/main/docs/"
	switch language {
	case "zh-Hans":
		return base + "zh-Hans/README.md"
	case "zh-Hant":
		return base + "zh-Hant/README.md"
	default:
		return base + "README.md"
	}
}

func (a *App) OpenDocumentation(language string) error {
	if a.ctx == nil {
		return errors.New("application is not ready")
	}
	runtime.BrowserOpenURL(a.ctx, documentationURL(language))
	return nil
}
