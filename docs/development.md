[English](development.md) · [繁體中文](zh-Hant/development.md) · [简体中文](zh-Hans/development.md)

# Development and acceptance

Environment: Windows amd64, Go 1.27+, Node.js 22.13+, Windows PowerShell 5.1 or PowerShell 7.

## Checks

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

Regenerate `frontend/src/api/wails.d.ts` whenever exported Go methods or DTOs change. CI verifies the generated contract with `-check`. Bridge type checks do not mean every Vue component uses TypeScript. Use Prettier/ESLint for frontend code and gofmt for Go. Builds use npm ci and write release artifacts to `artifacts/releases`.

## Operation contracts

- Failed probes remain unknown; do not substitute zero-value false.
- Mark backups saved only after every required field is written; validate all necessary fields before restoration.
- Refresh the affected module after both success and failure because a failed action may have partially changed the system.
- Serialize writes by resource; RDP, firewall and WinRM share a network lock. Locks apply within one process.
- Coalesce overlapping module probes. Cancel shared cancellable queries after the last caller leaves; uncancellable native queries finish without starting duplicates. System commands run with a concurrency limit of six.
- Windows Update restoration targets its previous settings; enabling Defender explicitly targets on.
- Restart/shutdown confirmations must state that applications are forcibly closed and ask the user to save work.
- Log operation summaries/errors, never passwords. Keep at most 500 entries and support export.

## Localization

English is the default/fallback. Catalogs in `frontend/src/i18n` contain matching English, Simplified Chinese and Traditional Chinese keys; messages use positional placeholders such as `{0}`. Call `t()` at the presentation layer. Keep backend status tokens raw when using comparisons. Reactive getters/computed values let cached views update when the locale changes.

On native startup, GetAppInfo reports the Windows user display language. The UI uses it unless `wintoolbox.language` contains a supported manual preference. Browser development falls back to navigator.languages. Normalize OS language tags to the region-neutral identifiers en, zh-Hans and zh-Hant. Existing stored regional identifiers migrate without losing the manual preference. Unsupported languages use English. Native setup uses `internal/locale` before WebView2 exists. OS command output, device/user names and release notes remain in their source language.

Keep English user guides under docs/, with corresponding files under docs/zh-Hans/ and docs/zh-Hant/. Add language navigation and preserve shared image links. Existing screenshots depict the earlier Simplified Chinese interface.

## Self-update

Separate check/download locks from state locks so status reads and cancellation remain responsive. Match checksums to the selected asset. Compare prereleases using their core version and prerelease identifier. Preserve automatic-download preferences locally.

Stage updates in a random directory accessible to the user, administrators and SYSTEM. Wait up to two minutes for exit, verify hashes before and after copying, and retain a `.previous` file using File.Replace. Roll back if startup fails or exits within five seconds. Keep staging files, the previous executable and logs for diagnosis. The five-second check is not a UI-ready protocol.

## Windows VM acceptance

1. Record initial settings/recovery points and test supported Windows versions and domain policy environments.
2. Inject backup write failures, insufficient permissions and probe timeouts; verify no false success/disabled state.
3. Simulate an RDP restart failure and confirm the UI reports the actual port and error.
4. Disable/restore Windows Update when initially disabled; verify the original configuration is retained.
5. Inject updater hash, copy and startup failures; verify recovery and logs.
6. Check 960×600/1280×800 windows and 125%/150%/200% scaling, all three languages, label wrapping and sidebar access.
7. Verify draft preservation, unknown states, download cancellation, log export and simultaneous update/system operation feedback.
8. On fresh English, Simplified Chinese and Traditional Chinese Windows profiles, verify native setup language and first UI language. Check manual preference persistence and unsupported-language fallback.

## Mock UI checks

```powershell
python -m pip install -r docs/requirements.txt
python -m playwright install chromium
# Run npm run dev in another terminal.
python scripts/ui-smoke.py
```

For multilingual mock UI checks, install the Node Playwright package in your tooling environment, start Vite on port 5173 and run `node scripts/ui-i18n-smoke.cjs`. Set `PLAYWRIGHT_MODULE` to an absolute package path if Playwright is installed outside this repository. The check covers all ten views in each language, startup detection, switching, confirmations, draft retention and preference persistence.

The scripts use a mocked Wails backend and write screenshots to artifacts/ui. Automated tests use mocks and temporary directories; they do not replace real Windows VM acceptance or change real account/firewall/update settings.
