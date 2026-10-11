[English](02-local-account.md) | [繁體中文](zh-TW/02-local-account.md) | [简体中文](zh-CN/02-local-account.md)

# 02 · Local accounts

Manage local user passwords, enabled state and administrator membership, or open **Security policies** to manage Guest, automatic logon, password complexity and account lockout. Domain accounts remain governed by domain policy.

## User management

1. Select a user under **User management**.
2. Enter the same new password twice and select **Change password**; confirm the operation.
3. Use **Enable / Disable**, **Grant administrator rights** or **Revoke administrator rights** as needed. The backend prevents disabling the current user or removing their administrator rights.
4. Verify that the new credentials work.

![Password and permissions](images/02-local-account-password.png)

## Security policies

| Action | Effect |
|---|---|
| Disable Guest account | Disables the local Guest account |
| Disable automatic logon | Disables AutoAdminLogon and clears Winlogon DefaultPassword |
| Apply password policy | Sets the minimum length and enables complexity |
| Enable lockout | Sets 10 attempts, a 30-minute lockout and a 30-minute reset window |
| Disable lockout | Sets the threshold to 0 and unlocks locked local accounts |
| Apply custom policy | Applies your threshold, lockout duration and reset window |

Current system values are used when available; otherwise lockout fields start at 10 / 30 / 30. When a policy is unknown, editable values represent the target rather than confirmed current settings. Group Policy can override local changes. Use strong passwords (the existing guidance recommends at least 12 characters, or 14 for servers, with mixed character types).

![Account policies](images/02-local-account.png)

Related: [Hardening guide](90-hardening-guide.md) · [Remote Desktop](03-remote-desktop.md)

Start by [downloading WinToolBox](00-download.md) and running it as administrator.

Screenshots show the earlier Simplified Chinese interface; the labels above describe the English interface. Download prompts are illustrations.
