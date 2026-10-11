[English](../project-review-2026-10-08.md) | [繁體中文](../zh-TW/project-review-2026-10-08.md) | [簡體中文](../zh-CN/project-review-2026-10-08.md)

# WinToolBox 項目審查

審查日期：2026-10-08。范圍：目錄、命名、Go 後端、Vue 前端、UI、系統操作鏈路、構建與質量規范。

## 結論與邊界

項目適合繼續沿用 Go + Wails + Vue 的架構。功能分包清晰，前端頁面與業務模塊大致對應，已實現異步頁面加載、狀態查詢序號、部分系統設置備份與寫後復查。當前最需要投入的是操作可靠性、狀態真實性、失敗恢復和自動化驗證，而非框架遷移或全面重寫。

以下“確認”指源碼中可直接觀察到的行為；潛在後果尚未通過故障注入復現。UI 視覺檢查基于倉庫帳戶、防火牆文件截圖及頁面源碼，不代表當前真實 Wails 窗口的完整交互驗收。沒有執行修改帳戶、防火牆、更新、Defender、RDP、時間或電源等系統操作。

## 項目結構與執行邏輯

```text
main.go                         提權 → 準備 WebView2 → 啟動 Wails
internal/ui/                    桌面殼、Wails 綁定、DTO、狀態匯總
internal/account/               使用者、密碼、鎖定、來賓、自動登錄
internal/rdp/                   RDP 開關/連接埠、NLA、歷史記錄
internal/firewall/              設定檔案、ping、放行規則、高危連接埠
internal/harden/                SMBv1、WinRM、匿名枚舉
internal/defender/              即時防護、備份恢復
internal/update/                Windows Update 策略、服務、任務、恢復
internal/selfupdate/            GitHub 更新發現、下載、校驗、程序替換
internal/sysinfo/               系統硬體、激活、快取
internal/wintime/               時區、NTP
internal/power/                 鎖定、重新啟動、關機、取消
internal/win/                   命令執行、連接埠校驗、提權、對話框、WebView2
frontend/src/views/             十個功能頁面
frontend/src/composables/       狀態獲取與操作執行
frontend/src/api/               手工維護的 Wails API 門面
scripts/                        版本同步、資源生成、構建
docs/                           使用說明及截圖生成工具
```

前端通常經 `useActions → createRunner → api → window.go.ui.App` 調用業務包。成功後刷新全局狀態；失敗顯示錯誤。`LoadStatus` 並發採集模塊並在 15 秒後返回可用結果，慢速概覽另行獲取。系統命令有獨立逾時和隱藏窗口設置。

這是桌面應用，沒有獨立 HTTP 後端服務。所謂“後端 UI”主要是 `internal/ui` 的 Wails 橋接和 Windows 原生啟動對話框。

## 檔案和目錄命名

| 范圍 | 現狀評價 | 建議 |
|---|---|---|
| Go 業務目錄 | account、rdp、firewall、power 等簡短清楚 | 保留，無需統一加 service 後綴 |
| update / selfupdate | 代碼可理解，但系統更新與應用更新容易混淆 | 可將 update 改為 winupdate；前端對應 WindowsUpdateView / AppUpdateView |
| wintime / sysinfo / syscmd | Go 風格合理，避免與標準庫 time 混淆 | 保留 |
| ui/bind.go、status.go、dto.go | 職責基本明確；util.go 名稱不夠具體 | bind.go 可用 app.go，util.go 隨真實職責拆分命名 |
| Vue 頁面 | PascalCase + View.vue，統一清晰 | 保留；HardenView 可改 HardeningView，但屬于低優先級 |
| useAction / useActions | 單復數差異掩蓋職責差異；useAction 實際導出 createRunner | 改 actionRunner.js 與 useSystemActions.js 更準確 |
| 字段縮寫 | accUser、fwPort、pol、st 在局部尚可，全局表單可讀性較低 | 表單用 accountUser、firewallPort；局部循環無需機械改名 |
| 協議大小寫 | Smb1 / SMBv1、WinRM、NLA 等混合 | Go 導出名統一 SMB1、WinRM、NLA；JSON 保持穩定 camelCase |
| 文件命名 | 數字前綴 + kebab-case 方便瀏覽 | 保留；工程審查/開發規范宜另設 development 或 architecture 目錄 |

命名調整需要同時更新導入、構建腳本、版本同步腳本、文件和 Wails 綁定引用；不建議為了風格一次性重命名所有檔案。

## 優先處理的問題

### P1：備份可能被錯誤標記為完整

