package ui

import (
	"embed"
	"fmt"
	"wintoolbox/internal/locale"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Run starts the Wails desktop shell with embedded frontend assets.
func Run(assets embed.FS) error {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "WinToolBox",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 255},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			Theme:                windows.SystemDefault,
			Messages: &windows.Messages{
				InstallationRequired: locale.Text("This application requires WebView2. Attempting automatic installation…", "本程序需要 WebView2 运行时。正在尝试自动安装…", "本程式需要 WebView2 執行階段。正在嘗試自動安裝…"),
				UpdateRequired:       locale.Text("The WebView2 runtime needs to be updated.", "WebView2 运行时版本过低，需要更新。", "WebView2 執行階段版本過低，需要更新。"),
				MissingRequirements:  locale.Text("Missing runtime requirements", "缺少运行组件", "缺少執行元件"),
				Webview2NotInstalled: locale.Text("WebView2 runtime is not installed", "未安装 WebView2 运行时", "未安裝 WebView2 執行階段"),
				Error:                locale.Text("Error", "错误", "錯誤"),
				FailedToInstall:      locale.Text("WebView2 installation failed", "WebView2 安装失败", "WebView2 安裝失敗"),
				DownloadPage:         locale.Text("Visit the download page to install WebView2:", "请前往下载页面安装 WebView2：", "請前往下載頁面安裝 WebView2："),
				PressOKToInstall:     locale.Text("Press OK to start installation.", "按“确定”开始安装。", "按「確定」開始安裝。"),
				ContactAdmin:         locale.Text("Contact your administrator to install the WebView2 runtime.", "请联系管理员安装 WebView2 运行时。", "請聯絡系統管理員安裝 WebView2 執行階段。"),
				InvalidFixedWebview2: locale.Text("The specified WebView2 path is invalid.", "指定的 WebView2 路径无效。", "指定的 WebView2 路徑無效。"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("wails: %w", err)
	}
	return nil
}
