[English](../00-download.md) · [繁體中文](../zh-Hant/00-download.md) · [简体中文](../zh-Hans/00-download.md)

# 00 · 下載 WinToolBox

## 下載與執行

1. 從 [GitHub 發行頁](https://github.com/lucaswren/wintoolbox/releases/latest)下載 Windows amd64 程式及同名 SHA256 校驗檔。
2. 在 PowerShell 執行下方命令，把範例檔名換成實際下載的檔名。確認結果與發行頁提供的校驗值一致。
3. 按兩下程式，在 Windows 權限提示中確認。遇到 SmartScreen 時，先確認下載來源和校驗值，再選擇 **其他資訊 → 仍要執行**。

```powershell
Get-FileHash .\WinToolBox-v1.2.0-windows-amd64.exe -Algorithm SHA256
```

首次啟動時，如果電腦缺少 Microsoft WebView2，工具會連線下載並安裝。進入介面後，可在左上角切換語言，工具會記住你的選擇。

[操作回饋與功能審查](operation-audit-2026-10-12.md)
