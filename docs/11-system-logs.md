# 11 · System log analysis

Open **System log analysis** to read selected Windows System events from the last 1–90 days. Choose a limit of 100, 300 or 1000 records, filter/search the timeline, and open a record for its message, XML, structured fields and nearby evidence. **Export diagnostic report** exports the entire loaded report as JSON, including host/account details where present; review it before sharing.

The page covers startup, shutdown requests and sequences, unclean shutdowns, bugchecks, hardware/storage reports, dump failures, service failures, updates, log clearing and clock changes. Classification matches both provider and ID. Counts refer to events, not the number of reboots; related records within ten minutes are clues rather than established causes. Reads are bounded, and disabled logs, permission failures and truncation are reported. Audit and system settings are not changed.

Event 41 alone does not establish power loss or another specific cause. Check stop codes and dumps, then correlate monitoring and VPS provider/host records. See Microsoft's [unexpected reboot investigation](https://learn.microsoft.com/en-us/troubleshoot/windows-server/performance/troubleshoot-unexpected-reboots-system-event-logs) and [Event 41 limitations](https://learn.microsoft.com/en-us/troubleshoot/windows-client/performance/event-id-41-restart).

Read the [detailed guide in Simplified Chinese](zh-Hans/11-system-logs.md) for event IDs, limitations and investigation steps.
