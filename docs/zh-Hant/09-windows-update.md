[English](../09-windows-update.md) | [繁體中文](../zh-Hant/09-windows-update.md) | [簡體中文](../zh-Hans/09-windows-update.md)

# 09 · 系統更新

一鍵暫停或恢復 Windows Update（恢復時優先還原關閉前的策略與服務設定，非整機快照）。

## 第一步：下載

https://github.com/lucaswren/wintoolbox/releases/latest

攔截點 **保留 / 保持**。詳：[00 · 下載](./00-download.md)

![下載攔截示意（模擬）](../images/00-download.png)

## 介面標注

![系統更新（真實介面）](../images/09-windows-update.png)

| # | 按鈕 | 說明 |
|---|------|------|
| ① | **關閉更新** | 暫停 Windows Update 相關策略/服務 |
| ② | **恢復更新** | 盡量還原關閉前設定 |

## 建議

- **常態建議保持更新開啟**；僅在臨時運維窗口關閉
- 關閉後記得用 ② 恢復
- 域/組策略環境可能無法被本工具完全覆蓋

相關：[90 · 系統加固指南](./90-hardening-guide.md) · [10 · 軟體更新](./10-app-update.md)
