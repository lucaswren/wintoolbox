"""Capture the built WinToolBox UI in all supported languages using sample data."""

import json
from io import BytesIO
from pathlib import Path
from urllib.parse import urlsplit

from PIL import Image, ImageDraw, ImageFont
from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "frontend/dist"
OUT = ROOT / "docs/images"
URL = "http://wintoolbox.test/"
SCALE = 2

MOCK_JS = r"""
(() => {
  if (!new URLSearchParams(location.search).has('first-run')) localStorage.setItem('wintoolbox.helpSeen', '1');
  const status = {
    overview: {
      hostname: 'WIN-SERVER01', osName: 'Windows Server 2022 Datacenter', osBuild: '20348.3807', arch: 'x64',
      manufacturer: 'Dell Inc.', model: 'PowerEdge R740', board: '0H21HH', bios: '2.18.1',
      cpu: 'Intel Xeon Silver 4210', cpuCores: 20, memoryTotalGB: 64, memoryAvailGB: 41.2,
      memoryModules: '4 x 16GB DDR4', resolution: '1920x1080', ips: ['10.0.0.12'],
      disks: ['C: 120GB free / 500GB'], diskSpace: [{drive:'C:', totalGiB:500, freeGiB:120, system:true}, {drive:'D:', totalGiB:100, freeGiB:8, system:false}], physicalDisks: ['Samsung SSD 980 1TB'],
      gpus: ['Microsoft Basic Display Adapter'], activated: true, activationStatus: '已激活'
    },
    accounts: [
      { name: 'Administrator', enabled: true, enabledUnknown: false, admin: true, adminUnknown: false, current: true },
      { name: 'ops', enabled: true, enabledUnknown: false, admin: false, adminUnknown: false, current: false }
    ],
    rdpEnabled: true, rdpPort: 3389, rdpAvailable: true, rdpNLA: false, rdpNLAUnknown: false,
    updateDisabled: false, updateUnknown: false,
    updateDetail: 'NoAutoUpdate=0 · wuauserv=auto · UsoSvc=auto · DoSvc=auto',
    firewallSummary: 'all-on', firewallDomain: '\u5f00', firewallPrivate: '\u5f00', firewallPublic: '\u5f00',
    firewallAllOn: true, firewallAllOff: false,
    pingBlocked: false, pingIPv4Blocked: false, pingIPv6Blocked: false, pingState: 'enabled',
    riskPortsBlocked: false, riskPortsPartial: false, riskPortsUnknown: false,
    riskPortsDetail: '\u9ad8\u5371\u7aef\u53e3\u9884\u8bbe\u672a\u542f\u7528',
    defenderDisabled: false, defenderUnknown: false, defenderDetail: 'DisableRealtimeMonitoring=false',
    timeText: '2026-09-11 12:00:00  ·  China Standard Time  ·  NTP time.windows.com',
    timeZone: 'China Standard Time', timeUnknown: false,
    timeZones: [{ id: 'China Standard Time', label: '(UTC+08:00) Beijing' }],
    ntpServer: 'time.windows.com', warnings: [],
    lockoutDisabled: true, lockoutUnknown: false, lockoutDetail: 'Threshold=0',
    lockoutThreshold: 0, lockoutDuration: 30, lockoutWindow: 30,
    guestExists: true, guestEnabled: true, guestUnknown: false, guestDetail: 'Guest enabled',
    autoLogonEnabled: true, autoLogonUnknown: false, autoLogonDetail: 'AutoAdminLogon=1',
    passwordMinLength: 8, passwordComplexity: false, passwordComplexityUnknown: false,
    passwordUnknown: false, passwordPolicyDetail: 'MinLen=8',
    smb1Disabled: false, smb1Unknown: false, smb1Detail: 'SMBv1 enabled',
    winrmHardened: false, winrmUnknown: false, winrmDetail: 'WinRM not hardened',
    anonymousOK: false, anonymousUnknown: false, anonymousDetail: 'Anonymous enum allowed'
  };
  const appInfo = { name: 'WinToolBox', version: 'v1.2.0' };
  const rules = [{ name: 'WinToolbox-Allow-8080', port: '8080', enabled: true }];
  const history = [{ kind: 'mru', host: '10.0.0.8', username: 'Administrator', source: 'MRU', detail: '', sid: '' }, { kind: 'credential', host: 'TERMSRV/server.example', username: 'ops', source: 'Credential Manager', detail: 'TERMSRV/server.example', sid: '' }, { kind: 'cache', host: 'Cache', username: '', source: 'mstsc', detail: 'Bitmap cache', sid: '' }];
  const upd = { currentVersion: 'v1.2.0', latestVersion: 'v1.2.0', hasUpdate: false, assetName: '', assetURL: '', assetSize: 0, assetSHA256: '', notes: '', downloaded: false, downloadPath: '', verified: false, error: '' };
  const ok = (v) => Promise.resolve(v);
  const noop = () => ok(null);
  const rdpSafety = { session: { id: 2, known: true, remote: true }, pending: false, transaction: '', deadline: '', phase: 'idle', kind: '', addresses: [], port: 3389, listening: true, service: 'Running', profiles: 'Domain=True/Block; Private=True/Block; Public=True/Block', rules: [{ name: 'RDP-TCP', action: 'Allow', protocol: 'TCP', ports: '3389', addresses: 'Any', source: 'Local', profile: 'Any' }], warnings: [], canConfirm: false };
  window.__rdpSafety = rdpSafety;
  const app = {
    OpenDocumentation: (language) => { window.__documentationLanguage = language; return noop(); },
    GetSystemHealth: () => ok({bootTime:'2026-10-22T00:00:00Z', services:[{name:'TermService',state:'running'},{name:'W32Time',state:'stopped'},{name:'Spooler',state:'missing'}], recentEvents:[{id:41,time:'2026-10-23T01:00:00Z'}], eventsPartial:false}),
    CopyHealthSummary: (text) => { window.__copiedSummary = text; return noop(); },
    CheckNetwork: (target, port, count, source) => ok({dns:'resolved', addresses:['203.0.113.2'], address:'203.0.113.2', ping:'no-reply', sent:4, received:0, averageMs:0, tcp:port ? 'connected' : 'skipped', port, source, tcpStats:{attempts:count, successes:count, minMs:12, averageMs:15, maxMs:18}}),
    TraceNetwork: (target, rounds, source) => ok({status:'complete', address:'203.0.113.2', reached:true, source, rounds, roundsCompleted:rounds, hops:[{number:1, addresses:['192.0.2.1'], sent:4, received:3, averageMs:2}, {number:2, addresses:[], sent:4, received:0, averageMs:0}, {number:3, addresses:['203.0.113.2'], sent:4, received:4, averageMs:14}]}),
    GetNetworkSources: () => ok([{address:'192.0.2.20',interface:'Ethernet'}, {address:'2001:db8::20',interface:'Ethernet'}]),
    CheckPublicEgress: (source) => {
      window.__egressSource = source;
      return ok([
        {family:4, endpoint:'api.ipify.org', source:'192.0.2.20', publicIp:'203.0.113.20', status:'ok'},
        {family:6, endpoint:'api6.ipify.org', source:source ? '' : '2001:db8::20', publicIp:source ? '' : '2001:db8::20', status:source ? 'skipped-family' : 'ok'}
      ]);
    },
    CheckLocalListener: () => ok('loopback-only'),
    GetAppInfo: () => ok(appInfo), GetStatus: () => ok(status), GetModuleStatus: () => ok(status), CancelAppUpdateDownload: noop,
    GetOverviewDetail: () => ok({ memoryModules: status.overview.memoryModules, physicalDisks: status.overview.physicalDisks, gpus: status.overview.gpus, activated: true, activationStatus: '已激活' }),
    GetTimeZones: () => ok(status.timeZones), GetFirewallRules: () => ok(rules), GetRdpHistory: () => ok(history),
    CheckAppUpdate: () => ok(upd), DownloadAppUpdate: () => ok(upd), GetAppUpdateInfo: () => ok(upd), ApplyAppUpdate: noop,
    ChangeAccountPassword: noop, SetAccountEnabled: noop, SetAccountAdmin: noop,
    DisableAccountLockout: noop, EnableAccountLockout: noop, SetAccountLockoutPolicy: noop,
    DisableGuestAccount: noop, DisableAutoLogon: noop, EnablePasswordPolicy: noop,
    ChangeRdpPort: noop, ToggleRdp: noop, SetRdpNLA: noop,
    GetRdpSafetyStatus: () => ok(JSON.parse(JSON.stringify(rdpSafety))),
    ApplyRdpWhitelist: (addresses) => { Object.assign(rdpSafety, { pending: true, transaction: 'mock-transaction', deadline: new Date(Date.now()+300000).toISOString(), phase: 'awaiting-confirmation', kind: 'whitelist', addresses: addresses.split(/[,;\s]+/), canConfirm: false }); return noop(); },
    ConfirmRdpSafety: () => { if (!rdpSafety.canConfirm) return Promise.reject('Reconnect first'); Object.assign(rdpSafety, {pending: false, phase: 'confirmed'}); return noop(); },
    RollbackRdpSafety: () => { Object.assign(rdpSafety, {pending: false, phase: 'rolled-back'}); return noop(); },
    GetRdpLoginReport: () => ok({collectedAt:'2026-10-27T02:00:00Z',startTime:'2026-10-20T02:00:00Z',truncated:false,summary:{total:2,logins:0,failures:1,networkFailures:0,disconnects:1,reconnects:0},sources:[{address:'203.0.113.7',logins:0,failures:1,networkFailures:0,lastSeen:'2026-10-27T01:00:00Z'}],findings:[],events:[{recordId:'1',time:'2026-10-27T01:00:00Z',id:24,provider:'Microsoft-Windows-TerminalServices-LocalSessionManager',channel:'Microsoft-Windows-TerminalServices-LocalSessionManager/Operational',user:'Administrator',address:'203.0.113.7',session:'2',category:'disconnect',meaning:'会话断开记录；断开不等于注销',detail:'Session disconnected (simulated)',xml:'<Event><System><EventID>24</EventID></System></Event>',data:[{name:'SessionID',value:'2'}]},{recordId:'2',time:'2026-10-27T00:59:00Z',id:4625,provider:'Microsoft-Windows-Security-Auditing',channel:'Security',user:'Administrator',address:'203.0.113.7',logonType:'10',status:'0xc000006d',subStatus:'0xc000006a',category:'failure',meaning:'远程交互式登录失败',reason:'密码不正确',detail:'Bad password (simulated)',xml:'<Event><EventData><Data Name="SubStatus">0xc000006a</Data></EventData></Event>',data:[{name:'SubStatus',value:'0xc000006a'}]}],warnings:['—']}),
    GetSystemEventReport: () => ok({collectedAt:'2026-10-23T02:00:00Z',startTime:'2026-10-16T02:00:00Z',truncated:false,summary:{total:3,boot:1,requests:1,unexpected:1,bugchecks:0},warnings:['统计为事件条数，不是重启次数；41、6008、1001 可能描述同一次故障。前后十分钟的关联仅为排查线索。'],events:[{recordId:'103',time:'2026-10-23T01:00:00Z',provider:'Microsoft-Windows-Kernel-Power',id:41,level:1,computer:'VPS-DEMO',message:'The system has rebooted without cleanly shutting down first. Simulated data.',xml:'<Event><System><EventID>41</EventID></System></Event>',data:[{name:'BugcheckCode',value:'0'}],category:'unexpected',meaning:'检测到未正常关机后的启动；此事件本身不能确定是断电、宿主机重置、蓝屏还是系统卡死。'},{recordId:'102',time:'2026-10-23T00:59:50Z',provider:'Microsoft-Windows-Kernel-General',id:12,computer:'VPS-DEMO',message:'System started (simulated)',data:[],category:'boot',meaning:'内核记录系统启动。'},{recordId:'101',time:'2026-10-22T03:00:00Z',provider:'User32',id:1074,computer:'VPS-DEMO',message:'A restart was requested (simulated)',data:[{name:'param1',value:'shutdown.exe'}],category:'request',meaning:'已记录进程或用户发起的关机/重启请求；这不能单独证明操作正常完成。'}],findings:[{recordId:'103',time:'2026-10-23T01:00:00Z',kind:'unexpected',stopCode:'',conclusion:'检测到未正常关机后的启动；此事件本身不能确定是断电、宿主机重置、蓝屏还是系统卡死。',nextSteps:['核对故障前后的监控和服务商记录。'],evidence:['103','102']}]}),
    ClearRdpHistory: noop, ClearRdpHistoryByKind: noop, DeleteRdpHistoryEntry: noop,
    AllowFirewallPort: noop, RemoveFirewallPort: noop, ClearFirewallAllowRules: noop,
    SetFirewallEnabled: noop, DisablePing: noop, EnablePing: noop, BlockRiskPorts: noop, UnblockRiskPorts: noop,
    DisableSMBv1: noop, HardenWinRM: noop, RestrictAnonymous: noop,
    ApplyTimeZone: noop, SaveNTPServer: noop, SyncNTP: noop, TestNTPServer: () => ok('ok'),
    LockPC: noop, RestartPC: noop, ShutdownPC: noop, AbortPower: noop,
    DisableUpdate: noop, EnableUpdate: noop, DisableDefender: noop, EnableDefender: noop
  };
  window.go = { ui: { App: app } };
})();
"""


