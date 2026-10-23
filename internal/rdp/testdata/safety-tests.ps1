param([string]$Script)
$ErrorActionPreference = 'Stop'
$root = Split-Path $Script
# Prevent test runs from contending with the application's global mutex.
$source = [IO.File]::ReadAllText($Script).Replace('Global\WinToolBox-RdpSafety-v1', ('Local\WinToolBox-RdpSafety-Test-' + $PID))
[IO.File]::WriteAllText($Script, $source)
$global:fake = $null
function Reset-Fixture {
  Remove-Item -LiteralPath (Join-Path $root 'pending.xml') -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath (Join-Path $root 'whitelist.xml') -Force -ErrorAction SilentlyContinue
  $global:fake = @{Port=3389; NLA=0; Listening=3389; Reconnected=$false; TaskFail=$false; WriteFail=$false; NewFail=$false; Restarts=0; Rules=@{}; Tasks=@{}; Writes=0; Firewall=$true; ProviderFail=$false; Events=@(); LogEnabled=$true; EventSession=22; EventRecord=101; EventAge=0; RestartFail=$false; DomainRole=3; ActivateFail=$false; LocalRules='True'; Policy=$null; PersistFail=$false}
  $global:fake.Rules['Existing-RDP'] = [pscustomobject]@{Name='Existing-RDP'; Direction='Inbound'; Enabled='True'; Action='Allow'; PolicyStoreSourceType='Local'; Profile='Any'; RemoteAddress=@('Any'); LocalPort='3389'; Protocol='TCP'; Service='TermService'}
}
function Assert($condition, $message) { if (!$condition) { throw ('ASSERT: ' + $message) } }
function Export-Clixml {
  param([Parameter(ValueFromPipeline=$true)]$InputObject,$LiteralPath,$Depth)
  process {
    if ($global:fake.PersistFail) { throw 'Injected state write failure' }
    $InputObject | Microsoft.PowerShell.Utility\Export-Clixml -LiteralPath $LiteralPath -Depth 12
  }
}
function Get-ItemProperty { param($LiteralPath,$Name,$Path,$ErrorAction)
  if ($Path -like '*Policies*' -or $LiteralPath -like '*Policies*' -or $LiteralPath -like '*SOFTWARE*') { return $global:fake.Policy }
  return [pscustomobject]@{PortNumber=$global:fake.Port; UserAuthentication=$global:fake.NLA; fDenyTSConnections=0}
}
function Set-ItemProperty { param($LiteralPath,$Name,$Type,$Value)
  $global:fake.Writes++
  if ($Name -eq 'PortNumber') { $global:fake.Port=[int]$Value }
  if ($Name -eq 'UserAuthentication') { $global:fake.NLA=[int]$Value }
}
function Get-CimInstance { param($ClassName,$Filter)
  if ($ClassName -eq 'Win32_ComputerSystem') { return [pscustomobject]@{DomainRole=$global:fake.DomainRole} }
  return [pscustomobject]@{State='Running';ProcessId=123}
}
function Get-Service { param($Name) return [pscustomobject]@{Status='Running'} }
function Get-NetTCPConnection { param($State,$LocalPort,$ErrorAction)
  if (!$LocalPort -or [int]$LocalPort -eq $global:fake.Listening) { return [pscustomobject]@{OwningProcess=123;LocalPort=$global:fake.Listening} }
}
function Get-NetUDPEndpoint { param($LocalPort,$ErrorAction) }
function Restart-Service { param($Name,[switch]$Force)
  if ($global:fake.RestartFail) { throw 'Injected restart failure' }
  $global:fake.Restarts++; $global:fake.Listening=$global:fake.Port
}
function Get-NetFirewallProfile { param($PolicyStore)
  if ($global:fake.ProviderFail) { throw 'Mock provider failed' }
  foreach ($name in @('Domain','Private','Public')) { [pscustomobject]@{Name=$name;Enabled=$global:fake.Firewall;DefaultInboundAction='Block';AllowLocalFirewallRules=$global:fake.LocalRules} }
}
function Set-NetFirewallProfile { param($Profile,$Enabled) $global:fake.Firewall=[bool]$Enabled; $global:fake.Writes++ }
function Get-NetFirewallPortFilter {
  param([Parameter(ValueFromPipeline=$true)]$InputObject,$PolicyStore)
  process {
    if ($InputObject) { [pscustomobject]@{Name=$InputObject.Name; Protocol=$InputObject.Protocol;LocalPort=$InputObject.LocalPort} }
    else { foreach ($r in @($global:fake.Rules.Values)) { [pscustomobject]@{Name=$r.Name;Protocol=$r.Protocol;LocalPort=$r.LocalPort} } }
  }
}
function Get-NetFirewallRule {
  param([Parameter(ValueFromPipeline=$true)]$InputObject,$PolicyStore,$Name)
  process {
    if ($InputObject) { $global:fake.Rules[$InputObject.Name] }
    elseif ($global:fake.Rules.ContainsKey($Name)) { $global:fake.Rules[$Name] }
    elseif ($ErrorActionPreference -ne 'SilentlyContinue') { throw 'Rule missing' }
  }
}
function Get-NetFirewallAddressFilter { param([Parameter(ValueFromPipeline=$true)]$InputObject) process { [pscustomobject]@{RemoteAddress=@($InputObject.RemoteAddress)} } }
function Get-NetFirewallServiceFilter { param([Parameter(ValueFromPipeline=$true)]$InputObject) process { [pscustomobject]@{Service=$InputObject.Service} } }
function Get-NetFirewallApplicationFilter { param([Parameter(ValueFromPipeline=$true)]$InputObject) process { [pscustomobject]@{Program='Any'} } }
function Set-NetFirewallRule {
  param([Parameter(ValueFromPipeline=$true)]$InputObject,$PolicyStore,$Name,$RemoteAddress)
  process {
    if (!$Name) { $Name=$InputObject.Name }
    if ($global:fake.WriteFail) { $global:fake.WriteFail=$false; throw 'Injected firewall write failure' }
    $global:fake.Rules[$Name].RemoteAddress=@($RemoteAddress | ForEach-Object { $_ -replace '/32$','' -replace '/128$','' }); $global:fake.Writes++
  }
}
function New-NetFirewallRule { param($PolicyStore,$Name,$DisplayName,$Direction,$Action,$Enabled,$Profile,$Protocol,$LocalPort,$RemoteAddress,$Service)
  if ($global:fake.NewFail) { throw 'Injected new-rule failure' }
  $global:fake.Rules[$Name]=[pscustomobject]@{Name=$Name;Direction=$Direction;Enabled='True';Action=$Action;PolicyStoreSourceType='Local';Profile=$Profile;Protocol=$Protocol;LocalPort=[string]$LocalPort;RemoteAddress=@($RemoteAddress | ForEach-Object { $_ -replace '/32$','' -replace '/128$','' });Service=$Service}
  $global:fake.Writes++
}
function Remove-NetFirewallRule { param([Parameter(ValueFromPipeline=$true)]$InputObject) process { $global:fake.Rules.Remove($InputObject.Name); $global:fake.Writes++ } }
function New-ScheduledTaskAction { param($Execute,$Argument) [pscustomobject]@{Execute=$Execute;Arguments=$Argument} }
function New-ScheduledTaskTrigger { param([switch]$Once,$At,$RepetitionInterval,[switch]$AtStartup) [pscustomobject]@{At=$At;Startup=$AtStartup} }
function New-ScheduledTaskSettingsSet { param([switch]$StartWhenAvailable,[switch]$AllowStartIfOnBatteries,[switch]$DontStopIfGoingOnBatteries,$ExecutionTimeLimit,$MultipleInstances) return @{} }
function Register-ScheduledTask { param($TaskName,$Action,$Trigger,$Settings,$User,$RunLevel,[switch]$Force)
  if ($global:fake.TaskFail) { throw 'Injected task registration failure' }
  if ($global:fake.ActivateFail -and $TaskName -eq 'WinToolBox-RdpSafety-Activate') { throw 'Injected activation registration failure' }
  $global:fake.Tasks[$TaskName]=[pscustomobject]@{State='Ready';Triggers=@($Trigger);Actions=@($Action);Principal=[pscustomobject]@{UserId=$User}}
}
function Get-ScheduledTask { param($TaskName,$ErrorAction) return $global:fake.Tasks[$TaskName] }
function Unregister-ScheduledTask { param($TaskName,$Confirm,$ErrorAction) $global:fake.Tasks.Remove($TaskName) }
function Get-WinEvent { param($FilterHashtable,$MaxEvents,$ErrorAction,$ErrorVariable,$ListLog,$LogName)
  if ($ListLog) { return [pscustomobject]@{IsEnabled=$global:fake.LogEnabled} }
  if ($LogName) { return [pscustomobject]@{RecordId=100} }
  if ($global:fake.ContainsKey('EventDenied') -and $global:fake.EventDenied) { throw 'Injected reconnect log failure' }
  if ($global:fake.Reconnected) {
    $e=[pscustomobject]@{TimeCreated=(Get-Date).AddSeconds($global:fake.EventAge);Id=25;RecordId=$global:fake.EventRecord}
    $e | Add-Member ScriptMethod ToXml { return ('<Event><UserData><EventXML><SessionID>' + $global:fake.EventSession + '</SessionID></EventXML></UserData></Event>') }
    return $e
  }
  return $global:fake.Events
}
function Apply($kind,$value) { & $Script -Action Apply -Kind $kind -Value $value -SessionID 22 -RemoteSession true }
function State { Import-Clixml -LiteralPath (Join-Path $root 'pending.xml') }
function Expect-Failure([scriptblock]$body, [string]$pattern) {
  $failed=$false
  try { & $body } catch { $failed=$true; Assert ($_.Exception.Message -like $pattern) ('Unexpected failure: ' + $_.Exception.Message) }
  Assert $failed ('Expected rejection: ' + $pattern)
}

