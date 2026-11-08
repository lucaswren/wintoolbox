$ErrorActionPreference = 'Stop'
$result = @{bootTime=''; services=@(); recentEvents=@(); eventsPartial=$false}

try {
    $boot = (Get-CimInstance -ClassName Win32_OperatingSystem -Property LastBootUpTime).LastBootUpTime
    if ($boot) { $result.bootTime = $boot.ToUniversalTime().ToString('o') }
} catch {}

$names = @('TermService', 'W32Time', 'wuauserv', 'MpsSvc', 'WinDefend', 'Spooler')
$services = $null
try { $services = @(Get-Service) } catch {}
foreach ($name in $names) {
    $state = 'unknown'
    if ($null -ne $services) {
        $service = $services | Where-Object Name -eq $name | Select-Object -First 1
        $state = if (!$service) { 'missing' } else { ([string]$service.Status).ToLowerInvariant() }
    }
    $result.services += @{name=$name; state=$state}
}

$since = (Get-Date).AddDays(-7)
$providers = @{
    41 = 'Microsoft-Windows-Kernel-Power'
    6008 = 'EventLog'
    1001 = 'Microsoft-Windows-WER-SystemErrorReporting'
}
foreach ($id in $providers.Keys) {
    try {
        $event = Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName=$providers[$id]; Id=$id; StartTime=$since} -MaxEvents 1
        if ($event) { $result.recentEvents += @{id=$id; time=$event.TimeCreated.ToUniversalTime().ToString('o')} }
    } catch {
        if ($_.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound,*') { $result.eventsPartial = $true }
    }
}
$result.recentEvents = @($result.recentEvents | Sort-Object { $_.time } -Descending)
ConvertTo-Json -InputObject $result -Depth 4 -Compress
