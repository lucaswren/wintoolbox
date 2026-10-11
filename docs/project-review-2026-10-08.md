[English](project-review-2026-10-08.md) | [繁體中文](zh-TW/project-review-2026-10-08.md) | [简体中文](zh-CN/project-review-2026-10-08.md)

# WinToolBox project review

Review date: 2026-10-08. Scope: structure, naming, Go backend, Vue frontend, UI, system operation flow, build and quality practices. This is a historical review of the code at that date; the later [delivery record](optimization-2026-10-08.md) describes fixes. Source references below refer to the reviewed revision.

## Conclusion and scope

Retain Go + Wails + Vue. Business modules are clearly separated and broadly match the views. Async views, request sequence numbers, selected backups and post-write checks already exist. Prioritize operation reliability, truthful status, failure recovery and automated checks rather than a framework migration or full rewrite.

Findings are confirmed from source; potential consequences were not reproduced through fault injection. Visual inspection used repository account/firewall screenshots and source rather than a complete live Wails interaction test. No account, firewall, update, Defender, RDP, time or power settings were changed.

## Structure and naming

Startup elevates, prepares WebView2 and launches Wails. internal/ui owns the shell, bindings, DTOs and status aggregation. Domain packages cover account, rdp, firewall, harden, defender, update, selfupdate, sysinfo, wintime and power. internal/win provides commands, port validation, elevation, dialogs and runtime preparation. Views call composables → API → window.go.ui.App; there is no standalone HTTP backend.

Keep short domain names and PascalCase View.vue names. update/selfupdate could be distinguished more clearly as Windows/app updates. wintime avoids a standard-library time conflict. bind/status/dto have clear roles; util should be named after its responsibility. Replace useAction/useActions with actionRunner/useSystemActions. Prefer readable global form names, consistent SMB1/WinRM/NLA capitalization and stable camelCase JSON. Keep numbered kebab-case user guides. Any rename must update imports, bindings, scripts and docs; avoid wholesale cosmetic renaming.

## Priority findings

### P1: Incomplete backups can be marked saved

References: internal/update/update.go:173; internal/defender/defender.go:190. Multiple SetDWordValue errors were ignored, with Saved=1 used as the completion signal. Restoration also ignored some field read errors, potentially treating missing values as zero or absent preferences/policies. Check every read/write, distinguish missing fields from access failures, validate integrity/schema and modify settings only after a complete backup. Retain backups after failed restoration and identify the failed stage.

### P1: Probe failures can appear as business state

References: internal/firewall/riskports.go:40,233; internal/ui/status.go:253,364; internal/update/update.go:53; internal/wintime/wintime.go:24. PowerShell/netsh failures could end as an apparently inactive risk-port preset. Update service unknowns and failed time queries were not propagated. Return validity/errors and enabled/disabled/partial/unknown states. Keep successful collection timestamps; never interpret a failed query as off, on or absent.

### P1: Partial writes leave stale UI state

References: frontend/src/composables/useAction.js:24–43; internal/rdp/rdp.go:243. The runner refreshed only after success. RDP could write a new port and firewall rule, then fail during restart, leaving the UI showing the old port. Refresh affected modules after every attempted write. Consider structured operation results distinguishing failure, partial application, verification and restart requirements.

### P1: Self-update replacement lacks recovery

Reference: internal/selfupdate/selfupdate.go:243. A temporary PowerShell script waited for exit, overwrote the executable and launched it without an old-file backup, rollback, launch verification or persistent error log. Validation and later file use were separated in time; this was a potential risk, not a demonstrated exploit. Use restricted staging, recheck the copied file, retain the old executable, roll back failed replacement/startup and persist logs. Signing can follow as a release improvement.

### P2: Restoration and enabling have conflicting targets

Reference: internal/update/update.go:142. Restoring an originally disabled configuration could be incorrectly reported as failure because the final check required Disabled=false. Verify against the saved target. Distinguish restoring previous settings from enabling recommended settings, including Defender semantics.

### P2: Global timeout does not cancel work

References: internal/ui/status.go:25; internal/win/syscmd/syscmd.go:43. A 15-second aggregation timeout did not reach commands using fresh Background contexts. Repeated refreshes could overlap, and bundled firewall probes held completed results until the slowest finished. Pass contexts, budget/coalesce per module, limit PowerShell concurrency and return independent results.