Reset-Fixture
$global:fake.TaskFail=$true
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*could not be armed*'
Assert ($global:fake.Writes -eq 0) 'No writes before recovery task is armed'

Reset-Fixture
$global:fake.Rules['Existing-RDP'].Service='Any'; $global:fake.Rules['Existing-RDP'].LocalPort='Any'
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*Broad/ambiguous*'
Assert ($global:fake.Writes -eq 0) 'Broad rules fail closed'

Reset-Fixture
$global:fake.Rules['Existing-RDP'].PolicyStoreSourceType='GroupPolicy'
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*managed allow rule*'
Assert ($global:fake.Writes -eq 0) 'Managed rules fail closed'

Reset-Fixture
Apply 'whitelist' '192.0.2.1/32'
$s=State
Assert ($s.Phase -eq 'awaiting-confirmation') 'Pending survives command exit'
Assert ($global:fake.Tasks.ContainsKey('WinToolBox-RdpSafety-Recover')) 'Independent recovery armed'
Expect-Failure { & $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession true } '*Reconnect with RDP*'
Expect-Failure { Apply 'nla' '1' } '*pending*'
Expect-Failure { & $Script -Action Rollback -Transaction 'stale' } '*Stale transaction*'
$global:fake.ProviderFail=$true
$status=(& $Script -Action Status | ConvertFrom-Json)
Assert ($status.pending -and !$status.canConfirm) 'Provider failure retains recovery banner'
$global:fake.ProviderFail=$false
& $Script -Action Rollback -Transaction $s.ID
Assert ((State).Phase -eq 'rolled-back') 'Explicit rollback completes'
Assert ($global:fake.Rules.Count -eq 1 -and $global:fake.Rules['Existing-RDP'].RemoteAddress[0] -eq 'Any') 'Rollback restores only touched rules'

