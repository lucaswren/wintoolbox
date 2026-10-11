[English](operation-audit-2026-10-12.md) · [繁體中文](zh-Hant/operation-audit-2026-10-12.md) · [简体中文](zh-Hans/operation-audit-2026-10-12.md)

# Functional, operation and UI audit · 2026-10-12

## Why the high-risk preset felt unresponsive

The old block path launched five separate PowerShell processes, one per port, each performing write and verification work. Failed attempts could invoke deletion, creation and further probes through netsh. Removal ran a PowerShell batch followed unconditionally by five netsh deletions. Per-command timeouts did not bound the whole workflow. A read-only firewall query on this workstation took approximately 2 seconds; this is an observation, not a before/after write benchmark.

The shared frontend runner disabled buttons but offered no running phase or elapsed time. It announced success before the final module refresh. Logs were collapsed and transient toasts could disappear before the user inspected the result. The refresh wrapper also discarded its returned snapshot, preventing a caller from checking unknown fields.

## Corrections delivered

- High-risk writes now run in one PowerShell process. The successful path skips all netsh repair calls. A shared 60-second command-workflow budget includes writes, fallback and verification; the batch itself has a 30-second limit. Windows/provider scheduling may still vary. No artificial percentage or fixed duration promise is displayed.
- Block verification checks inbound direction, TCP, enabled state, blocking action, all-profile scope and the expected local port. Missing, malformed or unreadable probe output remains unknown. Removal requires confirmation that all five named rules are absent, including disabled rules.
- The preset refuses to block an enabled Remote Desktop port that matches 135/139/445/5985/5986; inability to read the RDP configuration also stops the operation. Excluding 3389 alone was insufficient for custom ports.
- A persistent panel above the scrollable content shows running, checking system state, completed, verification-needed or incomplete, plus elapsed seconds. It remains visible across navigation. A failed write is followed by a state check because partial changes may already exist. Results remain until dismissed or replaced by another operation.
- Completion notification follows the final check. Unknown module state and check failures produce a persistent warning rather than an unconditional success toast. Commands already accepted by Windows are not automatically rolled back. The panel describes stages, not per-port live progress.
- The firewall page displays individual results for all five ports and distinguishes local rule configuration from effective network filtering.
- RDP toggling no longer optimistically changes the UI before reading state; redundant lockout refresh was removed. Local account enable/disable now reads back the account flag.
- Windows Update disable cannot return confirmed success on an unknown status. Its UI describes detected configuration instead of claiming the service is running. NTP testing requires a time-offset sample and rejects embedded Windows error codes even when w32tm exits successfully.

## Module-by-module assessment

| Module | Reviewed behavior and limits |
|---|---|
| Overview | Fast/detail collection, cache, activation tags, missing fields, GiB and free/total units. Hardware collection may time out; missing information is not fabricated. |
| Local accounts | Empty/mismatched password validation, current-user protection, NetAPI account changes, administrator membership checks, Guest/autologon verification, password/lockout propagation. Passwords cannot be read back; successful password writes are Windows API acceptance, not a login test. Domain policy can override local policy. |
| Remote Desktop | Port validation, firewall synchronization/rollback, registry/NLA checks, service restart failures and client-history cleanup. Configured port and enabled policy are not proof of a reachable listener. Built-in firewall group synchronization is asynchronous. |
| Firewall | Profiles, allow rules, ping, five-port block preset, idempotent updates/removal and unknown states. Rule presence is local configuration; disabled profiles, domain policy, address scope, cloud controls and routing can change actual connectivity. Other vendors' rules are not removed. |
| Security hardening | SMB server setting, optional feature removal, WinRM service/rules and anonymous enumeration registry settings. Component removal may require restart; the feature is not a comprehensive security baseline. Native service changes remain sequential and may be slow. |
| Microsoft Defender | Preference/policy writes, validated backups, tamper-protection failures and preference readback. Enable forces protection on rather than restoring an earlier disabled preference. The tool does not bypass Tamper Protection. |
| Time synchronization | Time-zone IDs, peer configuration, service restart, resync and stripchart samples. A successful sample proves one response, not long-term reliability; a resync command does not independently measure clock accuracy. |
| Power | Delay limits, privilege acquisition, forced shutdown/restart confirmation and cancellation. A scheduling success is not completion of shutdown. Lock may hide the app immediately. Save application work first. |
| Windows Update | Snapshot validity, policy/service/task restoration, corrupt-backup rejection and detected aggregate state. Restore can legitimately leave updates disabled if that was the saved configuration. Windows remediation and domain policy may change settings later. |
| App update | Release selection, checksum matching, bounded downloads/cancel, staging ACL, revalidation, replacement and rollback logs. Existing progress UI is separate from system-action phases. SHA256 is not publisher authentication; startup survival is not full application readiness. The binary is unsigned. |

## UI and documentation acceptance

All ten views were checked in English, Simplified Chinese and Traditional Chinese, including 960×600 layout, 1280×800 default overview overflow, cached forms, language persistence and confirmation dialogs. Browser tests now deliberately hold writes and checks open, then exercise successful, failed and unknown results in every language. They use mocks and do not perform administrative writes.

The operation panel is outside the main scroll area so it remains visible at the bottom of a long page. Long explanations wrap; unusually long results scroll within a bounded panel. Confirmation still occurs before writes; busy state prevents duplicate operations. Logs remain session-only; this change does not persist logs across restarts.

Guides now describe operation phases, partial changes, per-port results and the meaning of configuration-only states. Older annotated screenshots are historical and do not depict the new panel.

Validation: 15 frontend regression tests, lint/type/format checks; Go tests and vet; generated bridge check; read-only PowerShell syntax parsing of mutation scripts; three-language browser tests; production Windows build. No live block rules, account changes, Defender/update changes or power operations were executed on the workstation.

## Remaining release acceptance

Use disposable Windows VMs to test real administrative writes with success, service-disabled, domain-managed, access-denied and timeout scenarios. Measure actual five-port batch latency there; do not infer a speedup ratio from process counts. Test port conflicts, all profiles disabled, duplicate/narrowed rules, partial writes, cleanup of disabled rules, and an inbound probe from another machine. Include native UAC/WebView2, 125%/150%/200% DPI, multiple monitors, SMB restart behavior, Windows Update task variants and signed release packaging.

The 60-second budget applies to the high-risk command workflow, not every system action. SMB optional-feature work, Windows Update task/service changes and policy propagation can still take longer. Native calls and providers are not all cancellable. A timeout is not proof that no change occurred; refresh and inspect partial state.

References: [Microsoft firewall rule fields](https://learn.microsoft.com/en-us/windows/win32/api/netfw/nn-netfw-inetfwrule), [firewall rule and port-filter association](https://learn.microsoft.com/en-us/powershell/module/netsecurity/get-netfirewallportfilter).
