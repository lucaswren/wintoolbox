[English](../00-download.md) · [繁體中文](../zh-Hant/00-download.md) · [简体中文](../zh-Hans/00-download.md)

# 00 · 下载 WinToolBox

## 下载与运行

1. 从 [GitHub 发布页](https://github.com/lucaswren/wintoolbox/releases/latest)下载 Windows amd64 程序及同名 SHA256 校验文件。
2. 在 PowerShell 中运行下方命令，把示例文件名换成实际下载的文件名。确认结果与发布页提供的校验值一致。
3. 双击程序，在 Windows 权限提示中确认。遇到 SmartScreen 时，先确认下载来源和校验值，再选择 **更多信息 → 仍要运行**。

```powershell
Get-FileHash .\WinToolBox-v1.2.0-windows-amd64.exe -Algorithm SHA256
```

首次启动时，如果电脑缺少 Microsoft WebView2，工具会联网下载并安装。进入界面后，可在左上角切换语言，工具会记住你的选择。

[操作反馈与功能审查](operation-audit-2026-10-12.md)
