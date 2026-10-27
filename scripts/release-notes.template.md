## {{VERSION}}

### 更新内容

- 修复 Windows 11 修改 RDP 端口时，把仅适用于 System 进程的 Teredo 发现规则误判为宽泛 RDP 放行的问题；保留真实宽泛规则、白名单、回滚及重连确认检查。
- 优化安全加固状态读取：批量查询有效防火墙规则、原生只读服务查询、并行采集和统一超时；超时保留已读结果，权限错误标记为未知。
- 防止 WinRM 加固封堵自定义 RDP 端口：当前端口为 5985/5986 或无法确认 RDP 状态时中止写入。
- 增强远程登录分析：1–90 天范围、来源 IP 汇总、登录失败原因、短时间重复失败提示、搜索分类、原文/字段/XML 详情及完整 JSON 导出。
- 区分远程交互式登录、普通网络认证、身份验证成功及来源未知会话；通道不可读时保留其他结果并提示日志不完整，避免误报登录或无人访问。重复失败提示不代表已确认入侵。
- 远程登录和系统日志页面新增“清空结果”：仅重置当前显示、统计、详情、筛选和错误提示，**不删除 Windows 原始日志**；读取期间禁用，重读可恢复历史记录。
- 修复日志时区/小数秒排序、消息资源缺失处理、跨通道记录去重，以及修改筛选后未重读时导出范围不一致的问题。现有客户端历史、凭据和缓存清理功能保留。

### 验证与适用范围

- 前端检查与 20 项测试、Go 测试、静态检查、桥接契约及生产构建通过。
- 11 页模拟回归和 366 组多语言/小窗口排版检查通过；覆盖清空/重读、读取中禁用、失败保留、导出范围及 800×500 登录详情。它们使用模拟后端，不代表真实服务器断连或重启验收。
- Windows 11 本机只读验证：加固页面采集约 2.52 秒；远程日志读取返回可用通道，并正确提示 Security 通道权限不足。失败原因等异常路径使用隔离样本验证，未清除真实日志或修改本机连接配置。
- 面向 Windows Server 2016/2019/2022/2025 Desktop Experience 和 Windows 远程 VPS；Server Core、会话 0 和域控制器不支持。受管策略或无法确定有效规则时保守拒绝变更。
- **目前没有 Windows Server/VPS 测试机，实机断连、重连、SYSTEM 回滚任务和重启验收尚未执行。远程变更前应保留服务商独立控制台恢复入口，云安全组需另行核对。**
- 更新前请先确认或恢复待处理的 RDP 保护变更。
- [详细验证报告](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/zh-Hans/validation-vps-2026-10-23.md)

### Changes (English)

- Fixed Windows 11 Teredo/System rule misclassification during protected RDP port changes, bounded hardening-status reads, and prevented WinRM hardening from blocking custom RDP ports 5985/5986.
- Enhanced remote sign-in analysis with source summaries, failure reasons, repeated-failure hints, filters, XML details and JSON export. Network authentication and unconfirmed sessions are distinguished from remote desktop sign-ins; partial channel coverage is explicit.
- Added Clear results to both log pages. This resets the displayed results without deleting Windows event logs. Retained client history, credential and cache cleanup.
- Fixed timestamp ordering, message-resource error handling and exported collection scope. Passed Go/frontend checks, 11-page regression and 366 layout cases. Windows Server/VPS live acceptance remains unperformed; retain an independent recovery console and finish pending RDP changes before updating.

Users running v1.2.0 must download {{VERSION}} manually; automatic updates do not downgrade versions.

### Download and verification

The executable is not Authenticode-signed. Confirm the release source and verify the checksum before running. Follow your organization's application policy.

- File: `{{EXE}}`
- SHA256: {{SHA256}}

```powershell
Get-FileHash .\{{EXE}} -Algorithm SHA256
```

Release files: {{EXE}}, {{SHA256_FILE}}, LICENSE and NOTICE.

[English documentation](https://github.com/lucaswren/wintoolbox/blob/main/docs/README.md) · [繁體中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hant/README.md) · [简体中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hans/README.md)
