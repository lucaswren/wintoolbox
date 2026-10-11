[English](../project-review-2026-10-08.md) | [繁體中文](../zh-TW/project-review-2026-10-08.md) | [简体中文](../zh-CN/project-review-2026-10-08.md)

# WinToolBox 项目审查

审查日期：2026-10-08。范围：目录、命名、Go 后端、Vue 前端、UI、系统操作链路、构建与质量规范。

## 结论与边界

项目适合继续沿用 Go + Wails + Vue 的架构。功能分包清晰，前端页面与业务模块大致对应，已实现异步页面加载、状态查询序号、部分系统设置备份与写后复查。当前最需要投入的是操作可靠性、状态真实性、失败恢复和自动化验证，而非框架迁移或全面重写。

以下“确认”指源码中可直接观察到的行为；潜在后果尚未通过故障注入复现。UI 视觉检查基于仓库账户、防火墙文档截图及页面源码，不代表当前真实 Wails 窗口的完整交互验收。没有执行修改账户、防火墙、更新、Defender、RDP、时间或电源等系统操作。

## 项目结构与运行逻辑

```text
main.go                         提权 → 准备 WebView2 → 启动 Wails
internal/ui/                    桌面壳、Wails 绑定、DTO、状态汇总
internal/account/               用户、密码、锁定、来宾、自动登录
internal/rdp/                   RDP 开关/端口、NLA、历史记录
internal/firewall/              配置文件、ping、放行规则、高危端口
internal/harden/                SMBv1、WinRM、匿名枚举
internal/defender/              实时防护、备份恢复
internal/update/                Windows Update 策略、服务、任务、恢复
internal/selfupdate/            GitHub 更新发现、下载、校验、程序替换
internal/sysinfo/               系统硬件、激活、缓存
internal/wintime/               时区、NTP
internal/power/                 锁定、重启、关机、取消
internal/win/                   命令执行、端口校验、提权、对话框、WebView2
frontend/src/views/             十个功能页面
frontend/src/composables/       状态获取与操作执行
frontend/src/api/               手工维护的 Wails API 门面
scripts/                        版本同步、资源生成、构建
docs/                           使用说明及截图生成工具
```

前端通常经 `useActions → createRunner → api → window.go.ui.App` 调用业务包。成功后刷新全局状态；失败显示错误。`LoadStatus` 并发采集模块并在 15 秒后返回可用结果，慢速概览另行获取。系统命令有独立超时和隐藏窗口设置。

这是桌面应用，没有独立 HTTP 后端服务。所谓“后端 UI”主要是 `internal/ui` 的 Wails 桥接和 Windows 原生启动对话框。

## 文件和目录命名

| 范围 | 现状评价 | 建议 |
|---|---|---|
| Go 业务目录 | account、rdp、firewall、power 等简短清楚 | 保留，无需统一加 service 后缀 |
| update / selfupdate | 代码可理解，但系统更新与应用更新容易混淆 | 可将 update 改为 winupdate；前端对应 WindowsUpdateView / AppUpdateView |
| wintime / sysinfo / syscmd | Go 风格合理，避免与标准库 time 混淆 | 保留 |
| ui/bind.go、status.go、dto.go | 职责基本明确；util.go 名称不够具体 | bind.go 可用 app.go，util.go 随真实职责拆分命名 |
| Vue 页面 | PascalCase + View.vue，统一清晰 | 保留；HardenView 可改 HardeningView，但属于低优先级 |
| useAction / useActions | 单复数差异掩盖职责差异；useAction 实际导出 createRunner | 改 actionRunner.js 与 useSystemActions.js 更准确 |
| 字段缩写 | accUser、fwPort、pol、st 在局部尚可，全局表单可读性较低 | 表单用 accountUser、firewallPort；局部循环无需机械改名 |
| 协议大小写 | Smb1 / SMBv1、WinRM、NLA 等混合 | Go 导出名统一 SMB1、WinRM、NLA；JSON 保持稳定 camelCase |
| 文档命名 | 数字前缀 + kebab-case 方便浏览 | 保留；工程审查/开发规范宜另设 development 或 architecture 目录 |

命名调整需要同时更新导入、构建脚本、版本同步脚本、文档和 Wails 绑定引用；不建议为了风格一次性重命名所有文件。

## 优先处理的问题

