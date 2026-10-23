[English](../00-download.md) · [繁體中文](../zh-Hant/00-download.md) · [简体中文](../zh-Hans/00-download.md)

# 00 · 下載 WinToolBox

所有操作文件的**第一步**。

## 下載鏈接

- 最新版：https://github.com/lucaswren/wintoolbox/releases/latest
- 全部版本：https://github.com/lucaswren/wintoolbox/releases

下載 **`WinToolBox-v*.exe`**（可選同名 `.sha256`）。

## 攔截時點「保留 / 保持」

未簽名 exe **常被攔截**，不等于一定有毒。

**Edge / Chrome**：已阻止 → **…** → **保留 / 保持**（Keep）→ 必要時 **仍要保留**。

**SmartScreen**：更多資訊 → **仍要執行**。

![下載攔截示意（模擬）](../images/00-download.png)

## 校驗（可選）

```powershell
Get-FileHash .\WinToolBox-v1.1.0-windows-amd64.exe -Algorithm SHA256
```

與 Release / `.sha256` 對比。

## 執行

**以系統管理員身份執行**（或雙擊後在 UAC 允許）。

返回：[文件首頁](./README.md)

[操作回饋與功能審查](operation-audit-2026-10-12.md)
