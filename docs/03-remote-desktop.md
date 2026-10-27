[English](03-remote-desktop.md) · [繁體中文](zh-Hant/03-remote-desktop.md) · [简体中文](zh-Hans/03-remote-desktop.md)

# 03 · Remote Desktop

Manage the Remote Desktop service, listening port, Network Level Authentication (NLA) and server sign-in records.

1. Enter a port from 1024–65535 and select **Save port**. WinToolBox saves the original settings, arms an independent SYSTEM recovery task and adds scoped TCP/UDP rules. The service restarts after about ten seconds, disconnecting sessions.
2. Enable Remote Desktop if required and select **Enforce NLA**. Very old clients may be unable to connect with NLA enforced.
3. Connect using `IP:new-port`. Update cloud security groups and upstream firewall rules separately; changing the local port does not update them.
4. Disconnect and reconnect using the new settings within five minutes, then confirm in the global recovery banner. Existing sessions and console clicks cannot confirm. Without confirmation, settings are restored even if the tool exits.

The RDP page also provides IPv4/IPv6 source allowlists, listener checks tied to the TermService process, candidate rules from ActiveStore, and server logon/disconnection events. Allowlist changes restrict existing applicable allow rules; conflicting managed or broad rules prevent changes. NLA changes and enabling all firewall profiles use the same protection. Destructive generic network operations require the local console; pending transactions and confirmed allowlists restrict ordinary network writes.

Keep VPS console access available. Recovery covers saved host settings only, not cloud security groups, NAT, credentials or OS failures. Domain controllers and unknown/noninteractive sessions are refused; managed RDP policies must be changed by their administrator. Target systems are Windows Server 2016–2025 with Desktop Experience, PowerShell 5.1, NetSecurity and Task Scheduler. Multi-version VPS acceptance remains required. See the [detailed protection, diagnostics and console recovery guide (Simplified Chinese)](zh-Hans/03-remote-desktop.md#vps-防失联保护).

Changing ports alone does not replace access controls or NLA. Confirm that the old port is no longer exposed where appropriate.

**Access restrictions** shows the allowlist and local listener status. Raw rules and output are collapsed under **Diagnostics**. **Sign-in records** supports a 1–90 day range, source summaries, failure reasons, repeated-failure hints, search, event details/XML and full JSON export. Network authentication failures are distinguished from confirmed remote interactive logons; incomplete channel coverage is reported. **Clear results** resets this view without deleting Windows event logs. **Client history** retains search, type filters, single-record deletion, clearing a selected type and clearing all records, including saved credentials and bitmap cache. Records load when this tab is opened; cleanup does not change the Remote Desktop service or port.

![Remote Desktop](images/03-remote-desktop.png)

Start by [downloading WinToolBox](00-download.md) and running it as administrator.

Screenshots show the earlier Simplified Chinese interface; the labels above describe the English interface. Download prompts are illustrations.

[Operation feedback and audit](operation-audit-2026-10-12.md)