Reset-Fixture
$global:fake.WriteFail=$true
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*previous configuration restored*'
Assert ($global:fake.Rules.Count -eq 1) 'Partial write failure removes transaction-owned rules'

Reset-Fixture
Apply 'whitelist' '192.0.2.1/32'
$s=State; $s.Deadline=[datetime]::UtcNow.AddMinutes(-1); $s | Export-Clixml -LiteralPath (Join-Path $root 'pending.xml')
& $Script -Action Recover -Transaction $s.ID
Assert ((State).Phase -eq 'rolled-back') 'Deadline recovery works without application process'
Expect-Failure { & $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession true } '*No pending change*'

Reset-Fixture
Apply 'whitelist' '192.0.2.1/32'
$s=State; $global:fake.Reconnected=$true
& $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession true
Assert ((State).Phase -eq 'confirmed') 'New RDP session can confirm'
Assert (Test-Path -LiteralPath (Join-Path $root 'whitelist.xml')) 'Confirmed whitelist persists'
Expect-Failure { & $Script -Action Guard } '*whitelist is active*'

Reset-Fixture
$global:fake.Rules['Existing-RDP'].RemoteAddress=@('198.51.100.2/32')
Apply 'port' '3399'
$s=State
Assert ($global:fake.Restarts -eq 0 -and $global:fake.Listening -eq 3389) 'Restart is deferred to independent task'
$newRule=@($global:fake.Rules.Values | Where-Object Name -like '*-TCP' | Select-Object -First 1)[0]
Assert ($newRule.RemoteAddress[0] -eq '198.51.100.2') 'Port change preserves source restrictions'
& $Script -Action Activate -Transaction $s.ID
Assert ($global:fake.Listening -eq 3399) 'Activation checks real listener'
& $Script -Action Rollback -Transaction $s.ID
Assert ($global:fake.Port -eq 3389 -and $global:fake.Listening -eq 3389) 'Port recovery restores listener'

