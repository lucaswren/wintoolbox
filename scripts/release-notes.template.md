## {{VERSION}}

### 发布内容

- 本地账户、远程桌面、防火墙、安全加固、Defender、时间同步、电源及 Windows Update 管理
- 严格验证配置备份和恢复结果；按模块刷新、合并并发探测，为系统写操作增加互斥
- 软件更新提供 SHA256 验证、下载进度、取消及安装失败回退日志
- 改善账户页、操作日志及小窗口布局，增加桥接类型检查、回归测试和 Windows CI
- 更新接口统一为 `https://api.github.com/repos/lucaswren/wintoolbox/releases/latest`

本次重新发布为 {{VERSION}}。已安装 v1.2.0 的用户需手动下载并安装，软件不会自动降级。

### 下载说明（SmartScreen）

当前发布包**尚未使用 Authenticode 代码签名**。从浏览器下载后，Windows 可能提示「Windows 已保护你的电脑」或 SmartScreen 拦截，属常见现象（未签名 + 新文件哈希无信誉），**不一定是病毒**。

若确认来自本仓库 Release：

1. 点击 **更多信息** → **仍要运行**（企业策略可能禁止此选项）
2. 用下方 SHA256 校验文件完整性后再运行

### 校验

- 文件：`{{EXE}}`
- SHA256: {{SHA256}}

PowerShell 校验示例：

```powershell
Get-FileHash .\{{EXE}} -Algorithm SHA256
# 期望：{{SHA256}}
```

### 文档

- [文档首页](https://github.com/lucaswren/wintoolbox/blob/main/docs/README.md)
- [00 · 下载](https://github.com/lucaswren/wintoolbox/blob/main/docs/00-download.md)
- [90 · 系统加固指南](https://github.com/lucaswren/wintoolbox/blob/main/docs/90-hardening-guide.md)
- [02 · 本地账户](https://github.com/lucaswren/wintoolbox/blob/main/docs/02-local-account.md)
- [03 · 远程桌面](https://github.com/lucaswren/wintoolbox/blob/main/docs/03-remote-desktop.md)
- [09 · 系统更新](https://github.com/lucaswren/wintoolbox/blob/main/docs/09-windows-update.md)

### 资产

本版本附件：

- `{{EXE}}`（主程序）
- `{{SHA256_FILE}}`（SHA256 校验和）

> 软件内自更新会依次尝试：GitHub asset digest → 本说明中的 SHA256 行 → `*.sha256` / `SHA256.txt` 附件。