### P1：备份可能被错误标记为完整

依据：`internal/update/update.go:173`、`internal/defender/defender.go:190`。

备份流程忽略多个 `SetDWordValue` 的错误，最后仅依据 `Saved=1` 写入成功认定备份完成。若中间写入失败，后续恢复仍可能读取缺失字段。恢复代码也忽略部分字段读取错误：Update 的任务状态可落到零值，Defender 的 HadPref/HadPolicy 等可能被解释为不存在。

建议：逐项检查写入与读取错误；缺失和访问失败必须分开；记录 schemaVersion 与完整性；完整备份成功后才允许执行修改。恢复失败保留备份，错误说明失败阶段。

### P1：状态探测失败被解释成业务状态

依据：`internal/firewall/riskports.go:40,233`、`internal/ui/status.go:253,364`、`internal/update/update.go:53`、`internal/wintime/wintime.go:24`。

高危端口查询在 PowerShell 失败后回退 netsh；netsh 查询失败返回 false。最终所有查询失败也可变成“预设未启用”。桥接层只要收到结果便设置 RiskPortsUnknown=false。

Update 对 serviceStartType 的 unknown 没有向外传播 Unknown；桥接层收到结果即 UpdateUnknown=false。时间查询失败可返回 `-`，但收到结果仍被认为 TimeUnknown=false。

建议：查询结果返回有效性和错误，支持 enabled / disabled / partial / unknown；保留最后成功状态与采集时间，不能把查询失败当关闭、开启或不存在。

### P1：操作部分成功后，UI 可能继续显示旧状态

依据：`frontend/src/composables/useAction.js:24-43`、`internal/rdp/rdp.go:243`。

createRunner 只在 fn 成功后刷新。RDP 改端口先同步防火墙、写注册表，再重启服务；重启失败会返回错误，但端口已修改。此时前端只报错，不刷新。Update 等多阶段操作也有类似情况。

建议：系统写操作尝试完成后，无论成功或失败都刷新受影响模块。返回结构化 OperationResult，区分 failed、partial、applied、verified、requiresRestart，并展示当前值和恢复建议。

### P1：自更新替换缺少恢复路径，存在校验与使用的时间间隔

依据：`internal/selfupdate/selfupdate.go:243`。

Apply 在当前进程校验下载文件后启动临时 PowerShell 脚本。脚本等待退出，然后 Copy-Item 覆盖原 exe 并启动；没有旧文件备份、替换失败回滚、启动结果确认或持久失败日志。当前进程退出后无法展示复制失败。脚本使用文件前未再次校验，从检查到复制之间存在时间间隔；这属于待验证的安全风险，不是已证明的利用漏洞。

建议：独立更新器使用受限暂存目录；复制到目标旁临时文件并重新校验，备份旧 exe，完成替换，确认启动失败时恢复；保存更新日志供下次启动读取。签名作为后续发布能力补充。

### P2：“恢复原配置”和“恢复开启状态”判定冲突

依据：`internal/update/update.go:142`。

Enable 先恢复关闭前快照，再要求 GetStatus().Disabled=false。若关闭前已有禁用更新策略或服务，精确恢复后仍可能被报为失败。此时应评估“与快照一致”，不能只检查“已开启”。Defender 同样需要明确恢复语义。

建议：产品区分“恢复修改前配置”和“启用建议配置”；返回结果验证目标值。UI 按实际语义命名按钮。

### P2：全局超时没有取消底层任务

依据：`internal/ui/status.go:25`、`internal/win/syscmd/syscmd.go:43`。

15 秒只限制采集等待，业务函数不接收该 context；命令使用新的 Background context，仍可继续运行。缓冲 channel 避免发送永久阻塞，但反复刷新可使多个采集批次重叠。串行防火墙 bundle 还会使已完成的 profiles/ping 等待最后一个探测。

建议：传递 context；按模块设预算；全局刷新做请求合并，限制 PowerShell 并发；按页面查询或分模块推送结果。不同模块不要共享必须全部完成才返回的结果包。

### P2：后端缺少统一的写操作并发保护

依据：`internal/ui/bind.go`、Update/Defender 的备份后修改流程。

前端 busy 能阻止本窗口多数重复操作，但不是后端串行保证。现有 selfupdate 有互斥锁，RDP 有背景规则调度保护；没有覆盖所有系统写操作的协调层。多窗口/多实例或直接并发桥接调用时，备份、写入、恢复可能交错。

