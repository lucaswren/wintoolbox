[English](../00-download.md) · [繁體中文](../zh-Hant/00-download.md) · [简体中文](../zh-Hans/00-download.md)

# 00 · 下载 WinToolBox

所有操作文档的**第一步**。

## 下载链接

- 最新版：https://github.com/lucaswren/wintoolbox/releases/latest
- 全部版本：https://github.com/lucaswren/wintoolbox/releases

下载 **`WinToolBox-v*.exe`**（可选同名 `.sha256`）。

## 拦截时点「保留 / 保持」

未签名 exe **常被拦截**，不等于一定有毒。

**Edge / Chrome**：已阻止 → **…** → **保留 / 保持**（Keep）→ 必要时 **仍要保留**。

**SmartScreen**：更多信息 → **仍要运行**。

![下载拦截示意（模拟）](../images/00-download.png)

## 校验（可选）

```powershell
Get-FileHash .\WinToolBox-v1.1.2-windows-amd64.exe -Algorithm SHA256
```

与 Release / `.sha256` 对比。

## 运行

**以管理员身份运行**（或双击后在 UAC 允许）。

返回：[文档首页](./README.md)

[操作反馈与功能审查](operation-audit-2026-10-12.md)
