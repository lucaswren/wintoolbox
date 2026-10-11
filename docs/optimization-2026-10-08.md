[English](optimization-2026-10-08.md) · [繁體中文](zh-Hant/optimization-2026-10-08.md) · [简体中文](zh-Hans/optimization-2026-10-08.md)

# Four-stage optimization delivery record

Historical delivery record, 2026-10-08. Validation results below describe that delivery, not the current localization work.

## Delivered changes

1. Reliability: check each Update/Defender backup write and validate required restore fields; reject corrupt backups. Preserve unknown states for firewall, ping, updates, time and Defender. Verify the original module after both successful and failed writes. Secure updater staging, verify hashes before/after copying, retain the previous executable, log replacement failures and roll back startup failures.
2. State/concurrency: module status API, coalesced slow probes, shared contexts cancelled after the last caller leaves, and single background jobs for uncancellable native queries. Limit system commands to six and serialize writes by account/network/time resources. Restore Windows Update to its previous configuration; Defender enable explicitly targets on. Resolve the administrator group by SID and manage membership through NetAPI; query service state via the registry and task state via COM.
3. Engineering: generate bridge types from Go methods/JSON DTOs and type-check the frontend API. Add ESLint, Prettier, fault regression tests and Windows CI. Extract updater, snapshot, shared probe, log and module state logic; rename useAction/useActions to actionRunner/useSystemActions. Use npm ci, synchronize lockfile versions, centralize artifacts/releases and align environment docs.
4. UI/release: separate account tabs, retain drafts with KeepAlive, wrap buttons and scroll the sidebar. Avoid full masks for local operations, fix lockout colors and ping wording, and show collection time. Keep 500 log entries with module/severity colors, expansion, auto-scroll and export. Add update amount/speed/cancel and automatic-download preference. Explain forced power actions. Fix the reported Vue/source-map-js dependency vulnerabilities.

## Recorded validation

- Eleven Go regression tests covered backup write failures/corruption, unknown states, coalescing/cancellation, download status/cancellation cleanup, checksum matching, prereleases and updater startup rollback; all passed.
- Six frontend regression tests, lint, bridge types, generated-contract consistency and formatting passed.
- Go vet, gofmt and git diff --check passed.
- Mock UI checks covered ten pages at 960×600 and 1120×720, retained drafts, no page errors and no main-area horizontal overflow; screenshots were written to artifacts/ui.
- npm audit reported zero known vulnerabilities for the final lockfile at that time.
- The complete Windows desktop build passed; executable and SHA256 were written to artifacts/releases.
- Two third-party VueUse PURE-comment warnings remained without affecting build success.

Real system writes, real self-update installation, cross-Windows-version and high-DPI acceptance were not performed. UI checks mocked Wails; updater fault tests only replaced temporary test files. Follow [Development and acceptance](development.md) before deployment. Vue pages remain JavaScript although the bridge has type checks. Resource locks coordinate one process, not multiple instances.

The initial optimization delivery retained 1.2.0 and generated only a local build. A later user request reset the version to 1.0.0, aligned update requests/docs to lucaswren/wintoolbox, backed up and recreated the public repository, and published v1.0.0. Existing 1.2.0 users must install manually.