依據：`internal/update/update.go:173`、`internal/defender/defender.go:190`。

備份流程忽略多個 `SetDWordValue` 的錯誤，最後僅依據 `Saved=1` 寫入成功認定備份完成。若中間寫入失敗，後續恢復仍可能讀取缺失字段。恢復代碼也忽略部分字段讀取錯誤：Update 的任務狀態可落到零值，Defender 的 HadPref/HadPolicy 等可能被解釋為不存在。

建議：逐項檢查寫入與讀取錯誤；缺失和訪問失敗必須分開；記錄 schemaVersion 與完整性；完整備份成功後才允許執行修改。恢復失敗保留備份，錯誤說明失敗階段。

### P1：狀態探測失敗被解釋成業務狀態

依據：`internal/firewall/riskports.go:40,233`、`internal/ui/status.go:253,364`、`internal/update/update.go:53`、`internal/wintime/wintime.go:24`。

高危連接埠查詢在 PowerShell 失敗後回退 netsh；netsh 查詢失敗返回 false。最終所有查詢失敗也可變成“預設未啟用”。橋接層只要收到結果便設置 RiskPortsUnknown=false。

Update 對 serviceStartType 的 unknown 沒有向外傳播 Unknown；橋接層收到結果即 UpdateUnknown=false。時間查詢失敗可返回 `-`，但收到結果仍被認為 TimeUnknown=false。

建議：查詢結果返回有效性和錯誤，支持 enabled / disabled / partial / unknown；保留最後成功狀態與採集時間，不能把查詢失敗當關閉、開啟或不存在。

### P1：操作部分成功後，UI 可能繼續顯示舊狀態

依據：`frontend/src/composables/useAction.js:24-43`、`internal/rdp/rdp.go:243`。

createRunner 只在 fn 成功後刷新。RDP 改連接埠先同步防火牆、寫注冊表，再重新啟動服務；重新啟動失敗會返回錯誤，但連接埠已修改。此時前端只報錯，不刷新。Update 等多階段操作也有類似情況。

建議：系統寫操作嘗試完成後，無論成功或失敗都刷新受影響模塊。返回結構化 OperationResult，區分 failed、partial、applied、verified、requiresRestart，並展示當前值和恢復建議。

### P1：自更新替換缺少恢復路徑，存在校驗與使用的時間間隔

依據：`internal/selfupdate/selfupdate.go:243`。

Apply 在當前進程校驗下載檔案後啟動臨時 PowerShell 腳本。腳本等待退出，然後 Copy-Item 覆蓋原 exe 並啟動；沒有舊檔案備份、替換失敗回滾、啟動結果確認或持久失敗日志。當前進程退出後無法展示復制失敗。腳本使用檔案前未再次校驗，從檢查到復制之間存在時間間隔；這屬于待驗證的安全風險，不是已證明的利用漏洞。

建議：獨立更新器使用受限暫存目錄；復制到目標旁臨時檔案並重新校驗，備份舊 exe，完成替換，確認啟動失敗時恢復；保存更新日志供下次啟動讀取。簽名作為後續發布能力補充。

### P2：“恢復原設定”和“恢復開啟狀態”判定沖突

依據：`internal/update/update.go:142`。

Enable 先恢復關閉前快照，再要求 GetStatus().Disabled=false。若關閉前已有停用更新策略或服務，精確恢復後仍可能被報為失敗。此時應評估“與快照一致”，不能只檢查“已開啟”。Defender 同樣需要明確恢復語義。

建議：產品區分“恢復修改前設定”和“啟用建議設定”；返回結果驗證目標值。UI 按實際語義命名按鈕。

### P2：全局逾時沒有取消底層任務

依據：`internal/ui/status.go:25`、`internal/win/syscmd/syscmd.go:43`。

15 秒只限制採集等待，業務函數不接收該 context；命令使用新的 Background context，仍可繼續執行。緩沖 channel 避免發送永久阻塞，但反復刷新可使多個採集批次重疊。串行防火牆 bundle 還會使已完成的 profiles/ping 等待最後一個探測。

建議：傳遞 context；按模塊設預算；全局刷新做請求合並，限制 PowerShell 並發；按頁面查詢或分模塊推送結果。不同模塊不要共享必須全部完成才返回的結果包。

### P2：後端缺少統一的寫操作並發保護

依據：`internal/ui/bind.go`、Update/Defender 的備份後修改流程。

前端 busy 能阻止本窗口多數重復操作，但不是後端串行保證。現有 selfupdate 有互斥鎖，RDP 有背景規則調度保護；沒有覆蓋所有系統寫操作的協調層。多窗口/多實例或直接並發橋接調用時，備份、寫入、恢復可能交錯。

