"""Focused log-page layout/interaction audit. Uses mocked data; no OS writes."""
import ast
import json
import os
from pathlib import Path
from urllib.parse import urlsplit
from playwright.sync_api import sync_playwright, expect

ROOT = Path(__file__).resolve().parents[1]
tree = ast.parse((ROOT / 'docs/capture_screens.py').read_text(encoding='utf-8'))
mock = next(ast.literal_eval(n.value) for n in tree.body if isinstance(n, ast.Assign)
            and any(isinstance(t, ast.Name) and t.id == 'MOCK_JS' for t in n.targets))
out = ROOT / 'artifacts/ui-log-pages'
out.mkdir(parents=True, exist_ok=True)
catalogs = {lang: json.loads((ROOT / f'frontend/src/i18n/{lang}.json').read_text(encoding='utf-8')) for lang in ['zh-Hans','en','zh-Hant']}
fixture = r"""() => {
  const app=window.go.ui.App;
  for(const name of ['GetRdpLoginReport','GetSystemEventReport']){
    const original=app[name];
    app[name]=async(...args)=>{
      const r=structuredClone(await original(...args));
      r.events=Array.from({length:40},(_,i)=>({...r.events[i%r.events.length],recordId:String(i+1),user:'DOMAIN\\'+('LongAccount'.repeat(5)),address:'2001:db8:1234:5678:abcd:abcd:abcd:abcd',message:'Long original system message '.repeat(300),detail:'Long remote event message '.repeat(300),xml:'<Event>'+('Long XML evidence '.repeat(1500))+'</Event>',data:[{name:'Field-'+i,value:'Long structured evidence '.repeat(100)}]}));
      r.summary.total=r.events.length;r.warnings=['Long channel diagnostic '.repeat(60)];
      r.truncated=true;if(name==='GetRdpLoginReport')r.partial=true;
      return r;
    };
  }
}"""
issues=[]
cases=0
quick=os.environ.get('WINTOOLBOX_LOG_PAGES_QUICK')=='1'
with sync_playwright() as p:
    browser=p.chromium.launch(headless=True)
    page=browser.new_page(viewport={'width':800,'height':500})
    page.add_init_script("localStorage.setItem('wintoolbox.language',window.name||'zh-Hans');\n"+mock+'\n('+fixture+')()')
    errors=[]
    page.on('pageerror',lambda e:errors.append(str(e)))
    dist=(ROOT/'frontend/dist').resolve()
    def serve(route):
        path=(dist/(urlsplit(route.request.url).path.lstrip('/') or 'index.html')).resolve()
        if path.is_relative_to(dist) and path.is_file():route.fulfill(path=str(path))
        else:route.fulfill(status=404,body='Not found')
    page.route('http://wintoolbox.test/**',serve)
    def audit(panel,kind):
        global cases
        cases+=1
        page.mouse.move(0,0)
        page.wait_for_function("[...document.querySelectorAll('.app-main .el-loading-mask')].every(el=>!el.getClientRects().length)")
        found=panel.evaluate(r"""async root=>{
          const issues=[];const main=document.querySelector('.app-main');
          if(main.scrollWidth>main.clientWidth+2)issues.push('main horizontal overflow');
          const controls=[...root.querySelectorAll('button,.el-select__wrapper,.el-input__wrapper')].filter(el=>el.getClientRects().length&&!el.closest('.el-table'));
          for(const table of root.querySelectorAll('.el-table')){
            const buttons=[...table.querySelectorAll('tbody button')].filter(el=>el.getClientRects().length);
            for(const index of new Set([0,Math.floor(buttons.length/2),buttons.length-1]))if(buttons[index])controls.push(buttons[index]);
          }
          for(const el of controls){
            el.scrollIntoView({block:'center',inline:'nearest',behavior:'instant'});await new Promise(requestAnimationFrame);
            const r=el.getBoundingClientRect(),m=main.getBoundingClientRect();
            if(r.left<m.left-1||r.right>m.right+1||r.top<m.top-1||r.bottom>m.bottom+1)issues.push('control clipped '+el.textContent.slice(0,40));
            else{const hit=document.elementFromPoint((r.left+r.right)/2,(r.top+r.bottom)/2);if(hit&&!el.contains(hit))issues.push('control covered '+el.textContent.slice(0,40));}
          }
          return issues;
        }""")
        if found:issues.append({'kind':kind,'language':lang,'size':[width,height],'issues':found})
    for lang in catalogs:
        page.evaluate('(lang)=>window.name=lang',lang)
        page.goto('http://wintoolbox.test/',wait_until='networkidle')
        tr=lambda s:catalogs[lang].get(s,s)
        print('Log-page locale:',lang,flush=True)
        for width,height in ([(800,500)] if quick else [(800,500),(960,600),(1120,720),(1320,840)]):
            page.set_viewport_size({'width':width,'height':height})
            for key,index in [('system',8),('remote',2)]:
                page.locator('.side-menu .el-menu-item').nth(index).click()
                if key=='remote':page.get_by_role('tab',name=tr('登录记录'),exact=True).click()
                panel=page.locator('[data-view="systemlogs"]') if key=='system' else page.get_by_test_id('remote-login-logs')
                read=panel.get_by_role('button',name=tr('读取并分析' if key=='system' else '读取记录'),exact=True)
                read.click()
                table=panel.locator('.log-table') if key=='system' else page.get_by_test_id('login-event-table')
                expect(table.locator('.el-table__body tbody tr')).to_have_count(40)
                toggle=panel.get_by_role('button',name=tr('筛选与范围'),exact=True)
                if toggle.get_attribute('aria-expanded')=='true':toggle.click()
                expect(panel.locator('.log-options')).not_to_be_visible()
                expect(panel.get_by_test_id('log-summary')).to_be_visible()
                # Explanations and analysis do not consume the default results view.
                notes=panel.get_by_role('button',name=tr('日志说明'),exact=True)
                if notes.get_attribute('aria-expanded')=='true':notes.click()
                if key=='remote':
                    analysis=panel.get_by_role('button',name=tr('登录分析与来源汇总'),exact=True)
                    if analysis.get_attribute('aria-expanded')=='true':analysis.click()
                for expanded in [False,True]:
                    if bool(page.locator('.op-log__body').count())!=expanded:page.locator('.op-log__head button').first.click()
                    audit(panel,key+'/default/log='+str(expanded))
                    toggle.click();expect(panel.locator('.log-options')).to_be_visible()
                    audit(panel,key+'/filters/log='+str(expanded))
                    toggle.click()
                    notes.click();audit(panel,key+'/notes/log='+str(expanded));notes.click()
                # Table scrolling reaches final rows and the fixed detail action.
                table.locator('.el-scrollbar__wrap').first.evaluate('el=>el.scrollTop=el.scrollHeight')
                table.get_by_role('button',name=tr('查看详情'),exact=True).last.click()
                drawer=page.locator('.el-drawer:visible')
                drawer.wait_for(state='visible')
                expect(drawer.locator('pre:visible')).to_have_count(0)
                for label in ['事件原文','结构化字段','XML']:
                    drawer.get_by_role('button',name=tr(label),exact=True).click()
                body=drawer.locator('.el-drawer__body')
                body.evaluate('el=>el.scrollTop=el.scrollHeight')
                page.wait_for_timeout(300)
                r=drawer.bounding_box()
                if r['x'] < -1 or r['y'] < -1 or r['x']+r['width'] > width+1 or r['y']+r['height']>height+1:issues.append({'kind':'drawer bounds','language':lang,'size':[width,height]})
                if body.evaluate('el=>el.scrollWidth>el.clientWidth+2'):issues.append({'kind':'drawer horizontal overflow','language':lang,'size':[width,height]})
                cases+=1
                drawer.locator('.el-drawer__close-btn').click();drawer.wait_for(state='hidden')
                panel.get_by_role('button',name=tr('清空结果'),exact=True).click()
                read.click();expect(table.locator('.el-table__body tbody tr')).to_have_count(40)
                page.wait_for_timeout(350)
                if key=='system':expect(table.locator('.el-tag').first).to_have_text(tr('异常关机记录'))
                if page.locator('.op-log__body').count():page.locator('.op-log__head button').first.click()
                page.locator('.app-main').evaluate('el=>el.scrollTop=0')
                page.mouse.move(0,0);page.wait_for_timeout(250)
                if width in [800,1320]:page.screenshot(path=str(out/f'{key}-{lang}-{width}.png'))
    browser.close()
(out/('audit-quick.json' if quick else 'audit.json')).write_text(json.dumps({'cases':cases,'issues':issues,'pageErrors':errors},ensure_ascii=False,indent=2),encoding='utf-8')
print('Log page audit:',cases,'cases;',len(issues),'issues;',len(errors),'page errors')
assert not issues and not errors, str(issues[:8])+str(errors)
