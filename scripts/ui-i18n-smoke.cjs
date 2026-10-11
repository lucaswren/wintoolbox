// Mock-only UI checks. No call reaches a Windows administrative operation.
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const root = path.resolve(__dirname, '..')
const mock = fs.readFileSync(path.join(root, 'docs/capture_screens.py'), 'utf8').match(/MOCK_JS = r"""([\s\S]*?)"""/)[1]
const cases = [
  { language: 'en', menus: ['Overview','Local accounts','Remote Desktop','Firewall','Security hardening','Microsoft Defender','Time synchronization','Power','Windows Update','App update'], policy: 'Security policies', guest: 'Disable Guest account' },
  { language: 'zh-Hant', menus: ['本機概覽','本地帳戶','遠端桌面','防火牆','安全加固','Microsoft Defender','時間同步','電源','系統更新','軟體更新'], policy: '安全策略', guest: '停用來賓帳戶' },
  { language: 'zh-Hans', menus: ['本机概览','本地账户','远程桌面','防火墙','安全加固','Microsoft Defender','时间同步','电源','系统更新','软件更新'], policy: '安全策略', guest: '禁用来宾账户' },
]
const keys = ['overview','account','rdp','firewall','harden','defender','time','power','update','selfupdate']
async function main() {
  const browser = await chromium.launch({ headless: true })
  const out = path.join(root, 'artifacts/ui/i18n')
  fs.mkdirSync(out, {recursive:true})
  try {
    for (const tc of cases) {
      const context = await browser.newContext({ viewport: {width:960,height:600}, locale:'en-US' })
      await context.addInitScript(mock + `\nwindow.go.ui.App.GetAppInfo = async () => ({ name: 'WinToolBox', version: 'v1.0.0', language: ${JSON.stringify(tc.language)} });`)
      const page = await context.newPage()
      await page.setViewportSize({width:1280,height:800});
      const errors = []
      page.on('pageerror', error => errors.push(String(error)))
      await page.goto('http://127.0.0.1:5173', {waitUntil:'networkidle'})
      assert.equal(await page.locator('html').getAttribute('lang'), tc.language)
      for(const selector of ['.app-main','.side-menu']) {
        const size=await page.locator(selector).evaluate(el=>({height:el.clientHeight,scroll:el.scrollHeight,width:el.clientWidth,scrollWidth:el.scrollWidth}));
        assert.ok(size.scroll<=size.height+2 && size.scrollWidth<=size.width+2, tc.language+selector+JSON.stringify(size));
      }
      await page.screenshot({path:path.join(out,tc.language+'-overview-1280.png')});
      await page.setViewportSize({width:960,height:600});
      for (let i = 0; i < keys.length; i++) {
        const menu = page.getByRole('menuitem', {name:tc.menus[i],exact:true})
        await menu.scrollIntoViewIfNeeded()
        await menu.click()
        await page.locator(`[data-view="${keys[i]}"]`).waitFor()
        assert.ok(await page.locator('.app-main').evaluate(el => el.scrollWidth <= el.clientWidth + 2), tc.language + ':' + keys[i])
      }
      await page.getByRole('menuitem', {name:tc.menus[1],exact:true}).click()
      await page.getByRole('tab', {name:tc.policy,exact:true}).click()
      const field = page.locator('.el-tab-pane:visible .el-input-number input').first()
      await field.fill('15')
      await field.press('Tab')
      await page.screenshot({path:path.join(out,tc.language+'-account-960.png')})
      // Switch with the public selector; cached views and library dialogs must follow.
      if (tc.language === 'en') {
        await page.locator('.app-aside .el-select__wrapper').click()
        await page.getByRole('option', {name:'简体中文',exact:true}).click()
      }
      await page.locator('.app-aside .el-select__wrapper').click()
      await page.getByRole('option', {name:'English',exact:true}).click()
      assert.equal(await page.locator('html').getAttribute('lang'), 'en')
      assert.equal(await field.inputValue(), '15')
      await page.getByRole('button', {name:'Disable Guest account',exact:true}).click()
      await page.getByRole('button', {name:'Cancel',exact:true}).click()
      await page.reload({waitUntil:'networkidle'})
      assert.equal(await page.locator('html').getAttribute('lang'), 'en')
      assert.equal(await page.evaluate(() => localStorage.getItem('wintoolbox.language')), 'en')
      await page.setViewportSize({width:1120,height:720})
      await page.getByRole('menuitem', {name:'Firewall',exact:true}).click()
      await page.locator('[data-view="firewall"]').waitFor()
      await page.screenshot({path:path.join(out,tc.language+'-firewall-1120.png')})
      assert.deepEqual(errors, [])
      await context.close()
      console.log(`Passed ${tc.language}: 10 views, startup detection, switching, drafts, confirmation, persistence, layout`)
    }
  } finally { await browser.close() }
}
main().catch(error => { console.error(error); process.exitCode = 1 })

