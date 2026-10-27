"""Overlay layout and button audit against compiled UI with a simulated Wails backend.
Run python scripts/ui-overlays.py; no real Windows writes or network requests.
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
out = ROOT / "artifacts/ui-overlays"
out.mkdir(parents=True, exist_ok=True)

import json
catalogs={key:json.loads((ROOT/'frontend/src/i18n'/f'{key}.json').read_text(encoding='utf-8')) for key in ['en','zh-Hans','zh-Hant']}
fixture = r"""() => {
 const app=window.go.ui.App; window.__overlayWrites=[];
 const getStatus=app.GetStatus;
 const users=Array.from({length:100},(_,i)=>({name:'overlay-user-'+String(i).padStart(3,'0'),enabled:true,admin:false,current:false}));
 app.GetStatus=async()=>({...await getStatus(),accounts:users});
 app.GetModuleStatus=app.GetStatus;
 app.GetTimeZones=async()=>Array.from({length:100},(_,i)=>({id:'Zone-'+i,label:'(UTC+08:00) '+i+' '+ 'Long timezone description '.repeat(6)}));
 const getReport=app.GetSystemEventReport;
 app.GetSystemEventReport=async()=>{const r=structuredClone(await getReport());r.events[0].message='Long system event message. '.repeat(300);r.events[0].xml='<Event>'+('Long XML evidence '.repeat(1500))+'</Event>';r.events[0].data=Array.from({length:40},(_,i)=>({name:'Field-'+i,value:'Long structured evidence '.repeat(50)}));return r;};
 const getUpdate=app.CheckAppUpdate;
 app.CheckAppUpdate=async()=>({...await getUpdate(),hasUpdate:true,downloaded:true,verified:true,latestVersion:'1.0.2'});
 app.GetAppUpdateInfo=app.CheckAppUpdate;app.DownloadAppUpdate=app.CheckAppUpdate;
 for(const name of ['SaveNTPServer','ClearRdpHistory','ApplyRdpWhitelist','ApplyAppUpdate'])app[name]=async(...args)=>{window.__overlayWrites.push({name,args});if(name==='ApplyAppUpdate')throw Error('Simulated restart boundary');return null;};
}"""
issues=[];cases=0
quick=os.environ.get('WINTOOLBOX_OVERLAYS_QUICK')=='1'
with sync_playwright() as p:
    browser=p.chromium.launch(headless=True)
    page=browser.new_page(viewport={'width':800,'height':500})
    page.add_init_script("localStorage.setItem('wintoolbox.language',window.name||'zh-Hans');\n"+mock+'\n('+fixture+')()')
    errors=[]
    page.on('pageerror',lambda error:errors.append(str(error)))
    dist=(ROOT/'frontend/dist').resolve()
    def serve(route):
        asset=(dist/(urlsplit(route.request.url).path.lstrip('/') or 'index.html')).resolve()
        if asset.is_relative_to(dist) and asset.is_file():route.fulfill(path=str(asset))
        else:route.fulfill(status=404,body='Not found')
    page.route('http://wintoolbox.test/**',serve)
    def check_panel(panel,kind,buttons=False):
        global cases
        cases+=1
        page.mouse.move(0,0)
        panel.evaluate("async el=>{await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));for(let i=0;i<180;i++){let moving=false;for(let node=el;node;node=node.parentElement)if([...node.classList].some(c=>/-((enter|leave)-active)$/.test(c)))moving=true;if(!moving)break;await new Promise(requestAnimationFrame)}await Promise.all(el.getAnimations().map(a=>a.finished.catch(()=>{})))}")
        b=panel.bounding_box();vp=page.viewport_size
        found=[]
        if not b or b['x']< -1 or b['y']< -1 or b['x']+b['width']>vp['width']+1 or b['y']+b['height']>vp['height']+1:found.append('panel exceeds viewport: '+str(b))
        if buttons:
            for button in panel.locator('button:visible').all():
                button.scroll_into_view_if_needed()
                ok=button.evaluate("el=>{const r=el.getBoundingClientRect(),hit=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);return r.top>=0&&r.bottom<=innerHeight+1&&r.left>=0&&r.right<=innerWidth+1&&!!hit&&el.contains(hit)}")
                if not ok:found.append('button clipped/covered: '+button.inner_text())
        if not found and lang=='zh-Hans' and vp['width']==800:
            import re
            page.screenshot(path=str(out/('verified-'+re.sub(r'[^a-z0-9]+','-',kind.lower())+'.png')))
        if found:
            issues.append({'locale':lang,'size':page.viewport_size,'kind':kind,'issues':found})
            page.screenshot(path=str(out/f'issue-{len(issues)}.png'))
        return not found
    def menu(index):
        page.locator('.side-menu .el-menu-item').nth(index).click()
    def dialog(open_dialog,kind):
        for action in ['cancel','close','escape','confirm']:
            before=page.evaluate('window.__overlayWrites.length')
            open_dialog()
            box=page.locator('.el-message-box:visible')
            box.wait_for(state='visible')
            good=check_panel(box,kind+'/'+action,True)
            if good and kind=='long NTP confirmation':
                content=box.locator('.el-message-box__content')
                assert content.evaluate('el=>el.scrollHeight>el.clientHeight'), 'Long confirmation content must have its own scroll area'
                content.evaluate('el=>el.scrollTop=el.scrollHeight')
                assert content.evaluate('el=>el.scrollTop>0'), 'Full confirmation text must remain reachable'
            if not good:
                page.keyboard.press('Escape')
            elif action=='close':box.locator('.el-message-box__headerbtn').click()
            elif action=='escape':page.keyboard.press('Escape')
            else:box.locator('.el-message-box__btns button').nth(0 if action=='cancel' else 1).click()
            box.wait_for(state='hidden')
            if good:
                expected=before+(action=='confirm')
                page.wait_for_function('(n)=>window.__overlayWrites.length===n',arg=expected)
    def dropdown(trigger,kind):
        trigger.click()
        pop=page.locator('.el-select__popper:visible,.el-select-v2__popper:visible').last
        pop.wait_for(state='visible')
        good=check_panel(pop,kind)
        if good:
            pop.evaluate("async root=>{for(const el of [root,...root.querySelectorAll('*')])if(['auto','scroll'].includes(getComputedStyle(el).overflowY)&&el.scrollHeight>el.clientHeight){el.scrollTop=el.scrollHeight}await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)))}")
            option=pop.locator('[role="option"]:visible').last
            if option.count():
                if kind in ['account dropdown','virtual timezone dropdown']:assert '99' in option.inner_text(), 'The final option must be reachable by scrolling'
                option.scroll_into_view_if_needed()
                option.click()
            else:page.keyboard.press('Escape')
        else:page.keyboard.press('Escape')
        pop.wait_for(state='hidden')
    for lang in (['zh-Hans'] if quick else ['zh-Hans','en','zh-Hant']):
        page.evaluate('(lang)=>window.name=lang',lang)
        page.goto('http://wintoolbox.test/',wait_until='networkidle')
        expect(page.locator('html')).to_have_attribute('lang',lang)
        tr=lambda key:catalogs[lang].get(key,key)
        print('Overlay locale:',lang,flush=True)
        for width,height in ([(800,500)] if quick else [(800,500),(960,600),(1320,840)]):
            page.set_viewport_size({'width':width,'height':height})
            # Changing the language selector to its current option preserves the locale.
            page.locator('.app-aside .el-select').click()
            language_pop=page.locator('.el-select__popper:visible').last
            check_panel(language_pop,'language dropdown')
            page.keyboard.press('Escape');language_pop.wait_for(state='hidden')
            menu(1)
            dropdown(page.locator('[data-view="account"] .el-select'),'account dropdown')
            menu(6)
            dropdown(page.locator('[data-view="time"] .el-select'),'virtual timezone dropdown')
            field=page.get_by_placeholder('time.windows.com')
            field.fill(('time.example.net '*100).strip())
            dialog(lambda:page.get_by_role('button',name=tr('保存 NTP'),exact=True).click(),'long NTP confirmation')
            menu(2)
            page.get_by_role('tab',name=tr('客户端记录'),exact=True).click()
            page.wait_for_function("[...document.querySelectorAll('.app-main .el-loading-mask')].every(el=>!el.getClientRects().length)")
            dropdown(page.locator('[data-view="rdp"] .el-tab-pane:visible .el-select'),'client history dropdown')
            dialog(lambda:page.get_by_role('button',name=tr('清理全部记录'),exact=True).click(),'client cleanup confirmation')
            page.get_by_role('tab',name=tr('访问限制'),exact=True).click()
            page.locator('[data-view="rdp"] textarea').fill('192.0.2.1')
            dialog(lambda:page.get_by_role('button',name=tr('保存访问限制'),exact=True).click(),'RDP whitelist confirmation')
            menu(10)
            dialog(lambda:page.get_by_role('button',name=tr('安装并重启'),exact=True).click(),'software update confirmation')
            menu(8)
            toggle=page.locator('button[aria-controls="system-log-options"]')
            if toggle.get_attribute('aria-expanded')!='true':toggle.click()
            for index in range(3):dropdown(page.locator('[data-view="systemlogs"] .el-select').nth(index),'system log dropdown '+str(index))
            # Reset the category filter before opening an event with long evidence.
            page.locator('[data-view="systemlogs"] .el-select').nth(2).click()
            pop=page.locator('.el-select__popper:visible').last
            pop.locator('[role="option"]').first.click();pop.wait_for(state='hidden')
            for close in ['button','escape','mask']:
                page.locator('.log-table .el-table__body button').first.click()
                drawer=page.locator('.el-drawer:visible')
                drawer.wait_for(state='visible')
                check_panel(drawer,'long system log drawer/'+close,True)
                drawer.get_by_role('button',name='XML',exact=True).click()
                drawer.locator('.el-collapse-item__wrap:visible').last.evaluate("async el=>{await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));await Promise.all(el.getAnimations().map(a=>a.finished.catch(()=>{})))}")
                body=drawer.locator('.el-drawer__body')
                body.evaluate('el=>el.scrollTop=el.scrollHeight')
                assert body.evaluate('el=>el.scrollTop>0'), 'Full long XML must remain reachable by scrolling'
                if body.evaluate('el=>el.scrollWidth>el.clientWidth+2'):issues.append({'locale':lang,'size':[width,height],'kind':'drawer horizontal overflow'})
                if close=='button':drawer.locator('.el-drawer__close-btn').click()
                elif close=='escape':page.keyboard.press('Escape')
                else:page.mouse.click(5,height/2)
                drawer.wait_for(state='hidden')
    (out/('audit-quick.json' if quick else 'audit.json')).write_text(json.dumps({'cases':cases,'issues':issues},ensure_ascii=False,indent=2),encoding='utf-8')
    print('Overlay audit:',cases,'cases,',len(issues),'issues',flush=True)
    assert not errors,errors
    browser.close()
assert not issues,json.dumps(issues[:6],ensure_ascii=False)
