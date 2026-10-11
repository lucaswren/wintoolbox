## {{VERSION}}

### Changes

- English, Simplified Chinese and Traditional Chinese UI and documentation; automatic Windows display-language detection and persistent manual selection.
- Region-neutral language identifiers: en, zh-Hans and zh-Hant.
- Compact navigation and a two-column overview in the 1280×800 default window.
- Local accounts, Remote Desktop, firewall, Microsoft Defender, time, power and update management.
- Apache-2.0 licensing. Retain LICENSE and NOTICE when redistributing; third-party components retain their own licenses.

Users running v1.2.0 must install v1.0.0 manually; automatic updates do not downgrade versions.

### Download and verification

The executable is not Authenticode-signed. Confirm the release source and verify the checksum before running. Follow your organization's application policy.

- File: `{{EXE}}`
- SHA256: {{SHA256}}

```powershell
Get-FileHash .\{{EXE}} -Algorithm SHA256
```

Release files: {{EXE}}, {{SHA256_FILE}}, LICENSE and NOTICE.

[English documentation](https://github.com/lucaswren/wintoolbox/blob/main/docs/README.md) · [繁體中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hant/README.md) · [简体中文](https://github.com/lucaswren/wintoolbox/blob/main/docs/zh-Hans/README.md)
