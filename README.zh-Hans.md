[English](README.md) · [繁體中文](README.zh-Hant.md) · [简体中文](README.zh-Hans.md)

# WinToolBox

WinToolBox 是 Windows 本地管理工具箱，用于查看系统信息，管理本地账户、远程桌面、防火墙规则、Microsoft Defender、时间同步、电源和更新。基于 Go、Wails、Vue 和 Element Plus 开发。

版本：v1.1.2

## 文档

从 [GitHub Releases](https://github.com/lucaswren/wintoolbox/releases/latest) 下载。运行前请确认来源并校验 SHA256；下载和启动说明见 [00 · 下载](./docs/zh-Hans/00-download.md)。

- [文档首页](./docs/zh-Hans/README.md)
- [00 · 下载](./docs/zh-Hans/00-download.md)
- [90 · 系统加固指南](./docs/zh-Hans/90-hardening-guide.md)
- [02 · 本地账户](./docs/zh-Hans/02-local-account.md)
- [03 · 远程桌面](./docs/zh-Hans/03-remote-desktop.md)
- [04 · 防火墙](./docs/zh-Hans/04-firewall.md)
- [05 · 安全加固](./docs/zh-Hans/05-security-harden.md)
- [09 · 系统更新](./docs/zh-Hans/09-windows-update.md)
- [全部功能文档](./docs/zh-Hans/README.md)

## v1.1.2 发布说明

- 精简远程登录和系统日志页面，默认显示操作按钮、简短统计与关键记录。
- 筛选、范围与说明默认收起，原文、字段、XML 和关联证据点开查看；失败或记录不完整仍有简短提示。
- 优化小窗口换行、表格滚动和固定详情按钮，补齐英文标签。分析、搜索、导出、清空重读及客户端清理功能保留，Windows 原始日志不删除。
- 前端/Go 检查、11 页回归、168 组页面、234 组弹出层及最终 42 组最小窗口检查通过。
- Server/VPS 实机验收尚未执行；保留独立控制台，更新前先确认或恢复待处理的 RDP 保护变更。

[发布下载](https://github.com/lucaswren/wintoolbox/releases/tag/v1.1.2) · [详细验证报告](docs/zh-Hans/validation-vps-2026-10-23.md)

旧版 v1.2.0 用户仍需手动安装，自动更新不会降级。

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
Get-FileHash .\WinToolBox-v1.1.2-windows-amd64.exe -Algorithm SHA256
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

[全功能与界面审查](docs/zh-Hans/operation-audit-2026-10-12.md)
