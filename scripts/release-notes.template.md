## {{VERSION}}

### Changes

- Added English, Simplified Chinese and Traditional Chinese UI and documentation, automatic Windows display-language detection and persistent language selection.
- Improved navigation and the default 1280×800 layout; added persistent operation stages, elapsed time and verification results.
- Batched high-risk firewall port operations with a shared timeout and per-port verification; improved account, Remote Desktop, Windows Update and NTP operation feedback.
- Fixed UTF-16 password-policy parsing, Defender runtime-state detection, automatic-logon state checks, concurrent module refreshes, update-asset checksum matching and Windows Update policy error handling.
- Updated Go and frontend dependencies, regression tests and documentation.
- Adopted Apache-2.0 licensing. Retain LICENSE and NOTICE when redistributing; third-party components retain their own licenses.

Users running v1.2.0 must download {{VERSION}} manually; automatic updates do not downgrade versions.

### Download and verification

The executable is not Authenticode-signed. Confirm the release source and verify the checksum before running. Follow your organization's application policy.

- File: `{{EXE}}`
- SHA256: {{SHA256}}

```powershell
Get-FileHash .\{{EXE}} -Algorithm SHA256
```

Release files: {{EXE}}, {{SHA256_FILE}}, LICENSE and NOTICE.

[English documentation](https://github.com/lucaswren/wintoolbox/blob/main/docs/README.md) · [繁體中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hant/README.md) · [简体中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hans/README.md)