PAGES = [
    ("00-help", "系统信息", []),
    ("01-overview", "系统信息", []),
    ("02-local-account", "账户管理", ["禁用来宾账户", "关闭自动登录", "应用密码策略", "一键开启锁定"]),
    ("02-local-account-password", "账户管理", ["修改密码", "设为管理员"]),
    ("03-remote-desktop", "远程桌面", ["保存端口", "关闭", "强制 NLA"]),
    ("04-firewall", "防火墙", ["全部开启", "放行", "一键禁 ping", "拦高危入站端口"]),
    ("05-security-harden", "安全设置", ["禁用 SMBv1", "关闭并拦截 WinRM", "禁匿名枚举"]),
    ("06-defender", "防病毒", ["关闭实时防护", "启用实时防护"]),
    ("07-time-sync", "日期和时间", ["应用时区", "保存 NTP", "立即同步", "测试 NTP"]),
    ("08-power", "关机重启", ["锁定", "重启", "关机", "取消关机/重启"]),
    ("09-windows-update", "Windows 更新", ["关闭更新", "还原关闭前配置"]),
    ("10-app-update", "软件更新", ["检查更新", "下载更新", "安装并重启"]),
    ("11-system-logs", "系统日志", []),
    ("12-network-check", "网络检查", []),
    ("12-network-advanced", "网络检查", []),
    ("90-hardening-nav", "系统信息", []),
]


