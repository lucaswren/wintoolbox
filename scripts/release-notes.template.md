### 更新内容

- 新增网络检查，支持 DNS、Ping、TCP 端口、本机监听和高级 MTR。区分目标不回应、端口可连通和本机网络不可达。
- 系统信息新增磁盘余量、运行时长、服务状态、近期异常和可预览的故障摘要。
- 顶部新增帮助。首次启动弹出新手指引，可直接打开常用功能或当前语言的在线文档。
- 统一侧栏顺序、三语文档和截图，清理旧截图及已被保护流程替代的远程桌面代码。

本机 Windows 已对照验证公网 IPv4 DNS、Ping、TCP 443 和 MTR 多跳。Windows Server、VPS 和公网 IPv6 多跳仍待实机验证。

**此前已安装历史 1.2.0 的用户需手动下载安装本次发布。** 自动更新按版本号比较，无法识别同版本号的新包。

### Changes

- Added DNS, Ping, TCP and local listener checks, plus MTR in Advanced diagnostics.
- Added disk space, uptime, service status, recent system events and a previewable troubleshooting summary.
- Added a first-run guide and a Help button linking to the current language's documentation. The sidebar and documentation follow the same order.
- Verified public IPv4 checks on a local Windows computer. Windows Server, VPS and public IPv6 multi-hop tests remain pending.

Users with the historical 1.2.0 build must download this release manually. Automatic updates require a higher version number.

### 下载与校验 / Download and verification

- File: `{{EXE}}`
- SHA256: `{{SHA256}}`

```powershell
Get-FileHash .\{{EXE}} -Algorithm SHA256
```

对照 `{{SHA256_FILE}}` 校验。程序目前未签名，请从本页下载。
Verify the checksum against `{{SHA256_FILE}}`. Download this unsigned executable from this release page.

[English](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/README.md) · [繁體中文](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/zh-Hant/README.md) · [简体中文](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/zh-Hans/README.md)