Reset-Fixture
Apply 'whitelist' '192.0.2.1/32'
$s=State
$global:fake.Rules['Existing-RDP'].RemoteAddress=@('203.0.113.9/32')
Expect-Failure { & $Script -Action Rollback -Transaction $s.ID } '*Concurrent firewall change*'
Assert ((State).Phase -eq 'rollback-failed') 'Failed recovery remains pending for retry'
Assert ($global:fake.Rules.Count -eq 3) 'Temporary access is retained when original-rule restoration fails'
Assert ($global:fake.Rules['Existing-RDP'].RemoteAddress[0] -eq '203.0.113.9/32') 'Recovery does not overwrite another administrator'
$global:fake.Rules['Existing-RDP'].RemoteAddress=@('192.0.2.1/32')
& $Script -Action Rollback -Transaction $s.ID
Assert ((State).Phase -eq 'rolled-back') 'Recovery retries complete after conflict is resolved'

Reset-Fixture
Apply 'nla' '1'
$s=State
Assert ($global:fake.NLA -eq 1) 'NLA change submitted'
& $Script -Action Rollback -Transaction $s.ID
Assert ($global:fake.NLA -eq 0) 'NLA rollback restores original value'

Reset-Fixture
$global:fake.Firewall=$false
Apply 'firewall' '1'
$s=State
Assert $global:fake.Firewall 'Firewall change submitted'
& $Script -Action Rollback -Transaction $s.ID
Assert (!$global:fake.Firewall) 'Firewall rollback restores original profile states'

Reset-Fixture
$event=[pscustomobject]@{TimeCreated=(Get-Date);Id=24;Message='Disconnected';}
$event | Add-Member ScriptMethod ToXml { return '<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><UserData><EventXML><User>Administrator</User><SessionID>22</SessionID><Address>192.0.2.1</Address></EventXML></UserData></Event>' }
$global:fake.Events=@($event)
$logs=(& $Script -Action Events | ConvertFrom-Json)
Assert (@($logs.events).Count -gt 0 -and $logs.events[0].address -eq '192.0.2.1') 'Namespaced event XML is parsed without translated-message matching'
Write-Output 'RDP transaction scenarios passed'

Reset-Fixture
$global:fake.LogEnabled=$false
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*event log is disabled*'
Assert ($global:fake.Writes -eq 0) 'Unreadable confirmation evidence fails before mutation'

Reset-Fixture
$global:fake.DomainRole=5
Expect-Failure { Apply 'nla' '1' } '*Domain controllers*'
Assert ($global:fake.Writes -eq 0) 'Domain controllers remain read only'

foreach ($bad in @('old-record','old-time','future-time','wrong-session','console','missing-task','disabled-task','changed-task','changed-principal','disabled-rule','scope-bypass','firewall-off','expired','event-access')) {
  Reset-Fixture
  Apply 'whitelist' '192.0.2.1/32'
  $s=State; $global:fake.Reconnected=$true; $remote='true'; $pattern='*Reconnect with RDP*'
  switch ($bad) {
    'old-record' { $global:fake.EventRecord=99 }
    'old-time' { $global:fake.EventAge=-30 }
    'future-time' { $global:fake.EventAge=60 }
    'wrong-session' { $global:fake.EventSession=77 }
    'console' { $remote='false' }
    'missing-task' { $global:fake.Tasks.Remove('WinToolBox-RdpSafety-Recover'); $pattern='*Recovery task*' }
    'disabled-task' { $global:fake.Tasks['WinToolBox-RdpSafety-Recover'].State='Disabled'; $pattern='*Recovery task*' }
    'changed-task' { $global:fake.Tasks['WinToolBox-RdpSafety-Recover'].Actions[0].Arguments='changed'; $pattern='*Recovery task*' }
    'changed-principal' { $global:fake.Tasks['WinToolBox-RdpSafety-Recover'].Principal.UserId='another-user'; $pattern='*Recovery task identity*' }
    'disabled-rule' { $global:fake.Rules[$s.Created[0]].Enabled='False'; $pattern='*Protected rule changed*' }
    'scope-bypass' { $global:fake.Rules['Existing-RDP'].RemoteAddress=@('Any'); $pattern='*bypasses the whitelist*' }
    'firewall-off' { $global:fake.Firewall=$false; $pattern='*firewall profiles*' }
    'expired' { $s.Deadline=[datetime]::UtcNow.AddSeconds(-1); $s | Export-Clixml -LiteralPath (Join-Path $root 'pending.xml'); $pattern='*expired*' }
    'event-access' { $global:fake.EventDenied=$true; $pattern='*Injected reconnect log failure*' }
  }
  Expect-Failure { & $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession $remote } $pattern
  Assert ((State).Phase -eq 'awaiting-confirmation') ('Rejected confirmation retains recovery: ' + $bad)
}