建議：按資源設置操作鎖；備份和修改處于同一臨界區。是否需要單實例限制按使用場景決定，不宜用一把全局鎖阻塞所有只讀操作。

### P2：系統命令依賴語言和預設輸出編碼

依據：`internal/account/account.go:170,335`、`internal/update/update.go:490,541`、`internal/win/syscmd/syscmd.go:43`。

系統管理員組僅嘗試 Administrators 和 系統管理員，組被重命名或其他語言時不穩。服務、任務、規則存在文本關鍵字解析。syscmd 將所有輸出按 ACP 解碼，與未來輸出 UTF-8 的腳本缺少明確契約；stdout/stderr 混合也不利于穩定解析。

建議：組用固定 SID 解析真實名稱，逐步采用 NetAPI、SCM、任務 API；PowerShell 輸出固定 UTF-8 + JSON，錯誤輸出單獨處理。統一返回退出碼、逾時類別與安全的錯誤摘要。

### P2：自動化測試與持續檢查缺失

當前 Go 包全部顯示 `[no test files]`，前端只有 dev/build/preview，沒有 lint/test/typecheck，未發現倉庫 CI 工作流。

建議優先覆蓋：備份中途失敗不得標 Saved；查詢失敗必須 unknown；RDP 部分成功後刷新；恢復到原停用設定應成功；版本比較、checksum 多檔案匹配；歷史刪除路徑邊界；狀態競態。系統修改測試放 Windows 虛擬機與模擬依賴中。

## 前端和 UI

已有優點：菜單與功能頁一致；Element Plus 卡片/標簽/表單風格統一；共享 CSS 變量；頁面異步加載；dangerous action 確認；狀態請求序號避免舊返回覆蓋新返回；表單不會簡單隨刷新覆蓋所有編輯內容。

需要優化：

1. **顏色語義**：AccountView:54 把鎖定關閉標綠色 success、開啟標 warning，與安全加固頁面的視覺語義不一致。顏色表達風險/健康度，文字表達開關狀態。
2. **Ping 文案準確性**：FirewallView 把未檢測到本工具攔截規則表述為“可響應 ping”。實際響應還依賴其他規則、網路等；建議寫“未檢測到本工具的 ping 攔截”。連接埠預設也應表達工具規則狀態，避免等同于外部可達性。
3. **頁面密度**：帳戶頁卡片多，倉庫截圖中帳戶管理區域需要繼續滾動；可拆為使用者管理、安全策略兩個分區。頂部標題與頁面標題重復，可保留一個。
4. **局部刷新**：App.vue 的主區域 v-loading 綁定全量刷新，單項操作之後可能遮住整個頁面最長 15 秒。增加模塊 loading、模塊更新時間，避免所有操作等所有採集。
5. **切頁保留編輯**：v-if 切頁銷毀 View；父層 form 會保留，但 AccountView 本地自定義策略、dirty 標記會丟失。提升草稿到 composable 或用 KeepAlive 並管理快取刷新。
6. **窄屏布局**：CSS 動作條預設 nowrap，僅 viewport ≤720px 換行，但窗口最小寬960px；側欄占位後主區域更窄。該響應式分支通常不會在桌面窗口觸發。建議 container query 或常態 flex-wrap，並實測960×600、高 DPI；當前未實測溢出。
7. **操作進度**：自更新最長10分鐘下載，頁面主要顯示 checking，沒有下載百分比、速率、取消。後臺自動下載需要設置項或清晰狀態提示。
8. **強制關機**：power.go API 使用 force=1，命令回退使用 /f；現有確認文案僅提關機時間。應明確將強制關閉應用並提醒保存工作，或提供強制選項。
9. **日志**：App.vue:56 無容量上限；OpLog 是純字符串列表。建議限制條數、增加嚴重程度/模塊/階段/時間、復制導出和自動滾動；敏感字段不進入日志。
10. **重復錯誤反饋**：refresh 本身記錄並顯示失敗，上層 runner 再次記錄刷新失敗。建議由一個層級負責使用者反饋。

原生 UI 啟動順序是先提權，再 WebView2，最後桌面；WebView2 安裝器有下載進度和微軟簽名檢查，這是值得保留的設計。可以補充取消和離線安裝路徑，但需以實際安裝流程驗收。

## 代碼規范和可維護性

