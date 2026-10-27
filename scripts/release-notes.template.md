## {{VERSION}}

### 更新内容

- 精简远程登录和系统日志分析页面：默认只显示操作按钮、简短统计与关键记录，给表格留出更多空间。
- 时间范围、条数、分类和搜索统一放入默认收起的“筛选与范围”；有筛选条件时显示数量，避免收起后忽略正在生效的条件。
- 采集时间、完整错误及使用限制放入“日志说明”；读取失败、保留旧结果和日志不完整仍显示简短提示。
- 事件来源、ID、失败原因和状态码通过“查看详情”查看；原文、结构化字段、XML 及关联证据分别点击展开。来源汇总、重复失败分析、搜索、导出、清空与重读均保留。
- 优化小窗口布局：工具栏自动换行、筛选面板自适应分列、宽表保留横向滚动、详情按钮固定右侧；组件继续使用默认尺寸。
- 补齐英文界面标签。**清空结果仍只影响当前显示，不删除 Windows 原始日志**；客户端历史、凭据和缓存清理等现有功能保留。

### 验证与适用范围

- 前端检查与 20 项测试、Go 测试、静态检查、桥接契约及生产构建通过。
- 11 页模拟回归、两页 168 组多语言/四种窗口/日志展开收起专项检查、234 组弹出层检查，以及最终 42 组最小窗口复验通过；无控件遮挡或页面错误。验证使用模拟后端，不代表真实服务器断连或重启验收。
- 面向 Windows Server 2016/2019/2022/2025 Desktop Experience 和 Windows 远程 VPS；Server Core、会话 0 和域控制器不支持。受管策略或无法确定有效规则时保守拒绝变更。
- **目前没有 Windows Server/VPS 测试机，实机断连、重连、SYSTEM 回滚任务和重启验收尚未执行。远程变更前应保留服务商独立控制台恢复入口，云安全组需另行核对。**
- 更新前请先确认或恢复待处理的 RDP 保护变更。
- [详细验证报告](https://github.com/lucaswren/wintoolbox/blob/{{VERSION}}/docs/zh-Hans/validation-vps-2026-10-23.md)

### Changes (English)

- Simplified remote sign-in and system-log pages to show actions, compact counts and key records by default.
- Collapsed collection/filter settings and explanations; active filters remain indicated, and brief failure/incomplete-result notices remain visible.
- Kept all analysis, search, export, clear and reread functions. Original messages, fields, XML and related evidence expand on demand in event details. Clearing results retains Windows event logs.
- Improved small-window wrapping, scrolling and fixed detail buttons; filled missing English labels. Passed frontend/Go checks, 11-page regression, 168 log-page cases, 234 overlay cases and 42 final small-window cases. Server/VPS live acceptance remains unperformed.

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
