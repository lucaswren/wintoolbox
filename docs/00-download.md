[English](00-download.md) · [繁體中文](zh-Hant/00-download.md) · [简体中文](zh-Hans/00-download.md)

# 00 · Download and first launch

1. Download the Windows amd64 executable from [GitHub Releases](https://github.com/lucaswren/wintoolbox/releases/latest).
2. If your browser blocks the download, select **Keep** only after confirming its source.
3. Compare the SHA256 hash with the release checksum:

```powershell
Get-FileHash .\WinToolBox-v1.1.2-windows-amd64.exe -Algorithm SHA256
```

4. Launch the executable and accept Windows UAC elevation. If elevation fails, right-click and select **Run as administrator**.

The executable is currently unsigned. Windows SmartScreen may require **More info → Run anyway** after you verify the source and checksum.

WinToolBox runs directly; there is no separate application installation wizard. If WebView2 is missing, the first launch downloads the Microsoft bootstrapper, verifies its digital signature and installs the runtime with a progress window. Internet access is required for this step. If automatic installation fails, install WebView2 manually and launch WinToolBox again.

Setup follows your Windows display language: English, Simplified Chinese or Traditional Chinese. Other languages use English. In the application, choose **English**, **繁體中文** or **简体中文** from the sidebar; this choice is remembered on subsequent launches. The native runtime setup follows Windows independently of this saved application preference.

![Illustrative browser download prompt](images/00-download.png)
![Illustrative keep-download prompt](images/00-download-keep.png)

Screenshots show the earlier Simplified Chinese interface; the labels above describe the English interface. Download prompts are illustrations.

[Operation feedback and audit](operation-audit-2026-10-12.md)