建议：按资源设置操作锁；备份和修改处于同一临界区。是否需要单实例限制按使用场景决定，不宜用一把全局锁阻塞所有只读操作。

### P2：系统命令依赖语言和默认输出编码

依据：`internal/account/account.go:170,335`、`internal/update/update.go:490,541`、`internal/win/syscmd/syscmd.go:43`。

管理员组仅尝试 Administrators 和 管理员，组被重命名或其他语言时不稳。服务、任务、规则存在文本关键字解析。syscmd 将所有输出按 ACP 解码，与未来输出 UTF-8 的脚本缺少明确契约；stdout/stderr 混合也不利于稳定解析。

建议：组用固定 SID 解析真实名称，逐步采用 NetAPI、SCM、任务 API；PowerShell 输出固定 UTF-8 + JSON，错误输出单独处理。统一返回退出码、超时类别与安全的错误摘要。

### P2：自动化测试与持续检查缺失

当前 Go 包全部显示 `[no test files]`，前端只有 dev/build/preview，没有 lint/test/typecheck，未发现仓库 CI 工作流。

建议优先覆盖：备份中途失败不得标 Saved；查询失败必须 unknown；RDP 部分成功后刷新；恢复到原禁用配置应成功；版本比较、checksum 多文件匹配；历史删除路径边界；状态竞态。系统修改测试放 Windows 虚拟机与模拟依赖中。

## 前端和 UI

已有优点：菜单与功能页一致；Element Plus 卡片/标签/表单风格统一；共享 CSS 变量；页面异步加载；dangerous action 确认；状态请求序号避免旧返回覆盖新返回；表单不会简单随刷新覆盖所有编辑内容。

需要优化：

1. **颜色语义**：AccountView:54 把锁定关闭标绿色 success、开启标 warning，与安全加固页面的视觉语义不一致。颜色表达风险/健康度，文字表达开关状态。
2. **Ping 文案准确性**：FirewallView 把未检测到本工具拦截规则表述为“可响应 ping”。实际响应还依赖其他规则、网络等；建议写“未检测到本工具的 ping 拦截”。端口预设也应表达工具规则状态，避免等同于外部可达性。
3. **页面密度**：账户页卡片多，仓库截图中账户管理区域需要继续滚动；可拆为用户管理、安全策略两个分区。顶部标题与页面标题重复，可保留一个。
4. **局部刷新**：App.vue 的主区域 v-loading 绑定全量刷新，单项操作之后可能遮住整个页面最长 15 秒。增加模块 loading、模块更新时间，避免所有操作等所有采集。
5. **切页保留编辑**：v-if 切页销毁 View；父层 form 会保留，但 AccountView 本地自定义策略、dirty 标记会丢失。提升草稿到 composable 或用 KeepAlive 并管理缓存刷新。
6. **窄屏布局**：CSS 动作条默认 nowrap，仅 viewport ≤720px 换行，但窗口最小宽960px；侧栏占位后主区域更窄。该响应式分支通常不会在桌面窗口触发。建议 container query 或常态 flex-wrap，并实测960×600、高 DPI；当前未实测溢出。
7. **操作进度**：自更新最长10分钟下载，页面主要显示 checking，没有下载百分比、速率、取消。后台自动下载需要设置项或清晰状态提示。
8. **强制关机**：power.go API 使用 force=1，命令回退使用 /f；现有确认文案仅提关机时间。应明确将强制关闭应用并提醒保存工作，或提供强制选项。
9. **日志**：App.vue:56 无容量上限；OpLog 是纯字符串列表。建议限制条数、增加严重程度/模块/阶段/时间、复制导出和自动滚动；敏感字段不进入日志。
10. **重复错误反馈**：refresh 本身记录并显示失败，上层 runner 再次记录刷新失败。建议由一个层级负责用户反馈。

原生 UI 启动顺序是先提权，再 WebView2，最后桌面；WebView2 安装器有下载进度和微软签名检查，这是值得保留的设计。可以补充取消和离线安装路径，但需以实际安装流程验收。

## 代码规范和可维护性