- API 門面稱 Typed facade，但實際是普通 JS 和字符串方法名，沒有編譯期類型約束；DTO → useAppState → App.status → View 多次手工列舉同一字段。優先采用 Wails 生成綁定/模型與 TypeScript，按業務狀態分組。
- 大檔案集中在 firewall、sysinfo、rdp/history、account/lockout、App、useActions。按穩定職責拆分，例如 firewall/profiles.go、ping.go、allow_rules.go、probe.go；sysinfo/cache.go、hardware.go、activation.go；rdp/history 按憑據、注冊表、檔案拆分。
- 不要按行數機械拆檔案，也不必立即引入復雜倉儲層。先為外部系統調用增加可替換接口，以便故障測試。
- 前端格式混合緊湊/寬松空格；建議 Prettier + ESLint Vue。Go 用 gofmt，必要時加靜態檢查。
- 當前 gofmt -l 列出 password_policy.go、harden.go、sysinfo.go、ui/dto.go、ui/status.go 五個檔案，格式規范尚未完全落地。
- Windows 專用實現可用 `_windows.go` 或 build tags 明確平臺邊界；可測試的版本解析/校驗函數獨立出來，減少平臺依賴。
- 內部業務包直接生成中文 UI 文案，後續若需要國際化或統一 OperationResult，再逐步使用錯誤碼/錯誤類型；當前體量無需一次性重構全部錯誤。

## 構建、發布和文件

1. README 聲稱 Go 1.22+，go.mod 實際要求1.27，本機也是1.27.0。文件必須與項目設定一致。
2. build.ps1 只在 node_modules 不存在時 npm install；目錄存在不能保證與 lockfile 一致。CI 用 npm ci，開發模式另設顯式安裝/檢查流程。
3. sync-version 修改 package.json，沒有同步 package-lock.json；改版後可能有版本元數據漂移。避免正則修改 JSON，使用 JSON 解析並校驗；源碼常量可生成或 ldflags 注入。
4. 根目錄存在多個舊 exe/hash 與 syso，但 .gitignore 已排除這些產物，不能認定為 Git 污染。建議發布產物統一 artifacts 或 release 目錄；syso 遵守 Go 資源鏈接位置要求。
5. 自更新摘要檔案解析未按目標檔案匹配 SHA256.txt 多行內容；parseSHA256Text 取首個摘要，版本比較忽略預發布語義。當前單 exe 發布流程可以工作，多架構/預發布前應完善。
6. 更新下載在 mutex 內持續網路 I/O，Cached/Check 可長時間等待。鎖只保護狀態快照；下載以獨立任務和事件更新進度，拒絕重復任務。
7. 文件截圖腳本依賴 Playwright/Pillow，建議固定工具依賴並提供執行命令；截圖是模擬展示，不能代替系統集成測試。
8. 加開發文件：支持系統矩陣、貢獻規范、錯誤/狀態契約、系統操作驗證與回滾約定、版本發布檢查。

## 建議實施順序

| 階段 | 內容 | 驗收依據 |
|---|---|---|
| 第一階段 | 備份錯誤檢查、unknown傳播、失敗後狀態刷新、自更新回滾 | 故障注入覆蓋關鍵失敗階段 |
| 第二階段 | 恢復語義、context、按模塊刷新、後端操作鎖 | 原設定恢復一致；重復刷新不無限堆積 |
| 第三階段 | 生成綁定/類型、模塊拆分、統一規范與CI | 綁定變更可被類型檢查發現，Windows構建檢查通過 |
| 第四階段 | UI密度、進度、日志、縮放及發布體驗 | 最小窗口、高DPI、慢網路和未知狀態人工驗收 |

不建議此時遷移框架、引入微服務或復雜路由體系。先使現有功能可驗證、可解釋、可恢復。

## 本次驗證記錄

- `go test ./internal/...`：通過編譯，各包均無測試檔案，不能證明功能正確。
- `go test ./...`：全項目通過，各包均無測試檔案。
- `go vet ./internal/...`：退出碼0，無診斷。
- `npm run build`：首次沙箱 EPERM；獲自動審批後在沙箱外重試成功。兩條第三方 PURE 注釋警告，不影響構建。
- 前端首個 index JS約230.64kB，gzip79.88kB；Vue約81.73kB，gzip32.47kB；功能頁已拆包，目前包體不是主要優化瓶頸。
- `gofmt -l`：五個檔案需要格式化，本次未修改業務源碼。
- UI檢查：倉庫文件帳戶、防火牆截圖與頁面/CSS源碼；未啟動並操作真實系統管理員桌面窗口。

本次產物為審查報告；源碼修復需按上述順序另行實施。
