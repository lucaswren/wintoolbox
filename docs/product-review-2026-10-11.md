[English](product-review-2026-10-11.md) · [繁體中文](zh-Hant/product-review-2026-10-11.md) · [简体中文](zh-Hans/product-review-2026-10-11.md)

# Product review · 2026-10-11

Update: Apache-2.0 has now been adopted; see [LICENSE](../LICENSE) and [NOTICE](../NOTICE). The license recommendations below record the earlier review. Language identifiers now use region-neutral en, zh-Hans and zh-Hant.

Reviewed the Wails shell, all ten Vue views, language catalogs, account/RDP/firewall/hardening actions, status collection, native setup, self-update staging, build scripts and repository metadata.

## Delivered changes

- Standardized the displayed product and executable name to WinToolBox. Existing firewall rule identifiers, update-cache directories and language preferences retain their names for compatibility.
- Default window: 1280×800; minimum: 960×600. Overview cards now share a row and navigation is compact. Default overview and navigation have no overflow in English, Traditional Chinese and Simplified Chinese. Small windows, unusually long hardware inventories and long lists retain scrolling so information remains accessible.
- All ten modules, dialogs, logs and native setup support three languages. First startup uses Windows display language; unsupported languages fall back to English. Explicit selection persists. There is no separate application installer: the executable starts directly and offers localized WebView2 setup if needed.
- English is the primary documentation, with matching Chinese editions. Existing annotated screenshots remain historical Simplified Chinese illustrations.
- Corrected memory and disk units to GiB; volume values are free/total. Microsoft Defender is named explicitly. NLA requires authentication before session establishment; password complexity follows Windows policy and applies to subsequent password changes. Security controls are individual settings, not a complete security assessment. Update messages report whether a newer release was found.

## Review findings and limits

Resource locks and post-operation refresh protect related operations from overlap and expose partial failures. Self-update requires SHA256, rechecks staged files, restricts staging-directory access and keeps replacement/rollback logs. These mechanisms reduce risk but are not publisher authentication: release checksums share the release trust boundary. The executable still lacks Authenticode signing; signing is the recommended next release improvement.

Remote Desktop and firewall state describe local configuration, not cloud security groups or end-to-end reachability. Domain policy, Defender Tamper Protection and restart requirements can override or delay local actions. Do not use this tool as a substitute for an organization security baseline.

Validation: frontend lint/type checks, 11 regression tests and formatting; Go tests and vet; generated bridge consistency; Windows production build; mock-browser tests across all ten views in three languages at 960×600, persistence/drafts/dialogs, plus no-overflow assertions at 1280×800. Browser verification uses mock data and performs no administrative writes. Native UAC, clean-machine WebView2 installation, multiple physical monitors/DPI settings and destructive operations require a disposable Windows VM acceptance pass.

## Repository description and licensing

Proposed GitHub description: “WinToolBox is a multilingual Windows administration toolkit for local accounts, Remote Desktop, firewall, Microsoft Defender, time synchronization, power and update management. Built with Go, Wails and Vue.”

The repository has no LICENSE and GitHub reports no detected license. Public source alone does not grant general reuse rights; see [GitHub's license guidance](https://choosealicense.com/no-permission/).

Recommendation: [Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0.html) for commercial adoption with explicit contributor patent grants, retained notices and modification notices. [MIT](https://choosealicense.com/licenses/mit/) is a shorter permissive alternative. [GPL-3.0](https://choosealicense.com/licenses/gpl-3.0/) fits a goal of requiring distributed derivative versions to provide corresponding source under the license. No project license was selected or added in this change.

Direct dependency license spot-check: Wails, Vue, Element Plus and its icons use MIT; winres uses ISC; x/sys uses BSD-3-Clause. This is not a complete transitive dependency/license audit. Before publishing a licensed binary, inventory all included dependencies and ship the required license/copyright notices; document compiler/tool versions and the authoritative English license text.
