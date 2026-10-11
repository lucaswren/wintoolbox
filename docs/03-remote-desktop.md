[English](03-remote-desktop.md) | [繁體中文](zh-TW/03-remote-desktop.md) | [简体中文](zh-CN/03-remote-desktop.md)

# 03 · Remote Desktop

Manage the Remote Desktop service, listening port, Network Level Authentication (NLA) and local mstsc connection history.

1. Enter a port from 1–65535 and select **Save port**. WinToolBox updates the registry and synchronizes its `WinToolbox-RDP` firewall allow rule.
2. Enable Remote Desktop if required and select **Enforce NLA**. Very old clients may be unable to connect with NLA enforced.
3. Connect using `IP:new-port`. Update cloud security groups and upstream firewall rules separately; changing the local port does not update them.
4. Refresh and verify the actual state, especially after a partial failure. If the service or NLA state is unknown, refresh before proceeding.

Changing ports alone does not replace access controls or NLA. Confirm that the old port is no longer exposed where appropriate.

**Connection history** supports filtering/search, deleting one record, clearing a selected type and clearing all records. Cleanup covers host/MRU history, saved username hints, TERMSRV credentials, Default.rdp/recent .rdp files and bitmap cache. It does not change Remote Desktop service or port settings.

![Remote Desktop](images/03-remote-desktop.png)

Start by [downloading WinToolBox](00-download.md) and running it as administrator.

Screenshots show the earlier Simplified Chinese interface; the labels above describe the English interface. Download prompts are illustrations.
