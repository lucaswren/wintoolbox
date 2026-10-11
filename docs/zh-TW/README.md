[English](../README.md) | [繁體中文](../zh-TW/README.md) | [簡體中文](../zh-CN/README.md)

# WinToolBox 文件

操作說明。配圖中紅色數字與文中標注對應。
**下載圖為模擬示意**；軟體畫面為 **2× 高清真實介面截圖**。

## 命名規則

`序號-英文短名.md` / `序號-英文短名.png`，與側欄功能一一對應；`90-` 為綜合指南。

## 第一步：下載

https://github.com/lucaswren/wintoolbox/releases/latest

攔截時點 **保留 / 保持**；SmartScreen：**更多資訊 → 仍要執行**。詳見 [00 · 下載](./00-download.md)。

![下載攔截示意（模擬）](../images/00-download.png)

## 功能文件

| 文件 | 說明 |
|------|------|
| [00 · 下載](./00-download.md) | 下載、攔截、校驗、系統管理員執行 |
| [01 · 本機概覽](./01-overview.md) | 系統與硬體資訊 |
| [02 · 本地帳戶](./02-local-account.md) | 來賓/自動登錄、密碼策略、鎖定、改密 |
| [03 · 遠端桌面](./03-remote-desktop.md) | 連接埠、開關、NLA、連線記錄 |
| [04 · 防火牆](./04-firewall.md) | 設定檔案、放行、禁 ping、高危連接埠 |
| [05 · 安全加固](./05-security-harden.md) | SMBv1、WinRM、匿名枚舉 |
| [06 · 防病毒](./06-defender.md) | Defender 即時防護開關 |
| [07 · 時間同步](./07-time-sync.md) | 時區、NTP、對時與測試 |
| [08 · 電源](./08-power.md) | 鎖定 / 重新啟動 / 關機 / 取消 |
| [09 · 系統更新](./09-windows-update.md) | Windows Update 開關 |
| [10 · 軟體更新](./10-app-update.md) | 本軟體檢查、下載、安裝 |
| [90 · 系統加固指南](./90-hardening-guide.md) | 推薦順序：帳戶 → RDP → 防火牆 → 加固 → 更新 |

需**系統管理員權限**。域環境可能被組策略覆蓋。

[語言選擇](localization.md)
