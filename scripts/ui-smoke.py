"""Read-only frontend smoke test against the docs' simulated Wails backend.

Run with a Vite server on http://127.0.0.1:5173. Requires Playwright, as does
docs/capture_screens.py. No call reaches a real Windows administrative operation.
"""
import ast
from pathlib import Path

from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[1]
tree = ast.parse((ROOT / "docs/capture_screens.py").read_text(encoding="utf-8"))
mock = next(ast.literal_eval(node.value) for node in tree.body if isinstance(node, ast.Assign)
            and any(isinstance(t, ast.Name) and t.id == "MOCK_JS" for t in node.targets))
out = ROOT / "artifacts/ui"
out.mkdir(parents=True, exist_ok=True)

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 960, "height": 600})
    errors = []
    page.on("pageerror", lambda error: errors.append(str(error)))
    page.add_init_script(mock)
    page.add_init_script("localStorage.setItem('wintoolbox.language', 'zh-CN')")
    page.goto("http://127.0.0.1:5173", wait_until="networkidle")
    page.get_by_role("menuitem", name="本地账户").click()
    page.locator('[data-view="account"]').wait_for(state="visible")
    page.get_by_role("tab", name="安全策略").click()
    field = page.locator(".el-tab-pane:visible .el-input-number input").first
    field.fill("12")
    field.press("Tab")
    page.get_by_role("menuitem", name="防火墙").click()
    page.get_by_role("menuitem", name="本地账户").click()
    assert page.get_by_role("tab", name="安全策略").get_attribute("aria-selected") == "true"
    assert page.locator(".el-tab-pane:visible .el-input-number input").first.input_value() == "12"
    page.screenshot(path=str(out / "account-960.png"))
    page.get_by_role("tab", name="用户管理").click()
    for title, key in [("本机概览", "overview"), ("本地账户", "account"), ("远程桌面", "rdp"), ("防火墙", "firewall"), ("安全加固", "harden"), ("防病毒", "defender"), ("时间同步", "time"), ("电源", "power"), ("系统更新", "update"), ("软件更新", "selfupdate")]:
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
    page.set_viewport_size({"width": 1120, "height": 720})
    page.get_by_role("menuitem", name="本地账户").click()
    page.locator('[data-view="account"]').wait_for(state="visible")
    page.wait_for_timeout(300)
    page.screenshot(path=str(out / "account-1120.png"))
    assert not errors, errors
    browser.close()
print("UI smoke passed: 10 pages, 960x600/1120x720, drafts retained, no page errors")
