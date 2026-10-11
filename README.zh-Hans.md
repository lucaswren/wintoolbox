[English](README.md) | [繁體中文](README.zh-Hant.md) | [简体中文](README.zh-Hans.md)

# WinToolBox

WinToolBox 是 Windows 本地管理工具箱，用于查看系统信息，管理本地账户、远程桌面、防火墙规则、Microsoft Defender、时间同步、电源和更新。基于 Go、Wails、Vue 和 Element Plus 开发。

版本：v1.0.0

## 文档

各操作文档**第一步均为下载本软件**：https://github.com/lucaswren/wintoolbox/releases/latest
浏览器拦截时请点 **保留 / 保持**；详见 [00 · 下载](./docs/zh-Hans/00-download.md)。

- [文档首页](./docs/zh-Hans/README.md)
- [00 · 下载](./docs/zh-Hans/00-download.md)
- [90 · 系统加固指南](./docs/zh-Hans/90-hardening-guide.md)
- [02 · 本地账户](./docs/zh-Hans/02-local-account.md)
- [03 · 远程桌面](./docs/zh-Hans/03-remote-desktop.md)
- [04 · 防火墙](./docs/zh-Hans/04-firewall.md)
- [05 · 安全加固](./docs/zh-Hans/05-security-harden.md)
- [09 · 系统更新](./docs/zh-Hans/09-windows-update.md)
- [全部功能文档](./docs/zh-Hans/README.md)

## v1.0.0 发布说明

- 包含本地账户、远程桌面、防火墙、安全加固、Defender、时间、电源及 Windows Update 管理
- 配置备份仅在全部写入成功后标记有效；恢复前校验备份，操作后复查状态
- 按模块刷新状态，合并并发查询，为系统写操作增加互斥，减少重复请求和界面等待
- 软件更新支持 SHA256 校验、下载进度及取消，安装失败可回退并保留日志
- 优化账户页、操作日志和小窗口布局，增加桥接类型检查、回归测试和 Windows CI
- 更新请求统一使用 `lucaswren/wintoolbox`，发布附件提供 SHA256 校验文件

本次将版本号重新设为 v1.0.0。已安装 v1.2.0 的用户需手动下载并安装，此版本不会被识别为更高版本。

## 功能

| 模块     | 能力 |
|----------|------|
| 本机概览 | 主机名、系统+Build、激活、分辨率、IP、磁盘与硬件信息 |
| 本地账户 | 改密码、启用/禁用、管理员；来宾/自动登录；锁定与密码策略 |
| 远程桌面 | 开关、改端口并同步防火墙、强制 NLA；清理 mstsc 连接记录 |
| 防火墙   | 一键开/关配置文件；禁 ping；高危端口预设；TCP 放行规则 |
| 安全加固 | SMBv1、WinRM、匿名枚举一键加固 |
| 防病毒   | 关闭/恢复 Defender 实时防护（含策略回退与复查） |
| 时间同步 | 时区、NTP 预设、立即同步、测试 NTP |
| 电源     | 锁定、延时重启/关机、取消计划 |
| 系统更新 | 关闭/恢复 Windows Update（按关闭前的配置备份还原，非云主机磁盘快照） |
| 软件更新 | GitHub Release 检测、SHA256 校验下载与安装 |

## 支持系统

Windows 10 / 11，Windows Server 2016 / 2019 / 2022 / 2025

需管理员权限。界面基于 WebView2；若未安装会自动联网下载并显示安装进度。

## 架构

```mermaid
flowchart TB
  subgraph uiLayer [界面层]
    Vue[frontend Vue3 views]
    Api[frontend api/composables]
  end
  subgraph bridge [桥接层]
    Bind[internal/ui bind DTO status]
    Wails[Wails v2]
  end
  subgraph domain [领域层]
    Acc[account]
    Rdp[rdp]
    Fw[firewall]
    Hard[harden]
    Def[defender]
    Time[wintime]
    Pwr[power]
    Sys[sysinfo]
    Wu[update]
    Su[selfupdate]
  end
  subgraph platform [平台层]
    Win[internal/win]
  end
  Vue --> Api --> Wails --> Bind
  Bind --> Acc & Rdp & Fw & Hard & Def & Time & Pwr & Sys & Wu & Su
  Acc & Rdp & Fw & Hard & Def & Time & Pwr & Sys & Wu & Su --> Win
```

## 项目结构

```
wintoolbox/
├── version.json
├── docs/                   # 操作文档与标注配图
├── scripts/build.ps1
├── frontend/src/views/     # 与侧栏一一对应
└── internal/               # account/rdp/firewall/harden/...
```

### 前端 views 对照

| 侧栏     | 视图           | 后端包     |
|----------|----------------|------------|
| 本机概览 | OverviewView   | sysinfo    |
| 本地账户 | AccountView    | account    |
| 远程桌面 | RdpView        | rdp        |
| 防火墙   | FirewallView   | firewall   |
| 安全加固 | HardenView     | harden     |
| 防病毒   | DefenderView   | defender   |
| 时间同步 | TimeView       | wintime    |
| 电源     | PowerView      | power      |
| 系统更新 | UpdateView     | update     |
| 软件更新 | SelfUpdateView | selfupdate |

## 构建

**依赖**：Go 1.27+、Node.js 22+、WebView2 Runtime

```powershell
.\scripts\build.ps1
```

开发模式（需安装 Wails CLI）：

```powershell
wails dev
```

## 运行

双击 `WinToolBox.exe` 按 UAC 提权，或以管理员身份运行。

## 下载与 SmartScreen

从 GitHub Release 下载的安装包**当前未做 Authenticode 代码签名**。Windows 可能提示 SmartScreen 拦截——多为未签名 + 新文件无下载信誉，不代表文件一定有害。

若确认来自本仓库 [Releases](https://github.com/lucaswren/wintoolbox/releases)：

1. **更多信息** → **仍要运行**
2. 用 Release / `*.sha256` 中的哈希校验后再运行

```powershell
Get-FileHash .\WinToolBox-v1.0.0-windows-amd64.exe -Algorithm SHA256
```

## 技术栈

Go · Wails v2 · Vue3 · Element Plus · Windows API / netsh / w32tm / tzutil / slmgr / WebView2

## 开发检查

```powershell
go run ./scripts/genbindings
cd frontend
npm ci
npm run check
npm run build
cd ..
go test ./...
go vet ./...
```

生产构建产物位于 `artifacts/releases/`，开发约定和故障验收见 [开发规范](docs/zh-Hans/development.md)。

## 多语言

支持英语、繁体中文和简体中文，以英语为默认及回退语言。首次启动根据 Windows 显示语言自动选择；侧栏可手动切换并保存偏好。软件直接运行 EXE，缺少 WebView2 时的自动安装窗口也根据系统语言显示。详见[语言选择](docs/zh-Hans/localization.md)。

[软件审查与协议建议](docs/zh-Hans/product-review-2026-10-11.md)

## 开源协议

WinToolBox 源代码和文档采用 [Apache-2.0](LICENSE)，项目声明见 [NOTICE](NOTICE)。第三方组件保留各自的许可证。