- API 门面称 Typed facade，但实际是普通 JS 和字符串方法名，没有编译期类型约束；DTO → useAppState → App.status → View 多次手工列举同一字段。优先采用 Wails 生成绑定/模型与 TypeScript，按业务状态分组。
- 大文件集中在 firewall、sysinfo、rdp/history、account/lockout、App、useActions。按稳定职责拆分，例如 firewall/profiles.go、ping.go、allow_rules.go、probe.go；sysinfo/cache.go、hardware.go、activation.go；rdp/history 按凭据、注册表、文件拆分。
- 不要按行数机械拆文件，也不必立即引入复杂仓储层。先为外部系统调用增加可替换接口，以便故障测试。
- 前端格式混合紧凑/宽松空格；建议 Prettier + ESLint Vue。Go 用 gofmt，必要时加静态检查。
- 当前 gofmt -l 列出 password_policy.go、harden.go、sysinfo.go、ui/dto.go、ui/status.go 五个文件，格式规范尚未完全落地。
- Windows 专用实现可用 `_windows.go` 或 build tags 明确平台边界；可测试的版本解析/校验函数独立出来，减少平台依赖。
- 内部业务包直接生成中文 UI 文案，后续若需要国际化或统一 OperationResult，再逐步使用错误码/错误类型；当前体量无需一次性重构全部错误。

## 构建、发布和文档

1. README 声称 Go 1.22+，go.mod 实际要求1.27，本机也是1.27.0。文档必须与项目配置一致。
2. build.ps1 只在 node_modules 不存在时 npm install；目录存在不能保证与 lockfile 一致。CI 用 npm ci，开发模式另设显式安装/检查流程。
3. sync-version 修改 package.json，没有同步 package-lock.json；改版后可能有版本元数据漂移。避免正则修改 JSON，使用 JSON 解析并校验；源码常量可生成或 ldflags 注入。
4. 根目录存在多个旧 exe/hash 与 syso，但 .gitignore 已排除这些产物，不能认定为 Git 污染。建议发布产物统一 artifacts 或 release 目录；syso 遵守 Go 资源链接位置要求。
5. 自更新摘要文件解析未按目标文件匹配 SHA256.txt 多行内容；parseSHA256Text 取首个摘要，版本比较忽略预发布语义。当前单 exe 发布流程可以工作，多架构/预发布前应完善。
6. 更新下载在 mutex 内持续网络 I/O，Cached/Check 可长时间等待。锁只保护状态快照；下载以独立任务和事件更新进度，拒绝重复任务。
7. 文档截图脚本依赖 Playwright/Pillow，建议固定工具依赖并提供运行命令；截图是模拟展示，不能代替系统集成测试。
8. 加开发文档：支持系统矩阵、贡献规范、错误/状态契约、系统操作验证与回滚约定、版本发布检查。

## 建议实施顺序

| 阶段 | 内容 | 验收依据 |
|---|---|---|
| 第一阶段 | 备份错误检查、unknown传播、失败后状态刷新、自更新回滚 | 故障注入覆盖关键失败阶段 |
| 第二阶段 | 恢复语义、context、按模块刷新、后端操作锁 | 原配置恢复一致；重复刷新不无限堆积 |
| 第三阶段 | 生成绑定/类型、模块拆分、统一规范与CI | 绑定变更可被类型检查发现，Windows构建检查通过 |
| 第四阶段 | UI密度、进度、日志、缩放及发布体验 | 最小窗口、高DPI、慢网络和未知状态人工验收 |

不建议此时迁移框架、引入微服务或复杂路由体系。先使现有功能可验证、可解释、可恢复。

## 本次验证记录

- `go test ./internal/...`：通过编译，各包均无测试文件，不能证明功能正确。
- `go test ./...`：全项目通过，各包均无测试文件。
- `go vet ./internal/...`：退出码0，无诊断。
- `npm run build`：首次沙箱 EPERM；获自动审批后在沙箱外重试成功。两条第三方 PURE 注释警告，不影响构建。
- 前端首个 index JS约230.64kB，gzip79.88kB；Vue约81.73kB，gzip32.47kB；功能页已拆包，目前包体不是主要优化瓶颈。
- `gofmt -l`：五个文件需要格式化，本次未修改业务源码。
- UI检查：仓库文档账户、防火墙截图与页面/CSS源码；未启动并操作真实管理员桌面窗口。

本次产物为审查报告；源码修复需按上述顺序另行实施。
