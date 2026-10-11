[English](../07-time-sync.md) · [繁體中文](../zh-Hant/07-time-sync.md) · [简体中文](../zh-Hans/07-time-sync.md)

# 07 · 時間同步

設置時區、NTP 源，立即對時與連通性測試。

## 第一步：下載

https://github.com/lucaswren/wintoolbox/releases/latest

攔截點 **保留 / 保持**。詳：[00 · 下載](./00-download.md)

![下載攔截示意（模擬）](../images/00-download.png)

## 介面標注

![時間同步（真實介面）](../images/07-time-sync.png)

| # | 按鈕 | 說明 |
|---|------|------|
| ① | **應用時區** | 應用下拉框所選時區 |
| ② | **保存 NTP** | 寫入 NTP 伺服器地址 |
| ③ | **立即同步** | 立刻向當前 NTP 對時 |
| ④ | **測試 NTP** | 檢測 NTP 是否可達 |

也可用「常用 NTP」芯片快速填入 Microsoft / 阿里雲 / 騰訊雲等。

打開本頁會盡量自動定位當前時區。狀態讀取失敗時會出現警告，稍後 **刷新** 再試。

相關：[文件首頁](./README.md) · [08 · 電源](./08-power.md)

[操作回饋與功能審查](operation-audit-2026-10-12.md)