Reset-Fixture
Apply 'whitelist' '2001:db8::/64,192.0.2.1/32'
$s=State; $global:fake.Reconnected=$true
& $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession true
Assert ((State).Phase -eq 'confirmed') 'Mixed IPv4 and IPv6 sources can be confirmed'

Reset-Fixture
Apply 'port' '3399'
$s=State; & $Script -Action Activate -Transaction $s.ID
$global:fake.RestartFail=$true
Expect-Failure { & $Script -Action Rollback -Transaction $s.ID } '*restart failure*'
Assert ($global:fake.Rules.Count -eq 3) 'New port access remains when old listener cannot restart'
$global:fake.RestartFail=$false
& $Script -Action Rollback -Transaction $s.ID
Assert ((State).Phase -eq 'rolled-back' -and $global:fake.Listening -eq 3389) 'Failed listener recovery can retry'
Write-Output 'Extended RDP abnormal scenarios passed'

Reset-Fixture
$global:fake.ActivateFail=$true
Expect-Failure { Apply 'port' '3399' } '*previous configuration restored*'
Assert ($global:fake.Port -eq 3389 -and $global:fake.Rules.Count -eq 1 -and (State).Phase -eq 'rolled-back') 'Activation task registration failure restores the full change'

Reset-Fixture
$global:fake.Policy=[pscustomobject]@{UserAuthentication=1}
Expect-Failure { Apply 'nla' '0' } '*managed by policy*'
Assert ($global:fake.Writes -eq 0) 'Managed NLA policy is not overwritten'

Reset-Fixture
$global:fake.LocalRules='False'
Expect-Failure { Apply 'whitelist' '192.0.2.1/32' } '*local rule merging*'
Assert ($global:fake.Writes -eq 0) 'Disabled rule merging fails before writes'

Reset-Fixture
Apply 'whitelist' '192.0.2.1/32'
$s=State; $global:fake.Reconnected=$true
& $Script -Action Confirm -Transaction $s.ID -SessionID 22 -RemoteSession true
$global:fake.Reconnected=$false
Apply 'whitelist' '198.51.100.2/32'
$s=State
& $Script -Action Rollback -Transaction $s.ID
$policy=Import-Clixml -LiteralPath (Join-Path $root 'whitelist.xml')
Assert ($policy.Addresses[0] -eq '192.0.2.1/32' -and $global:fake.Rules.Count -eq 3) 'Editing an existing whitelist restores prior policy and rules'
$global:fake.Firewall=$false
Expect-Failure { Apply 'port' '3399' } '*existing whitelist is not enforced*'
Write-Output 'RDP policy and multi-transaction scenarios passed'

Reset-Fixture
Apply 'port' '3399'
$s=State; & $Script -Action Activate -Transaction $s.ID
$s.Deadline=[datetime]::UtcNow.AddSeconds(-1); $s | Export-Clixml -LiteralPath (Join-Path $root 'pending.xml')
$global:fake.PersistFail=$true
Expect-Failure { & $Script -Action Recover -Transaction $s.ID } '*Original configuration restored*'
Assert ($global:fake.Port -eq 3389 -and $global:fake.Listening -eq 3389 -and $global:fake.Rules.Count -eq 1) 'Disk write failure cannot block restoring original connectivity'
Assert ($global:fake.Tasks.ContainsKey('WinToolBox-RdpSafety-Recover')) 'Recovery task retained when final state cannot be persisted'
$global:fake.PersistFail=$false
& $Script -Action Recover -Transaction $s.ID
Assert ((State).Phase -eq 'rolled-back') 'Recovery can finish after persistence recovers'
Write-Output 'RDP recovery persistence failure scenario passed'
