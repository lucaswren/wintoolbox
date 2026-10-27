"""Read-only frontend smoke test against the docs' simulated Wails backend.

Run with a Vite server on http://127.0.0.1:5173. Requires Playwright, as does
docs/capture_screens.py. No call reaches a real Windows administrative operation.
Set WINTOOLBOX_LAYOUT_AUDIT=1 to check all pages, sizes and languages, or stress
to check only stacked failure/recovery banners with an expanded activity log.
"""
import ast
import os
from urllib.parse import urlsplit
from pathlib import Path

from playwright.sync_api import sync_playwright, expect

ROOT = Path(__file__).resolve().parents[1]
tree = ast.parse((ROOT / "docs/capture_screens.py").read_text(encoding="utf-8"))
mock = next(ast.literal_eval(node.value) for node in tree.body if isinstance(node, ast.Assign)
            and any(isinstance(t, ast.Name) and t.id == "MOCK_JS" for t in node.targets))
out = ROOT / "artifacts/ui"
out.mkdir(parents=True, exist_ok=True)

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True, timeout=30000)
    page = browser.new_page(viewport={"width": 960, "height": 600})
    errors = []
    page.on("pageerror", lambda error: errors.append(str(error)))
    page.add_init_script(mock)
    page.add_init_script("localStorage.setItem('wintoolbox.language', 'zh-Hans')")
    url = os.environ.get("WINTOOLBOX_UI_URL", "http://127.0.0.1:5173")
    if os.environ.get("WINTOOLBOX_UI_DIST") == "1":
        # Fulfill the compiled UI directly; no local server/proxy is required.
        dist = (ROOT / "frontend/dist").resolve()
        def serve(route):
            asset = (dist / (urlsplit(route.request.url).path.lstrip("/") or "index.html")).resolve()
            if asset.is_relative_to(dist) and asset.is_file():
                route.fulfill(path=str(asset))
            else:
                route.fulfill(status=404, body="Not found")
        page.route("http://wintoolbox.test/**", serve)
        url = "http://wintoolbox.test/"
    page.goto(url, wait_until="networkidle")
    expect(page.locator('.op-log__body')).to_have_count(0)
    assert page.locator('.op-log').bounding_box()['height'] <= 48, 'Collapsed log must occupy only its toolbar'
    page.locator('.op-log').get_by_role('button',name='展开',exact=True).click()
    expect(page.locator('.op-log__body')).to_be_visible()
    page.locator('.op-log').get_by_role('button',name='收起',exact=True).click()
    expect(page.locator('.op-log__body')).to_have_count(0)
    assert page.locator(".side-menu .el-menu-item").first.evaluate("el => el.getBoundingClientRect().height") == 40, "Navigation items must use compact height"
    page.get_by_role("menuitem", name="账户管理").click()
    page.locator('[data-view="account"]').wait_for(state="visible")
    rename = page.locator('.rename-action')
    button = rename.get_by_role('button', name='修改用户名', exact=True).bounding_box()
    note = rename.locator('p').bounding_box()
    assert note['y'] >= button['y'] + button['height'], 'Rename note must be below the button'
    assert abs(note['x'] - button['x']) <= 1, 'Rename note and button must align'
    page.screenshot(path=str(out / 'account-rename-960.png'))
    page.get_by_role("tab", name="安全策略").click()
    field = page.locator(".el-tab-pane:visible .el-input-number input").first
    field.fill("12")
    field.press("Tab")
    page.get_by_role("menuitem", name="防火墙").click()
    page.get_by_role("menuitem", name="账户管理").click()
    assert page.get_by_role("tab", name="安全策略").get_attribute("aria-selected") == "true"
    assert page.locator(".el-tab-pane:visible .el-input-number input").first.input_value() == "12"
    page.screenshot(path=str(out / "account-960.png"))
    page.get_by_role("tab", name="用户管理").click()
    for title, key in [("系统信息", "overview"), ("账户管理", "account"), ("远程桌面", "rdp"), ("防火墙", "firewall"), ("安全设置", "harden"), ("Microsoft Defender", "defender"), ("日期和时间", "time"), ("关机重启", "power"), ("系统日志", "systemlogs"), ("Windows 更新", "update"), ("软件更新", "selfupdate")]:
        item = page.get_by_role("menuitem", name=title, exact=True)
        item.scroll_into_view_if_needed()
        item.click()
        page.locator(f'[data-view="{key}"]').wait_for(state="visible")
        page.wait_for_timeout(250)
        assert page.locator(".app-main").evaluate("el => el.scrollWidth <= el.clientWidth + 2"), title
    page.get_by_role("menuitem", name="防火墙").click()
    page.locator('[data-view="firewall"]').wait_for(state="visible")
    page.wait_for_timeout(300)
    page.screenshot(path=str(out / "firewall-960.png"))
    page.get_by_role("menuitem", name="远程桌面").click()
    page.get_by_role("tab", name="访问限制", exact=True).click()
    assert not page.get_by_text("TermService:", exact=False).is_visible(), "Raw diagnostics must start collapsed"
    page.get_by_role("tab", name="客户端记录", exact=True).click()
    page.get_by_role("button", name="清理全部记录", exact=True).wait_for(state="visible")
    page.get_by_placeholder("搜索主机 / 用户名 / 来源 / 详情").fill("TERMSRV")
    expect(page.locator('[data-view="rdp"] .el-tab-pane:visible .el-table__body tbody tr')).to_have_count(1)
    page.get_by_placeholder("搜索主机 / 用户名 / 来源 / 详情").fill("")
    expect(page.locator('[data-view="rdp"] .el-tab-pane:visible .el-table__body tbody tr')).to_have_count(3)
    page.evaluate("() => { window.__clientCleanup=[]; for (const name of ['ClearRdpHistory','ClearRdpHistoryByKind','DeleteRdpHistoryEntry']) { const fn=window.go.ui.App[name]; window.go.ui.App[name]=(...args)=>{window.__clientCleanup.push({name,args}); return fn(...args)} } }")
    page.get_by_role("button",name="删除",exact=True).first.click()
    page.locator('.el-message-box').get_by_role("button",name="取消",exact=True).click()
    assert page.evaluate("window.__clientCleanup.length") == 0, "Cancelled cleanup must not write"
    page.get_by_role("button",name="删除",exact=True).first.click()
    page.locator('.el-message-box').get_by_role("button",name="确定",exact=True).click()
    page.wait_for_function("window.__clientCleanup.length === 1")
    page.locator('[data-view="rdp"] .el-tab-pane:visible .el-select').click()
    page.get_by_role('option',name='凭据',exact=True).click()
    page.get_by_role("button",name="清理当前类型",exact=True).click()
    page.locator('.el-message-box').get_by_role("button",name="确定",exact=True).click()
    page.wait_for_function("window.__clientCleanup.some(x=>x.name==='ClearRdpHistoryByKind'&&x.args[0]==='credential')")
    page.get_by_role("button",name="清理全部记录",exact=True).click()
    page.locator('.el-message-box').get_by_role("button",name="确定",exact=True).click()
    page.wait_for_function("window.__clientCleanup.some(x=>x.name==='ClearRdpHistory')")
    page.wait_for_timeout(250)
    page.screenshot(path=str(out / "rdp-client-history-960.png"))
    result = page.get_by_test_id('operation-status')
    expect(result).to_contain_text('已清理 RDP 连接记录')
    assert result.bounding_box()['height'] <= 48, 'Completed operation must fit on one short line'
    assert '确认清理' not in result.inner_text(), 'Confirmation details belong in the log'
    result.get_by_role('button',name='查看日志',exact=True).click()
    assert page.locator('.op-log__body').bounding_box()['height'] == 180
    assert '确认清理本机 RDP 客户端连接记录' in page.locator('.op-log').inner_text()
    page.locator('.op-log').get_by_role('button',name='收起',exact=True).click()
    expect(result).to_be_hidden(timeout=5000)
    expect(page.locator('.op-log__body')).to_have_count(0)
    with page.expect_download() as log_info:
        page.locator('.op-log').get_by_role('button',name='导出',exact=True).click()
    log_info.value.save_as(str(out / 'activity-log-fixture.txt'))
    exported_log = (out / 'activity-log-fixture.txt').read_text(encoding='utf-8')
    assert '已清理 RDP 连接记录' in exported_log and '确认清理本机 RDP 客户端连接记录' in exported_log, 'Collapsed log must retain full exportable details'
    page.evaluate("() => { window.__savedClear=window.go.ui.App.ClearRdpHistory; window.go.ui.App.ClearRdpHistory=()=>Promise.reject('Injected cleanup failure: '+ 'full diagnostic details '.repeat(20)); }")
    page.get_by_role('button',name='清理全部记录',exact=True).click()
    page.locator('.el-message-box').get_by_role('button',name='确定',exact=True).click()
    page.locator('[data-testid="operation-status"].failed').wait_for(state='visible')
    assert result.bounding_box()['height'] <= 48, 'Failure details must not expand the header'
    assert 'Injected cleanup failure' not in result.inner_text()
    expect(page.locator('.op-log__body')).to_have_count(0)
    result.get_by_role('button',name='查看日志',exact=True).click()
    assert 'Injected cleanup failure' in page.locator('.op-log').inner_text()
    page.locator('.op-log').get_by_role('button',name='收起',exact=True).click()
    expect(page.locator('.op-log__body')).to_have_count(0)
    page.locator('.el-message-box').wait_for(state='hidden')
    page.wait_for_timeout(3300)
    expect(result).to_be_visible()
    page.screenshot(path=str(out / 'operation-failure-960.png'))
    page.evaluate("() => { window.go.ui.App.ClearRdpHistory=window.__savedClear; }")
    result.get_by_role('button',name='关闭提示',exact=True).click()
    result.wait_for(state='hidden')

    page.get_by_role("tab",name="访问限制",exact=True).click()
    page.wait_for_timeout(250)
    page.screenshot(path=str(out / "rdp-simple-960.png"))
    page.get_by_placeholder("每行一个公网 IP 或网段，例如 203.0.113.8").fill("192.0.2.1/32")
    page.get_by_role("button", name="保存访问限制").click()
    page.locator('.el-message-box').get_by_role("button", name="确定", exact=True).click()
    keep = page.get_by_role("button", name="重新连接成功，保留配置")
    keep.wait_for(state="visible")
    assert keep.is_disabled(), "An existing session must not confirm a protected change"
    page.get_by_role("menuitem", name="系统信息").click()
    assert keep.is_visible(), "Pending recovery must remain visible after navigation"
    page.evaluate("window.__rdpSafety.canConfirm = true")
    page.get_by_role("menuitem", name="远程桌面").click()
    page.get_by_role("button", name="检查连接状态").click()
    page.wait_for_function("!Array.from(document.querySelectorAll('button')).find(x=>x.textContent.includes('重新连接成功，保留配置')).disabled")
    banner = page.get_by_test_id("rdp-safety-banner").bounding_box()
    button = keep.bounding_box()
    page.screenshot(path=str(out / "rdp-pending-960.png"))
    title = page.get_by_test_id("rdp-safety-banner").locator('.el-alert__title').bounding_box()
    assert title["y"] >= banner["y"] and button["y"] + button["height"] <= banner["y"] + banner["height"], f"Recovery controls clipped: {banner}, {button}, {title}"
    keep.click()
    page.locator('.el-message-box').get_by_role("button", name="确定", exact=True).click()
    keep.wait_for(state="hidden")
    page.get_by_role("tab",name="登录记录",exact=True).click()
    page.get_by_role("button", name="读取记录").click()
    page.get_by_test_id("login-event-table").get_by_text("203.0.113.7", exact=True).first.wait_for(state="visible")
    page.wait_for_timeout(250)
    page.screenshot(path=str(out / "rdp-events-960.png"))
    # Failure and cancellation paths must keep recovery and entered addresses intact.
    page.get_by_role("tab",name="访问限制",exact=True).click()
    save = page.get_by_role("button", name="保存访问限制", exact=True)
    check = page.get_by_role("button", name="检查连接状态", exact=True)
    address = page.get_by_placeholder("每行一个公网 IP 或网段，例如 203.0.113.8")
    save.click()
    page.locator('.el-message-box').get_by_role("button", name="取消", exact=True).click()
    assert not page.evaluate("window.__rdpSafety.pending"), "Cancel must not start a transaction"
    page.evaluate("() => { window.__savedApply=window.go.ui.App.ApplyRdpWhitelist; window.go.ui.App.ApplyRdpWhitelist=()=>Promise.reject('Injected whitelist write failure'); }")
    save.click()
    page.locator('.el-message-box').get_by_role("button", name="确定", exact=True).click()
    page.get_by_text("操作未完成，请展开诊断详情查看原因。",exact=True).wait_for(state="visible")
    assert address.input_value() == "192.0.2.1/32", "Failed write must retain user input"
    page.get_by_role("button",name="诊断详情",exact=True).click()
    page.get_by_text("Injected whitelist write failure",exact=True).wait_for(state="visible")
    page.get_by_role("button",name="诊断详情",exact=True).click()
    page.evaluate("() => { window.go.ui.App.ApplyRdpWhitelist=window.__savedApply; window.__rdpSafety.session.known=false; }")
    check.click()
    expect(save).to_be_disabled()
    page.evaluate("window.__rdpSafety.session.known=true")
    check.click()
    expect(save).to_be_enabled()
    page.evaluate("() => { Object.assign(window.__rdpSafety,{pending:true,transaction:'expired-fixture',phase:'awaiting-confirmation',deadline:new Date(Date.now()-60000).toISOString(),canConfirm:true}); }")
    check.click()
    expect(keep).to_be_disabled()
    page.get_by_role("button",name="立即恢复原配置",exact=True).click()
    page.locator('.el-message-box').get_by_role("button",name="确定",exact=True).click()
    keep.wait_for(state="hidden")
    assert not page.evaluate("window.__rdpSafety.pending"), "Manual recovery must remain accessible after expiry"
    page.evaluate("() => { window.__savedEvents=window.go.ui.App.GetRdpLoginReport; window.go.ui.App.GetRdpLoginReport=()=>Promise.reject('Injected event read failure'); }")
    page.get_by_role("tab",name="登录记录",exact=True).click()
    page.get_by_role("button",name="读取记录",exact=True).click()
    page.get_by_text("读取失败，下面保留上次结果；原因见日志说明。",exact=True).wait_for(state="visible")
    assert page.get_by_test_id("login-event-table").get_by_text("203.0.113.7",exact=True).first.is_visible(), "Read failure must retain prior events"
    page.evaluate("() => { window.go.ui.App.GetRdpLoginReport=window.__savedEvents; }")
    page.wait_for_function("!document.querySelector('.el-message')",timeout=6000)
    page.set_viewport_size({"width": 1120, "height": 720})
    page.get_by_role("menuitem", name="账户管理").click()
    page.locator('[data-view="account"]').wait_for(state="visible")
    page.wait_for_timeout(300)
    page.screenshot(path=str(out / "account-1120.png"))
    page.get_by_role('menuitem', name='系统日志').click()
    panel = page.locator('[data-view="systemlogs"]')
    page.get_by_role('button', name='查看详情').first.wait_for(state='visible')
    search = page.get_by_placeholder('搜索来源、事件 ID、进程或详情')
    search.fill('1074')
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(1)
    search.fill('')
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(3)
    page.screenshot(path=str(out / 'system-logs-1120.png'))
    page.get_by_role('button', name='查看详情').first.click()
    page.get_by_text('事件详情与排查建议', exact=True).wait_for(state='visible')
    page.get_by_text('BugcheckCode', exact=True).wait_for(state='visible')
    page.wait_for_function("Array.from(document.querySelectorAll('.el-drawer')).some(el=>{const r=el.getBoundingClientRect();return r.width>400&&r.left>=0&&r.right<=innerWidth+1})")
    page.screenshot(path=str(out / 'system-event-detail-1120.png'))
    page.keyboard.press('Escape')
    page.get_by_text('事件详情与排查建议', exact=True).wait_for(state='hidden')
    page.set_viewport_size({'width':1320,'height':840})
    page.locator('.app-main').evaluate('el=>el.scrollTop=0')
    page.screenshot(path=str(out / 'system-logs-1320.png'))
    panel.locator('.log-toolbar .el-select').first.click()
    page.get_by_role('option',name='最近 30 天',exact=True).click()
    with page.expect_download() as info:
        page.get_by_role('button',name='导出诊断报告').click()
    download=info.value
    download.save_as(str(out/'system-report-fixture.json'))
    import json
    system_export=json.loads((out/'system-report-fixture.json').read_text(encoding='utf-8'))
    assert len(system_export['events']) == 3 and system_export['requestedDays'] == 7, 'Export scope must belong to the loaded report'
    page.evaluate("() => { window.__savedSystemReader=window.go.ui.App.GetSystemEventReport; window.go.ui.App.GetSystemEventReport=()=>Promise.reject('Injected system log read failure'); }")
    page.get_by_role('button',name='读取并分析').click()
    page.get_by_text('本次读取失败，下面保留的是上次成功结果。',exact=True).wait_for(state='visible')
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(3)
    page.evaluate("() => { window.go.ui.App.GetSystemEventReport=window.__savedSystemReader; }")
    # Clearing displayed results must not destroy Windows logs or allow stale reads to repopulate the view.
    page.evaluate("() => {window.__logClearCalls=[]; window.go.ui.App.ClearEventLog=()=>window.__logClearCalls.push('system');}")
    panel.get_by_role('button',name='清空结果',exact=True).click()
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(0)
    expect(panel.get_by_role('button',name='导出诊断报告')).to_be_disabled()
    expect(page.get_by_text('本次读取失败，下面保留的是上次成功结果。',exact=True)).to_have_count(0)
    expect(search).to_have_value('')
    panel.get_by_role('button',name='读取并分析').click()
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(3)
    page.evaluate("() => {window.__systemReader=window.go.ui.App.GetSystemEventReport;window.go.ui.App.GetSystemEventReport=()=>new Promise(resolve=>window.__finishSystem=()=>resolve({events:[],warnings:[],findings:[],summary:{total:0}}));}")
    panel.get_by_role('button',name='读取并分析').click()
    expect(panel.get_by_role('button',name='清空结果',exact=True)).to_be_disabled()
    page.evaluate("() => {window.__finishSystem();window.go.ui.App.GetSystemEventReport=window.__systemReader;}")
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(0)
    panel.get_by_role('button',name='读取并分析').click()
    expect(panel.locator('.log-table .el-table__body tbody tr')).to_have_count(3)
    page.get_by_role('menuitem',name='远程桌面',exact=True).click()
    page.get_by_role('tab',name='登录记录',exact=True).click()
    remote=page.get_by_test_id('remote-login-logs')
    table=page.get_by_test_id('login-event-table')
    remote.get_by_role('button',name='读取记录',exact=True).click()
    expect(table.locator('.el-table__body tbody tr')).to_have_count(2)
    remote.get_by_role('button',name='登录分析与来源汇总',exact=True).click()
    expect(remote.get_by_text('当前样本未达到重复失败提示阈值，不代表没有异常。',exact=True)).to_be_visible()
    remote.get_by_role('button',name='登录分析与来源汇总',exact=True).click()
    remote.get_by_role('textbox',name='搜索登录记录',exact=True).fill('0xc000006a')
    expect(table.locator('.el-table__body tbody tr')).to_have_count(1)
    expect(table.get_by_text('密码不正确',exact=True)).to_be_visible()
    table.get_by_role('button',name='查看详情').first.click()
    page.get_by_text('登录事件详情',exact=True).wait_for(state='visible')
    expect(page.locator('.el-drawer').get_by_text('SubStatus',exact=True)).to_be_visible()
    page.keyboard.press('Escape')
    page.get_by_text('登录事件详情',exact=True).wait_for(state='hidden')
    remote.locator('.remote-toolbar .el-select').first.click()
    page.get_by_role('option',name='最近 30 天',exact=True).click()
    with page.expect_download() as info:
        remote.get_by_role('button',name='导出诊断报告').click()
    info.value.save_as(str(out/'remote-login-report-fixture.json'))
    exported=json.loads((out/'remote-login-report-fixture.json').read_text(encoding='utf-8'))
    assert len(exported['events'])==2 and exported['summary']['failures']==1 and exported['requestedDays']==7, 'Export contains full report, not only filtered rows'
    remote.get_by_role('button',name='清空结果',exact=True).click()
    expect(page.get_by_test_id('login-event-table')).to_have_count(0)
    expect(remote.get_by_role('button',name='导出诊断报告')).to_be_disabled()
    remote.get_by_role('button',name='读取记录',exact=True).click()
    expect(table.locator('.el-table__body tbody tr')).to_have_count(2)
    expect(remote.get_by_role('textbox',name='搜索登录记录',exact=True)).to_have_value('')
    page.evaluate("() => {window.__remoteReader=window.go.ui.App.GetRdpLoginReport;window.go.ui.App.GetRdpLoginReport=()=>new Promise(resolve=>window.__finishRemote=()=>resolve({events:[],warnings:[],summary:{total:0}}));}")
    remote.get_by_role('button',name='读取记录',exact=True).click()
    expect(remote.get_by_role('button',name='清空结果',exact=True)).to_be_disabled()
    page.evaluate("() => {window.__finishRemote();window.go.ui.App.GetRdpLoginReport=window.__remoteReader;}")
    expect(table.locator('.el-table__body tbody tr')).to_have_count(0)
    remote.get_by_role('button',name='读取记录',exact=True).click()
    expect(table.locator('.el-table__body tbody tr')).to_have_count(2)
    assert page.evaluate('window.__logClearCalls.length')==0, 'Clear results must not call a Windows log-clearing backend'
    page.set_viewport_size({'width':800,'height':500})
    table.get_by_role('button',name='查看详情').first.click()
    page.get_by_text('登录事件详情',exact=True).wait_for(state='visible')
    page.wait_for_function("[...document.querySelectorAll('.el-drawer')].some(el=>{const r=el.getBoundingClientRect();return r.width>400&&r.left>=0&&r.right<=innerWidth+1})")
    page.screenshot(path=str(out/'remote-login-detail-800.png'))
    page.keyboard.press('Escape')
    page.get_by_text('登录事件详情',exact=True).wait_for(state='hidden')
    if os.environ.get('WINTOOLBOX_LAYOUT_AUDIT') in ('1', 'stress'):
        import json
        issues = []
        normal_cases = 0
        stress_cases = 0
        menus = [('overview', 1), ('account', 2), ('rdp', 4), ('firewall', 1), ('harden', 1), ('defender', 1), ('time', 1), ('power', 1), ('systemlogs', 1), ('update', 1), ('selfupdate', 1)]
        inspector = r"""async () => {
          const issues=[];
          const main=document.querySelector('.app-main'), log=document.querySelector('.op-log');
          const visible=el=>el.getClientRects().length && getComputedStyle(el).visibility!=='hidden';
          const rect=el=>el.getBoundingClientRect();
          let m=rect(main),l=rect(log);
          if(main.scrollWidth>main.clientWidth+2)issues.push('main horizontal overflow '+main.scrollWidth+'/'+main.clientWidth);
          if(l.bottom>innerHeight+1||m.bottom>l.top+1)issues.push('footer overlaps/clips main '+JSON.stringify({mainBottom:m.bottom,logTop:l.top,logBottom:l.bottom,height:innerHeight}));
          if(m.height<64)issues.push('main too short '+m.height);
          const buttons=[...main.querySelectorAll('button,.el-input__wrapper,.el-select__wrapper,.el-textarea,.el-checkbox')].filter(visible);
          for(const el of buttons){
            el.scrollIntoView({block:'center',inline:'nearest',behavior:'instant'});
            await new Promise(requestAnimationFrame);
            const r=rect(el); m=rect(main);
            const label=el.textContent.trim().slice(0,70);
            if(r.left<m.left-1||r.right>m.right+1||r.top<m.top-1||r.bottom>m.bottom+1)issues.push('button clipped '+label+' '+JSON.stringify({x:r.x,y:r.y,w:r.width,h:r.height,main:m.toJSON()}));
            else {const hit=document.elementFromPoint((r.left+r.right)/2,(r.top+r.bottom)/2); if(hit&&!el.contains(hit))issues.push('button covered '+label+' by '+hit.className);}
          }
          for(const table of [...main.querySelectorAll('.el-table')].filter(visible)){
            if(rect(table).width>main.clientWidth+1)issues.push('table exceeds main');
            const body=table.querySelector('.el-scrollbar__wrap');
            if(body&&body.scrollWidth>body.clientWidth+2&&getComputedStyle(body).overflowX==='hidden')issues.push('table cannot scroll horizontally');
          }
          for(const region of [...document.querySelectorAll('.app-header,.operation-status,.rdp-safety-banner,.op-log__head')].filter(visible)){
            for(const el of [...region.querySelectorAll('button')].filter(visible)){
              el.scrollIntoView({block:'nearest',inline:'nearest',behavior:'instant'});
              await new Promise(requestAnimationFrame);
              const r=rect(el),p=rect(region);
              if(r.top<Math.max(0,p.top)-1||r.bottom>Math.min(innerHeight,p.bottom)+1||r.left<0||r.right>innerWidth+1)issues.push('shell button clipped '+el.textContent.trim());
              const hit=document.elementFromPoint((r.left+r.right)/2,(r.top+r.bottom)/2);
              if(hit&&!el.contains(hit))issues.push('shell button covered '+el.textContent.trim());
            }
          }
          if(scrollY!==0||document.documentElement.scrollHeight>innerHeight+1)issues.push('window itself scrolled/clipped '+scrollY+'/'+document.documentElement.scrollHeight+'/'+innerHeight);
          return issues;
        }"""
        for language in ([] if os.environ.get('WINTOOLBOX_LAYOUT_AUDIT') == 'stress' else ['简体中文', 'English', '繁體中文']):
            print('Layout audit language:',language,flush=True)
            page.locator('.app-aside .el-select').click()
            page.get_by_role('option',name=language,exact=True).click()
            for width,height in [(800,500),(960,600),(1120,720),(1320,840)]:
                page.set_viewport_size({'width':width,'height':height})
                for expanded in [False, True]:
                    body=page.locator('.op-log__body')
                    if bool(body.count()) != expanded:
                        page.locator('.op-log__head button').first.click()
                    for index,(key,tabs) in enumerate(menus):
                        page.locator('.side-menu .el-menu-item').nth(index).click()
                        page.locator(f'[data-view="{key}"]').wait_for(state='visible')
                        for tab_index in range(tabs):
                            if tabs>1:
                                tab=page.locator(f'[data-view="{key}"] .el-tabs__item').nth(tab_index)
                                # Element Plus provides arrow controls for overflowing tab bars.
                                for _ in range(5):
                                    box=tab.bounding_box(); wrap=page.locator(f'[data-view="{key}"] .el-tabs__nav-scroll').bounding_box()
                                    if box['x']>=wrap['x'] and box['x']+box['width']<=wrap['x']+wrap['width']+1:break
                                    arrow=page.locator(f'[data-view="{key}"] .el-tabs__nav-'+('prev' if box['x']<wrap['x'] else 'next'))
                                    arrow.click();page.wait_for_timeout(80)
                                tab.click()
                            page.wait_for_function("[...document.querySelectorAll('.app-main .el-loading-mask')].every(el=>!el.getClientRects().length)")
                            page.mouse.move(0,0)
                            normal_cases += 1
                            found=page.evaluate(inspector)
                            if found:
                                case={'language':language,'size':[width,height],'expanded':expanded,'view':key,'tab':tab_index,'issues':found}
                                issues.append(case)
                                if len(issues)<15:page.screenshot(path=str(out/f'layout-issue-{len(issues)}.png'))
        # Worst-case shell: a persistent failure, pending recovery and expanded log.
        page.locator('.app-aside .el-select').click()
        page.get_by_role('option',name='简体中文',exact=True).click()
        page.set_viewport_size({'width':960,'height':600})
        page.locator('.side-menu .el-menu-item').nth(2).click()
        page.get_by_role('tab',name='客户端记录',exact=True).click()
        page.evaluate("() => {window.go.ui.App.ClearRdpHistory=()=>Promise.reject('Layout stress failure');}")
        page.get_by_role('button',name='清理全部记录',exact=True).click()
        page.locator('.el-message-box').get_by_role('button',name='确定',exact=True).click()
        page.locator('[data-testid="operation-status"].failed').wait_for(state='visible')
        page.locator('.el-message-box').wait_for(state='hidden')
        page.evaluate("() => {Object.assign(window.__rdpSafety,{pending:true,transaction:'layout-stress',phase:'awaiting-confirmation',deadline:new Date(Date.now()+3600000).toISOString(),canConfirm:false});}")
        page.get_by_role('tab',name='访问限制',exact=True).click()
        page.get_by_role('button',name='检查连接状态',exact=True).click()
        page.get_by_test_id('rdp-safety-banner').wait_for(state='visible')
        if not page.locator('.op-log__body').count():page.locator('.op-log__head button').first.click()
        for language in ['简体中文','English','繁體中文']:
            page.locator('.app-aside .el-select').click()
            page.get_by_role('option',name=language,exact=True).click()
            for width,height in [(800,500),(960,600)]:
                page.set_viewport_size({'width':width,'height':height})
                page.mouse.move(0,0)
                stress_cases += 1
                found=page.evaluate(inspector)
                page.locator('.app-main').evaluate('el=>el.scrollTop=0')
                page.wait_for_timeout(200)
                page.screenshot(path=str(out/f'layout-verified-{language}-{width}.png'))
                if found:
                    issues.append({'language':language,'size':[width,height],'stress':True,'issues':found})
                    page.screenshot(path=str(out/f'layout-stress-{language}-{width}.png'))
        (out/('layout-stress-audit.json' if os.environ.get('WINTOOLBOX_LAYOUT_AUDIT') == 'stress' else 'layout-audit.json')).write_text(json.dumps({'normalCases':normal_cases,'stackedCases':stress_cases,'issues':issues},ensure_ascii=False,indent=2),encoding='utf-8')
        print('Layout audit:',normal_cases,'normal cases,',stress_cases,'stacked-banner cases,',len(issues),'cases with issues',flush=True)
        assert not issues, json.dumps(issues[:8],ensure_ascii=False)
    assert not errors, errors
    browser.close()
print("UI smoke passed: 11 pages, three sizes, RDP tabs, collapsed diagnostics, collapsed log toolbar/expand/export, success auto-hide, persistent failure, compact result banner, log details, client cleanup and rename layout, cancel/failure/unknown session/expired confirmation/manual recovery, system event timeline/detail/export/read failure")
