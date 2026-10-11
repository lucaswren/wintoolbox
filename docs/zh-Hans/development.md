[English](../development.md) · [繁體中文](../zh-Hant/development.md) · [简体中文](../zh-Hans/development.md)

# 开发和验收规范

环境：Windows amd64、Go 1.27+、Node.js 22.13+、Windows PowerShell 5.1 或 PowerShell 7。

## 检查流程

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

后端导出方法、参数或 JSON DTO 有变化时，重新生成 `frontend/src/api/wails.d.ts`。CI 使用 `-check` 验证契约未过期；前端 API 门面的类型检查会检测方法名、参数数量和返回类型。当前未将所有 Vue 页面迁移到 TypeScript。

前端统一 Prettier 格式及 ESLint Vue 规则，Go 使用 gofmt。依赖以 package-lock.json 固定，CI 与默认构建使用 npm ci。构建产物写入 artifacts/releases，根目录 WinToolBox.exe 作为本地运行文件保留。

## 操作契约

- 状态探测失败表示 unknown；不能使用布尔零值替代未知。
- 所有配置备份字段写入成功后才设置 Saved；恢复前先读取全部所需字段。
- 部分操作成功后仍可能返回错误，因此前端无论成功失败都复查相关模块。
- 后端按资源序列化写操作；RDP、防火墙、WinRM 共用网络资源锁。
- 状态查询按模块合并重叠任务。取消感知的模块在全部调用者离开后取消共享命令；不能取消的原生调用继续完成，但同一探测不会重复启动。系统命令并发最多6个。
- Windows Update 的“还原关闭前配置”以原配置为目标；原来已禁用时不强制开启。
- Defender 的“启用实时防护”明确以开启为目标，不承诺还原之前的关闭状态。
- 重启和关机会强制关闭应用，确认提示必须说明并提醒保存工作。
- 日志仅记录操作摘要和错误，不记录密码；最多500条，可导出。

## 自更新

检查/下载任务锁与状态锁分离，下载期间状态读取和取消不被网络请求阻塞。下载进度经现有桥接轮询，SHA256 匹配指定安装包；预发布版本按核心版本及预发布标识排序。软件更新页可关闭自动下载，偏好在本地保存。

安装包暂存到当前用户、管理员和SYSTEM可访问的随机目录。更新器退出等待最长2分钟，复制前后检查哈希，用 File.Replace 保留 `.previous` 文件；启动失败或5秒内退出时还原旧文件。日志在用户缓存 WinToolbox/updates/last-install.log，暂存与旧文件保留用于故障诊断。

进程存活5秒是有限的启动检测，不是 UI 就绪协议。真实更新链路、UAC 和杀毒软件干预须在虚拟机验收。

## Windows 虚拟机验收

1. 记录系统原值与还原点，分别验证每个支持的 Windows 版本及域策略环境。
2. 注入备份写失败、权限不足、状态超时，确认不误显示成功或关闭。
3. RDP重启失败后确认界面显示实际新端口和失败说明。
4. Update在原配置禁用时关闭再还原，确认原值一致且报告还原成功。
5. 对更新器注入哈希错误、复制失败、启动失败，确认旧exe可恢复，日志可追踪。
6. 检查960×600、1120×720及125%/150%/200%缩放，侧栏与操作按钮均可访问。
7. 检查切页草稿保留、未知状态、下载取消、日志导出，以及后台更新与系统操作同时进行的反馈。

自动化测试使用模拟对象和临时目录，不修改真实系统配置；不能替代上述虚拟机验收。

## 模拟界面验收

```powershell
python -m pip install -r docs/requirements.txt
python -m playwright install chromium
# 在一个终端运行 npm run dev，另一个终端执行：
python scripts/ui-smoke.py
```

脚本复用文档模拟后端，检查十个页面、960×600/1120×720窗口与草稿保留；截图输出到 artifacts/ui，所有系统写操作均为模拟函数。