### P2: Writes lack backend coordination

Reference: internal/ui/bind.go and Update/Defender backup-modify flows. Frontend busy is not a backend serialization guarantee. Concurrent calls/instances could interleave backup, write and restore. Lock by resource and keep backup/write together. Decide single-instance behavior from actual needs; do not block reads with one global lock.

### P2: Commands depend on language and encoding

References: internal/account/account.go:170,335; internal/update/update.go:490,541; internal/win/syscmd/syscmd.go:43. Looking up only Administrators/管理员 fails for renamed or other-language groups. Service/task/rule parsing depends on text, and ACP decoding lacks an explicit contract for UTF-8 scripts. Resolve groups by SID, adopt NetAPI/SCM/task APIs, use fixed UTF-8 JSON for scripts and separate stdout/stderr. Return exit codes, timeout categories and safe summaries.

### P2: Automated checks are missing

At review time Go packages had no tests, the frontend exposed only dev/build/preview, and no CI workflow was found. Cover incomplete backups, unknown propagation, partial RDP writes, restoration of disabled settings, version/checksum selection, history path bounds and status races. Use mocks and Windows VMs for system changes.

## Frontend/UI recommendations

Retain aligned navigation/views, shared Element Plus styling/CSS variables, async loading, action confirmations, request sequence protection and draft-aware refreshes. Improve:

1. Lockout colors: communicate health/risk rather than simply switch position.
2. Ping wording: no WinToolBox block does not prove external reachability.
3. Account density: separate user management and security policies; remove duplicated page titles.
4. Local refresh: avoid a whole-page loading mask for a single operation; show module timestamps.
5. Drafts: retain local policy fields/dirty state through navigation using composable state or KeepAlive.
6. Narrow windows: enable action wrapping and test 960×600/high DPI; the old ≤720px breakpoint rarely applied with a 960px minimum window.
7. Update progress: show percentage, speed and cancellation; expose automatic-download preference.
8. Power confirmation: explicitly state force-closing applications and remind users to save work.
9. Logs: bound capacity, include module/time/severity, export and auto-scroll; exclude secrets.
10. Errors: give one layer responsibility for user-facing refresh failure reporting.

Preserve the native startup order and the WebView2 download progress/Microsoft signature verification.

## Engineering, build and documentation

Generate bridge bindings/models and add type checks; avoid repeating manually maintained fields across DTO, state, App and views. Split large files by stable responsibility rather than line count. Add replaceable system-call interfaces for fault tests before introducing complex layers. Adopt Prettier/ESLint/gofmt and explicit Windows platform boundaries. Five files were unformatted at review time. Domain Chinese messages can later move toward localization/error codes without rewriting every error immediately.

Align README dependencies with go.mod (1.27 rather than the then-stated 1.22). Use npm ci to match the lockfile, synchronize package.json and package-lock.json versions via JSON parsing, and centralize artifacts. Ignored old binaries/hash/resource files are not proof of Git pollution. Match SHA256 entries to the asset and handle prerelease ordering before multi-architecture releases. Avoid holding update state locks during network I/O. Pin screenshot tool dependencies and document their commands; screenshots do not replace integration tests. Document operation/status contracts, system matrices, rollback and release checks.

## Recommended stages and recorded verification

1. Backup checks, unknown propagation, failed-operation refresh and updater rollback; verify with fault injection.
2. Restoration semantics, contexts, module refresh and write locks; verify target consistency and bounded repeated probes.
3. Generated contracts, responsibility-based modules, formatting and CI; verify contract drift detection and Windows builds.
4. UI density, progress, logs, scaling and release experience; accept minimum windows, high DPI, slow networks and unknown states.

At review time go test ./internal/... and go test ./... compiled successfully but had no test files; go vet ./internal/... returned no diagnostics. The frontend build initially encountered sandbox EPERM, then succeeded outside the sandbox after automatic approval, with two third-party PURE-comment warnings. The initial index JS was approximately 230.64 kB (79.88 kB gzip), and Vue 81.73 kB (32.47 kB gzip); bundle size was not the main bottleneck. gofmt listed five files. UI review was limited to screenshots/source; this report did not modify business code.
