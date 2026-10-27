[English](README.md) · [繁體中文](README.zh-Hant.md) · [简体中文](README.zh-Hans.md)

# WinToolBox

WinToolBox is a Windows administration toolkit for inspecting system information and managing local accounts, Remote Desktop, firewall rules, Microsoft Defender, time synchronization, power, and updates. Built with Go, Wails, Vue and Element Plus.
Version: v1.1.1

## Languages and documentation

English is the default/fallback and primary documentation language. English, Traditional Chinese and Simplified Chinese are supported. First launch follows the user's Windows display language and uses English when no supported language matches. A sidebar selector overrides detection and remembers the choice.

WinToolBox runs directly as an EXE. If WebView2 is missing, its automatic installation window follows the Windows display language before the app starts. There is no separate application installation wizard. OS command output, device/user names and release notes retain their source language.

- [English documentation](docs/README.md)
- [繁體中文文件](docs/zh-Hant/README.md)
- [简体中文文档](docs/zh-Hans/README.md)
- [Download and first launch](docs/00-download.md)
- [Language selection](docs/localization.md)
- [Windows hardening guide](docs/90-hardening-guide.md)
- [Development and acceptance](docs/development.md)

## Features

| Module | Capabilities |
|---|---|
| Overview | Hostname, Windows/build, activation, resolution, IPs, volumes and hardware |
| Local accounts | Passwords, enabled state, administrator rights, Guest, automatic logon, lockout and password policies |
| Remote Desktop | Service, port with firewall synchronization, NLA and mstsc history cleanup |
| Firewall | Profiles, ping blocking, high-risk ports and inbound TCP allow rules |
| Security hardening | SMBv1, WinRM and anonymous enumeration |
| Microsoft Defender | Disable/enable Defender real-time protection with fallback and verification |
| Time synchronization | Time zones, NTP presets, synchronization and NTP testing |
| Power | Lock, delayed restart/shutdown and schedule cancellation |
| Windows Update | Disable and restore configuration saved before disabling |
| App update | GitHub releases, SHA256 verification, download progress/cancellation and installation |

## Supported systems

Windows 10 / 11 and Windows Server 2016 / 2019 / 2022 / 2025, amd64. Administrator privileges are required. Missing WebView2 is downloaded automatically; internet access is required for this step.

## Build and run

Dependencies: Go 1.27+, Node.js 22.13+ and WebView2 Runtime.

```powershell
.\scripts\build.ps1
```

Development with the Wails CLI:

```powershell
wails dev
```

Run `WinToolBox.exe` and accept UAC elevation, or select **Run as administrator**. Production artifacts are written to `artifacts/releases/`.

## Download and verification

Download from [GitHub Releases](https://github.com/lucaswren/wintoolbox/releases/latest). The executable currently has no Authenticode signature, so the browser or SmartScreen may block it. Confirm its source and compare the release SHA256 before selecting **Keep**, or **More info → Run anyway**.

```powershell
Get-FileHash .\WinToolBox-v1.1.1-windows-amd64.exe -Algorithm SHA256
```

## Architecture

```mermaid
flowchart LR
  Views[Vue views] --> API[API and composables]
  API --> Wails[Wails bridge]
  Wails --> UI[internal/ui DTOs and status]
  UI --> Domain[account / rdp / firewall / harden / defender / wintime / power / update / selfupdate / sysinfo]
  Domain --> Windows[internal/win and Windows APIs]
```

`frontend/src/views/` matches the ten sidebar modules. `frontend/src/i18n/` holds translation catalogs. `internal/locale/` detects Windows language and translates native setup prompts. English guides live in `docs/`, with corresponding Chinese guides in `docs/zh-Hans/` and `docs/zh-Hant/` and shared images.

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
```

## v1.1.1 release notes

- Fixed Windows 11 Teredo/System firewall-rule misclassification during RDP port changes and hardening-status timeouts.
- Prevented WinRM hardening from blocking custom RDP ports 5985/5986.
- Enhanced remote sign-in analysis with source summaries, failure reasons, repeated-failure hints, filtering, event/XML details and JSON export.
- Added **Clear results** to both log pages; Windows event logs are retained. Client history, credential and cache cleanup remain available.
- Fixed timestamp ordering, partial-read handling and exported collection scope. Go/frontend checks, 11-page regression and 366 layout cases passed.
- Windows Server/VPS live acceptance remains unperformed; retain an independent console recovery channel and finish pending RDP changes before updating.

[Download release](https://github.com/lucaswren/wintoolbox/releases/tag/v1.1.1) · [Validation report (Simplified Chinese)](docs/zh-Hans/validation-vps-2026-10-23.md)

Users running the older v1.2.0 build must still install manually; automatic updates do not downgrade versions.

[Product review and license recommendations](docs/product-review-2026-10-11.md)

## License

WinToolBox source code and documentation are licensed under [Apache-2.0](LICENSE). See [NOTICE](NOTICE) for project attribution. Third-party components retain their own licenses.

[Functional and UI audit](docs/operation-audit-2026-10-12.md)
