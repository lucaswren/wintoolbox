[English](05-security-harden.md) | [繁體中文](zh-Hant/05-security-harden.md) | [简体中文](zh-Hans/05-security-harden.md)

# 05 · Security hardening

Apply individual controls for common server exposure points:

| Action | Effect and compatibility |
|---|---|
| Disable SMBv1 | Disables SMBv1 on the SMB server; very old clients may lose share access and a restart may be required |
| Disable and block WinRM | Disables the WinRM service and blocks inbound TCP 5985/5986; remote PowerShell/Ansible may stop working |
| Block anonymous enumeration | Sets RestrictAnonymous=1 and RestrictAnonymousSAM=1 to reduce anonymous enumeration of local accounts and shares |

Review dependencies before confirming. Group Policy may override these settings. Refresh after the operation and inspect each control's verified or unknown state.

![Security hardening](images/05-security-harden.png)

Start by [downloading WinToolBox](00-download.md) and running it as administrator.

Screenshots show the earlier Simplified Chinese interface; the labels above describe the English interface. Download prompts are illustrations.