def serve(route):
    asset = (DIST / (urlsplit(route.request.url).path.lstrip("/") or "index.html")).resolve()
    if asset.is_relative_to(DIST) and asset.is_file():
        route.fulfill(path=str(asset))
    else:
        route.fulfill(status=404, body="Not found")


def screenshot(page, output, targets):
    marks = []
    for number, target in enumerate(targets, 1):
        expect(target).to_be_visible()
        expect(target).to_be_in_viewport(ratio=0.95)
        bounds = target.bounding_box()
        marks.append((bounds["x"] + 2, bounds["y"] + 2, str(number)))
    png = page.screenshot(animations="disabled")

    with Image.open(BytesIO(png)) as image:
        draw = ImageDraw.Draw(image)
        font = ImageFont.load_default(size=25)
        for x, y, number in marks:
            x, y = x * SCALE, y * SCALE
            draw.ellipse((x - 19, y - 19, x + 19, y + 19), fill="#a82434", outline="white", width=2)
            draw.text((x, y), number, font=font, fill="white", anchor="mm")
        image.save(output, optimize=True)


def capture(browser, language):
    catalog = json.loads((ROOT / f"frontend/src/i18n/{language}.json").read_text(encoding="utf-8"))
    output = OUT / language
    output.mkdir(parents=True, exist_ok=True)
    page = browser.new_page(viewport={"width": 1280, "height": 960}, device_scale_factor=SCALE)
    errors = []
    page.on("pageerror", lambda error: errors.append(str(error)))
    page.route(URL + "**", serve)
    page.add_init_script(MOCK_JS)
    page.add_init_script(f"localStorage.setItem('wintoolbox.language', {json.dumps(language)})")
    page.goto(URL)
    expect(page.locator("html")).to_have_attribute("lang", language)

    def menu(key):
        return page.get_by_role("menuitem", name=catalog[key], exact=True)

    main = page.locator(".app-main")

    def button(key):
        return main.get_by_role("button", name=catalog[key], exact=True)

    for filename, title, buttons in PAGES:
        height = {
            "01-overview": 1140,
            "02-local-account": 1280,
            "02-local-account-password": 800,
            "04-firewall": 1280,
            "05-security-harden": 920,
            "12-network-check": 1100,
            "12-network-advanced": 930,
            "90-hardening-nav": 1140,
        }.get(filename, 680)
        page.set_viewport_size({"width": 1280, "height": height})
        menu(title).click()
        if filename == "00-help":
            page.get_by_role("button", name=catalog["帮助"], exact=True).click()
            expect(page.get_by_role("dialog", name=catalog["新手指引"])).to_be_visible()
        elif filename == "02-local-account":
            page.locator("#tab-policies").click()
        elif filename == "02-local-account-password":
            page.locator("#tab-users").click()
        elif filename == "11-system-logs":
            button("读取并分析").click()
            expect(main.get_by_role("button", name=catalog["查看详情"]).first).to_be_visible()
            expect(main.locator(".el-loading-mask")).to_have_count(0)
        elif filename == "12-network-check":
            main.get_by_role("textbox", name=catalog["目标地址"]).fill("example.com")
            main.get_by_role("textbox", name=catalog["TCP 端口（选填）"]).fill("443")
            button("开始检查").click()
            expect(main.get_by_text(catalog["未收到 Ping 回应，但 TCP 端口可连通。目标可能关闭了 Ping 回应。"])).to_be_visible()
            button("查询公网出口").click()
            expect(main.get_by_text("203.0.113.20", exact=True)).to_be_visible()
        elif filename == "12-network-advanced":
            button("高级诊断").click()
            main.locator(".source-select .el-select__wrapper").click()
            page.get_by_role("option", name="192.0.2.20 · Ethernet").click()
            button("运行 MTR").click()
            expect(main.get_by_text(catalog["目标已回应。"], exact=True)).to_be_visible()
            main.evaluate("el => el.scrollTop = el.scrollHeight")
            expect(button("运行 MTR")).to_be_in_viewport()
        elif filename in ("01-overview", "90-hardening-nav"):
            expect(button("预览故障摘要")).to_be_enabled()
            expect(main.get_by_text(catalog["空间偏低"])).to_be_visible()
        expect(main).to_have_attribute("aria-busy", "false")
        targets = [button(key) for key in buttons]
        if filename == "02-local-account-password":
            users = page.locator("#pane-users")
            targets = [
                users.get_by_role("combobox"),
                *[users.locator('input[type="password"]').nth(i) for i in range(2)],
                *targets,
            ]
        elif filename == "03-remote-desktop":
            targets.insert(0, main.get_by_role("spinbutton"))
        elif filename == "90-hardening-nav":
            targets = [menu(key) for key in ("账户管理", "远程桌面", "防火墙", "安全设置", "Windows 更新")]
        page.mouse.move(235, 0)
        if filename == "12-network-advanced":
            main.locator(".el-collapse").screenshot(path=str(output / f"{filename}.png"), animations="disabled")
        else:
            screenshot(page, output / f"{filename}.png", targets)
        if filename == "00-help":
            page.get_by_role("button", name=catalog["知道了"], exact=True).click()
            expect(page.get_by_role("dialog", name=catalog["新手指引"])).not_to_be_visible()
        print(language, filename, flush=True)
    if errors:
        raise RuntimeError(f"{language}: {errors}")
    page.close()


def main():
    if not (DIST / "index.html").is_file():
        raise SystemExit("Build the frontend first: npm --prefix frontend run build")
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        for language in ("en", "zh-Hans", "zh-Hant"):
            capture(browser, language)


if __name__ == "__main__":
    main()
