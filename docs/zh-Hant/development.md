[English](../development.md) | [繁體中文](../zh-Hant/development.md) | [簡體中文](../zh-Hans/development.md)

# 開發和驗收規范

環境：Windows amd64、Go 1.27+、Node.js 22.13+、Windows PowerShell 5.1 或 PowerShell 7。

## 檢查流程

```powershell
go run ./scripts/genbindings
cd frontend
npm ci
npm run check
npm run build
cd ..
go test ./...
go vet ./...
./scripts/build.ps1 -SkipInstall
```

後端導出方法、參數或 JSON DTO 有變化時，重新生成 `frontend/src/api/wails.d.ts`。CI 使用 `-check` 驗證契約未過期；前端 API 門面的類型檢查會檢測方法名、參數數量和返回類型。當前未將所有 Vue 頁面遷移到 TypeScript。

前端統一 Prettier 格式及 ESLint Vue 規則，Go 使用 gofmt。依賴以 package-lock.json 固定，CI 與預設構建使用 npm ci。構建產物寫入 artifacts/releases，根目錄 WinToolBox.exe 作為本地執行檔案保留。

## 操作契約

- 狀態探測失敗表示 unknown；不能使用布爾零值替代未知。
- 所有設定備份字段寫入成功後才設置 Saved；恢復前先讀取全部所需字段。
- 部分操作成功後仍可能返回錯誤，因此前端無論成功失敗都復查相關模塊。
- 後端按資源序列化寫操作；RDP、防火牆、WinRM 共用網路資源鎖。
- 狀態查詢按模塊合並重疊任務。取消感知的模塊在全部調用者離開後取消共享命令；不能取消的原生調用繼續完成，但同一探測不會重復啟動。系統命令並發最多6個。
- Windows Update 的“還原關閉前設定”以原設定為目標；原來已停用時不強制開啟。
- Defender 的“啟用即時防護”明確以開啟為目標，不承諾還原之前的關閉狀態。
- 重新啟動和關機會強制關閉應用，確認提示必須說明並提醒保存工作。
- 日志僅記錄操作摘要和錯誤，不記錄密碼；最多500條，可導出。

## 自更新

檢查/下載任務鎖與狀態鎖分離，下載期間狀態讀取和取消不被網路請求阻塞。下載進度經現有橋接輪詢，SHA256 匹配指定安裝包；預發布版本按核心版本及預發布標識排序。軟體更新頁可關閉自動下載，偏好在本地保存。

安裝包暫存到當前使用者、系統管理員和SYSTEM可訪問的隨機目錄。更新器退出等待最長2分鐘，復制前後檢查哈希，用 File.Replace 保留 `.previous` 檔案；啟動失敗或5秒內退出時還原舊檔案。日志在使用者快取 WinToolbox/updates/last-install.log，暫存與舊檔案保留用于故障診斷。

進程存活5秒是有限的啟動檢測，不是 UI 就緒協議。真實更新鏈路、UAC 和殺毒軟體干預須在虛擬機驗收。

## Windows 虛擬機驗收

1. 記錄系統原值與還原點，分別驗證每個支持的 Windows 版本及域策略環境。
2. 注入備份寫失敗、權限不足、狀態逾時，確認不誤顯示成功或關閉。
3. RDP重新啟動失敗後確認介面顯示實際新連接埠和失敗說明。
4. Update在原設定停用時關閉再還原，確認原值一致且報告還原成功。
5. 對更新器注入哈希錯誤、復制失敗、啟動失敗，確認舊exe可恢復，日志可追蹤。
6. 檢查960×600、1120×720及125%/150%/200%縮放，側欄與操作按鈕均可訪問。
7. 檢查切頁草稿保留、未知狀態、下載取消、日志導出，以及後臺更新與系統操作同時進行的反饋。

自動化測試使用模擬對象和臨時目錄，不修改真實系統設定；不能替代上述虛擬機驗收。

## 模擬介面驗收

```powershell
python -m pip install -r docs/requirements.txt
python -m playwright install chromium
# 在一個終端執行 npm run dev，另一個終端執行：
python scripts/ui-smoke.py
```

腳本復用文件模擬後端，檢查十個頁面、960×600/1120×720窗口與草稿保留；截圖輸出到 artifacts/ui，所有系統寫操作均為模擬函數。
