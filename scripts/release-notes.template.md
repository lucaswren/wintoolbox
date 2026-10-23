## {{VERSION}}

### 更新内容

- 新增 RDP 来源 IP 白名单、会话保护、独立计划任务超时回滚，以及重新连接后确认保留；检查实际监听和有效防火墙规则，并提供登录、断连记录。
- 新增系统日志分析：开机、关机、重启、异常重启和蓝屏线索，支持时间关联、原始事件/XML 查看与 JSON 导出。异常重启记录不直接等同于故障根因。
- 新增账户用户名修改；完整保留客户端历史、凭据、文件与缓存清理。
- 简化菜单与操作反馈：成功提示 3 秒自动隐藏，失败提示保留，详细结果写入日志；底部活动日志默认收起。
- 优化小窗口、多语言、表格与按钮布局；长弹窗正文滚动，确认、取消和关闭按钮保持可见；组件使用默认尺寸，窗口适配工作区和 DPI。
- 修复软件更新取消、校验失败和替换失败恢复的测试问题，补充 RDP 白名单、回滚和重连确认异常路径验证。

### 验证与适用范围

- 前端检查与 20 项测试、Go 测试、静态检查、桥接契约及生产构建通过。
- 小窗口页面排版 366 组、弹窗/下拉菜单/日志详情 234 组模拟交互通过。它们使用模拟后端，不代表真实服务器断连或重启验收。
- 面向 Windows Server 2016/2019/2022/2025 Desktop Experience 和 Windows 远程 VPS；Server Core、会话 0 和域控制器不支持。受管策略或无法确定有效规则时保守拒绝变更。
- **目前没有 Windows Server/VPS 测试机，实机断连、重连、SYSTEM 回滚任务和重启验收尚未执行。远程变更前应保留服务商独立控制台恢复入口，云安全组需另行核对。**
- [详细验证报告](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/zh-Hans/validation-vps-2026-10-23.md)

### Changes (English)

- Added RDP source-IP allowlists, independent timeout rollback, reconnect confirmation, listener/firewall checks and session event records.
- Added Windows system-event analysis with restart/crash evidence, original XML and JSON export; retained all client history/credential/cache cleanup functions.
- Added account rename and simplified menus, result messages and collapsible activity logs. Improved small-window layouts and scrollable confirmations.
- Passed automated Go/frontend checks and 600 simulated layout/overlay cases. Windows Server/VPS live acceptance remains unperformed because no test server is available; retain an independent console recovery channel before remote changes.

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
