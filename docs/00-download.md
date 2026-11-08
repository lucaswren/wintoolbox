[English](00-download.md) · [繁體中文](zh-Hant/00-download.md) · [简体中文](zh-Hans/00-download.md)

# 00 · Download and first launch

## Download and run

1. Download the Windows amd64 executable and its matching SHA256 file from [GitHub Releases](https://github.com/lucaswren/wintoolbox/releases/latest).
2. In PowerShell, run the command below using the downloaded file name. Compare the result with the checksum on the release page before opening the file.
3. Open the executable and accept the Windows administrator prompt. If Windows SmartScreen prompts you, check the source and checksum before selecting **More info → Run anyway**.

```powershell
Get-FileHash .\WinToolBox-v1.2.0-windows-amd64.exe -Algorithm SHA256
```

On first launch, WinToolBox may download and install the Microsoft WebView2 runtime if it is missing. This requires internet access. Use the language selector at the top left to change the app language.

[Operation feedback and audit](operation-audit-2026-10-12.md)
