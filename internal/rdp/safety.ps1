param(
  [ValidateSet('Status','Apply','Confirm','Rollback','Recover','Activate','Guard','Events')][string]$Action,
  [string]$Kind = '', [string]$Value = '', [string]$Transaction = '',
  [uint32]$SessionID = 0, [string]$RemoteSession = 'false'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2
$root = Split-Path -Parent $PSCommandPath
$stateFile = Join-Path $root 'pending.xml'
$policyFile = Join-Path $root 'whitelist.xml'
$taskName = 'WinToolBox-RdpSafety-Recover'
$activateName = 'WinToolBox-RdpSafety-Activate'
$rdpKey = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp'
$tsKey = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'

function Save-State($state, [string]$path = $stateFile) {
  $temp = $path + '.' + [guid]::NewGuid().ToString('N') + '.tmp'
  try {
    $state | Export-Clixml -LiteralPath $temp -Depth 12
    if (Test-Path -LiteralPath $path) { [IO.File]::Replace($temp, $path, [NullString]::Value) }
    else { [IO.File]::Move($temp, $path) }
  } finally { if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force } }
}
function Read-State {
  if (Test-Path -LiteralPath $stateFile) { return Import-Clixml -LiteralPath $stateFile }
  return $null
}
function Write-Audit([string]$message) {
  # Logging must never prevent restoring connectivity when the disk is full.
  try { Add-Content -LiteralPath (Join-Path $root 'recovery.log') -Value (([datetime]::UtcNow.ToString('o')) + ' ' + $message) -Encoding UTF8 } catch {}
}
function Port-Matches($ports, [int]$port) {
  foreach ($entry in @($ports)) {
    foreach ($part in ([string]$entry -split ',')) {
      if ($part -eq 'Any' -or $part -eq [string]$port) { return $true }
      if ($part -match '^(\d+)-(\d+)$' -and $port -ge [int]$Matches[1] -and $port -le [int]$Matches[2]) { return $true }
      # Dynamic RPC and symbolic ports cannot be resolved safely by this tool.
    }
  }
  return $false
}
function Get-Port { return [int](Get-ItemProperty -LiteralPath $rdpKey -Name PortNumber).PortNumber }
function Get-Candidates([int]$port, [string]$store) {
  foreach ($filter in @(Get-NetFirewallPortFilter -PolicyStore $store)) {
    if ([string]$filter.Protocol -notin @('TCP','UDP','Any','6','17','256')) { continue }
    if (!(Port-Matches $filter.LocalPort $port)) { continue }
    foreach ($rule in @($filter | Get-NetFirewallRule | Where-Object { [string]$_.Direction -eq 'Inbound' -and [string]$_.Enabled -eq 'True' })) {
    $address = $rule | Get-NetFirewallAddressFilter
    $service = $rule | Get-NetFirewallServiceFilter
    $app = $rule | Get-NetFirewallApplicationFilter
    # Service-specific rules for other Windows services cannot authorize RDP.
    if ([string]$service.Service -notin @('Any','TermService','')) { continue }
    $program = [Environment]::ExpandEnvironmentVariables([string]$app.Program)
    if ($program -notin @('Any','System','') -and [IO.Path]::GetFileName($program) -ne 'svchost.exe') { continue }
    [pscustomobject]@{
      Rule = $rule; Filter = $filter; Addresses = @($address.RemoteAddress)
      Service = [string]$service.Service; Program = [string]$app.Program
    }
    }
  }
}
function Test-Listening([int]$port) {
  $svc = Get-CimInstance Win32_Service -Filter "Name='TermService'"
  if ($svc.State -ne 'Running' -or $svc.ProcessId -eq 0) { return $false }
  return @((Get-NetTCPConnection -State Listen -ErrorAction Stop) | Where-Object { $_.LocalPort -eq $port -and $_.OwningProcess -eq $svc.ProcessId }).Count -gt 0
}
function Test-Reconnected($state) {
  if ($RemoteSession -ne 'true') { return $false }
  $errors=@()
  $events = @(Get-WinEvent -FilterHashtable @{
    LogName='Microsoft-Windows-TerminalServices-LocalSessionManager/Operational'
    Id=21,25; StartTime=([datetime]$state.Applied).ToLocalTime()
  } -MaxEvents 200 -ErrorAction SilentlyContinue -ErrorVariable errors)
  foreach($item in $errors){if($item.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*'){throw ('Cannot read reconnection evidence: '+$item.Exception.Message)}}
  foreach ($event in $events) {
    if ($event.Id -notin @(21,25) -or $event.RecordId -le $state.BaselineRecordID -or $event.TimeCreated.ToUniversalTime() -gt [datetime]::UtcNow.AddSeconds(2)) { continue }
    $xml = [xml]$event.ToXml()
    foreach ($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="UserData"]/*/*'))) {
      if ($node.LocalName -eq 'SessionID' -and [string]$node.InnerText -eq [string]$SessionID -and $event.TimeCreated.ToUniversalTime() -gt [datetime]$state.Applied) { return $true }
    }
  }
  return $false
}
function Reconnect-Baseline {
  $channel = 'Microsoft-Windows-TerminalServices-LocalSessionManager/Operational'
  if (!(Get-WinEvent -ListLog $channel).IsEnabled) { throw 'Reconnect event log is disabled. Enable it before protected changes.' }
  $errors = @()
  $latest = @(Get-WinEvent -LogName $channel -MaxEvents 1 -ErrorAction SilentlyContinue -ErrorVariable errors)
  foreach ($item in $errors) { if ($item.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') { throw $item } }
  if ($latest.Count) { return [long]$latest[0].RecordId }
  return [long]0
}
function Assert-RecoveryTask($state) {
  $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
  if (!$task -or [string]$task.State -eq 'Disabled' -or @($task.Actions).Count -ne 1 -or $task.Actions[0].Arguments -notlike ('*-Transaction ' + $state.ID) -or $task.Actions[0].Arguments -notlike '*-Action Recover *') { throw 'Recovery task is missing, disabled or changed. Restore from the console.' }
  $expected=Task-Action 'Recover' $state.ID
  if ($task.Actions[0].Execute -ne $expected.Execute -or $task.Actions[0].Arguments -ne $expected.Arguments -or [string]$task.Principal.UserId -notin @('SYSTEM','S-1-5-18','NT AUTHORITY\SYSTEM')) { throw 'Recovery task identity or action changed. Restore from the console.' }
}
function Assert-EffectiveAccess($state) {
  if ($state.Kind -eq 'whitelist' -or $null -ne $state.OldPolicy -or $state.Kind -eq 'firewall') {
    if (@(Get-NetFirewallProfile -PolicyStore ActiveStore | Where-Object { !$_.Enabled -or [string]$_.DefaultInboundAction -ne 'Block' -or [string]$_.AllowLocalFirewallRules -eq 'False' }).Count) { throw 'Effective firewall profiles do not enforce the requested configuration.' }
  }
  if ($state.Kind -eq 'whitelist' -or $null -ne $state.OldPolicy) {
    foreach ($candidate in @(Get-Candidates (Get-Port) 'ActiveStore')) {
      if ([string]$candidate.Rule.Action -eq 'Allow' -and !(Same-Values $candidate.Addresses $state.Addresses)) { throw ('Another effective rule bypasses the whitelist: ' + $candidate.Rule.Name) }
    }
  }
  foreach ($name in @($state.Created)) {
    $rule = Get-NetFirewallRule -PolicyStore ActiveStore -Name $name
    $filter = $rule | Get-NetFirewallPortFilter
    $addresses = @(($rule | Get-NetFirewallAddressFilter).RemoteAddress)
    $service = [string]($rule | Get-NetFirewallServiceFilter).Service
    $protocol = 'TCP'; if ($name.EndsWith('-UDP')) { $protocol = 'UDP' }
    if ([string]$rule.Enabled -ne 'True' -or [string]$rule.Action -ne 'Allow' -or [string]$rule.Direction -ne 'Inbound' -or $service -ne 'TermService' -or [string]$filter.Protocol -notin @($protocol, $(if ($protocol -eq 'TCP') {'6'} else {'17'})) -or [string]$filter.LocalPort -ne [string](Get-Port) -or !(Same-Values $addresses $state.Addresses)) { throw ('Protected rule changed or is not effective: ' + $name) }
  }
}
function Task-Action([string]$verb, [string]$id) {
  $shell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
  return New-ScheduledTaskAction -Execute $shell -Argument ('-NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + $PSCommandPath + '" -Action ' + $verb + ' -Transaction ' + $id)
}
function Register-Recovery($state) {
  $once = New-ScheduledTaskTrigger -Once -At ([datetime]$state.Deadline).ToLocalTime() -RepetitionInterval (New-TimeSpan -Minutes 1)
  $startup = New-ScheduledTaskTrigger -AtStartup
  $settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 5) -MultipleInstances IgnoreNew
  Register-ScheduledTask -TaskName $taskName -Action (Task-Action 'Recover' $state.ID) -Trigger @($once,$startup) -Settings $settings -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
  $task = Get-ScheduledTask -TaskName $taskName
  if ([string]$task.State -eq 'Disabled' -or !$task.Triggers.Count) { throw 'Recovery task was not armed; no configuration was changed.' }
  Assert-RecoveryTask $state
}
function Canonical-Address([string]$value) {
  $parts = $value.Split('/')
  $ip = [Net.IPAddress]::None
  if (![Net.IPAddress]::TryParse($parts[0], [ref]$ip)) { return $value.ToLowerInvariant() }
  $bytes = $ip.GetAddressBytes()
  $bits = $bytes.Length * 8
  if ($parts.Count -gt 1) {
    if ($parts[1] -match '^\d+$') { $bits = [int]$parts[1] }
    else {
      $mask = [Net.IPAddress]::None
      if (![Net.IPAddress]::TryParse($parts[1], [ref]$mask)) { return $value.ToLowerInvariant() }
      $binary = ($mask.GetAddressBytes() | ForEach-Object { [Convert]::ToString($_,2).PadLeft(8,'0') }) -join ''
      if ($binary -notmatch '^1*0*$') { return $value.ToLowerInvariant() }
      $bits = ($binary -replace '0','').Length
    }
  }
  if ($bits -lt 0 -or $bits -gt $bytes.Length * 8) { return $value.ToLowerInvariant() }
  for ($i=0; $i -lt $bytes.Length; $i++) {
    $remaining = [Math]::Max(0, [Math]::Min(8, $bits - $i*8))
    $bytes[$i] = $bytes[$i] -band ((255 -shl (8-$remaining)) -band 255)
  }
  return ([Net.IPAddress]::new([byte[]]$bytes)).ToString().ToLowerInvariant() + '/' + $bits
}
function Same-Values($a, $b) {
  return (@($a | ForEach-Object { Canonical-Address ([string]$_) } | Sort-Object -Unique) -join ',') -eq (@($b | ForEach-Object { Canonical-Address ([string]$_) } | Sort-Object -Unique) -join ',')
}
function Restore-State($state) {
  $state.Phase = 'rolling-back'
  try { Save-State $state } catch { Write-Audit ('Recovery status write failed; attempting original configuration anyway: ' + $_.Exception.Message) }
  $failures = @()
  foreach ($saved in @($state.Rules)) {
    try {
      $rule = Get-NetFirewallRule -PolicyStore PersistentStore -Name $saved.Name
      $current = @(($rule | Get-NetFirewallAddressFilter).RemoteAddress)
      if (Same-Values $current $saved.After) { $rule | Set-NetFirewallRule -RemoteAddress $saved.Before | Out-Null }
      elseif (!(Same-Values $current $saved.Before)) { throw ('Concurrent firewall change: ' + $saved.Name) }
    } catch { $failures += $_.Exception.Message }
  }
  if ($state.Kind -eq 'whitelist' -and (Test-Path -LiteralPath $policyFile)) {
    try {
      $currentPolicy = Import-Clixml -LiteralPath $policyFile
      if (Same-Values $currentPolicy.Addresses $state.Addresses) {
        if ($null -ne $state.OldPolicy) { Save-State $state.OldPolicy $policyFile }
        else { Remove-Item -LiteralPath $policyFile -Force }
      } elseif ($null -eq $state.OldPolicy -or !(Same-Values $currentPolicy.Addresses $state.OldPolicy.Addresses)) { throw 'Concurrent whitelist metadata change.' }
    } catch { $failures += $_.Exception.Message }
  }
  try {
    if ($state.Kind -eq 'port') {
      $current = Get-Port
      if ($current -eq [int]$state.Value) { Set-ItemProperty -LiteralPath $rdpKey -Name PortNumber -Type DWord -Value $state.OldPort }
      elseif ($current -ne [int]$state.OldPort) { throw 'Concurrent RDP port change; manual recovery required.' }
      if (!(Test-Listening $state.OldPort)) { Restart-Service TermService -Force }
      if (!(Test-Listening $state.OldPort)) { throw 'Old RDP listener not ready; recovery will retry. A console reboot may be necessary.' }
    }
    if ($state.Kind -eq 'nla') {
      $current = [int](Get-ItemProperty -LiteralPath $rdpKey -Name UserAuthentication).UserAuthentication
      if ($current -eq [int]$state.Value) { Set-ItemProperty -LiteralPath $rdpKey -Name UserAuthentication -Type DWord -Value $state.OldNLA }
      elseif ($current -ne [int]$state.OldNLA) { throw 'Concurrent NLA change; manual recovery required.' }
    }
    if ($state.Kind -eq 'firewall') {
      foreach ($profile in @($state.Profiles)) {
        # Profiles already enabled before the change were never changed by us.
        if ([string]$profile.Enabled -ne 'True') { Set-NetFirewallProfile -Profile $profile.Name -Enabled $profile.Enabled }
      }
    }
  } catch { $failures += $_.Exception.Message }
  # Retain transaction-owned access until original rules/listener are restored.
  # Deleting these first can destroy the only remaining working RDP path.
  if (!$failures.Count) {
    foreach ($name in @($state.Created)) {
      try { Get-NetFirewallRule -PolicyStore PersistentStore -Name $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule }
      catch { $failures += $_.Exception.Message }
    }
  }
  if ($failures.Count) {
    $state.Phase = 'rollback-failed'
    try { Save-State $state } catch { $failures += ('Recovery state write failed: ' + $_.Exception.Message) }
    Write-Audit ($state.ID + ' rollback failed: ' + ($failures -join '; '))
    throw ($failures -join '; ')
  }
  $state.Phase = 'rolled-back'
  try { Save-State $state } catch {
    Write-Audit ($state.ID + ' original configuration restored; final state write failed, recovery task retained')
    throw ('Original configuration restored, but recovery state could not be saved; task retained for retry: ' + $_.Exception.Message)
  }
  Write-Audit ($state.ID + ' restored')
  Unregister-ScheduledTask -TaskName $activateName -Confirm:$false -ErrorAction SilentlyContinue
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
}
function Is-Pending($state) { return $null -ne $state -and $state.Phase -notin @('confirmed','rolled-back') }
function Assert-NoPending($state) { if (Is-Pending $state) { throw 'A protected change is pending. Reconnect and confirm, or restore it first.' } }
function Get-Health($state) {
  $port = Get-Port
  $warnings = @('Local checks do not verify cloud security groups, NAT, gateways or reachability from your client.')
  $profiles = @(Get-NetFirewallProfile -PolicyStore ActiveStore)
  $rules = @()
  foreach ($candidate in @(Get-Candidates $port 'ActiveStore')) {
    $r = $candidate.Rule
    $rules += [pscustomobject]@{
      name=[string]$r.Name; action=[string]$r.Action; protocol=[string]$candidate.Filter.Protocol
      ports=(@($candidate.Filter.LocalPort) -join ','); addresses=($candidate.Addresses -join ',')
      source=[string]$r.PolicyStoreSourceType; profile=[string]$r.Profile
    }
    if ([string]$r.Action -eq 'Block') { $warnings += ('Potential blocking rule (check all filters): ' + $r.Name) }
    if ($candidate.Addresses -contains 'Any' -and [string]$r.Action -eq 'Allow') { $warnings += ('Potential unrestricted allow rule (check all filters): ' + $r.Name) }
  }
  if (@($profiles | Where-Object { !$_.Enabled -or [string]$_.DefaultInboundAction -ne 'Block' }).Count) { $warnings += 'Some firewall profiles are disabled or do not default to blocking inbound traffic.' }
  $pending = Is-Pending $state
  $listening = Test-Listening $port
  $canConfirm = $false
  if ($pending -and [datetime]::UtcNow -lt [datetime]$state.Deadline -and $state.Phase -eq 'awaiting-confirmation' -and $listening) {
    $canConfirm = Test-Reconnected $state
  }
  if ($pending) {
    try { Assert-RecoveryTask $state; if ($canConfirm) { Assert-EffectiveAccess $state } }
    catch { $warnings += $_.Exception.Message; $canConfirm = $false }
  }
  [pscustomobject]@{
    pending=$pending; transaction=$(if ($pending) { $state.ID } else { '' })
    deadline=$(if ($pending) { ([datetime]$state.Deadline).ToUniversalTime().ToString('o') } else { '' })
    phase=$(if ($state) { $state.Phase } else { 'idle' }); kind=$(if ($state) { $state.Kind } else { '' })
    addresses=@($(if (Test-Path -LiteralPath $policyFile) { (Import-Clixml -LiteralPath $policyFile).Addresses }))
    port=$port; listening=$listening; service=[string](Get-Service TermService).Status
    profiles=(($profiles | ForEach-Object { $_.Name + '=' + $_.Enabled + '/' + $_.DefaultInboundAction }) -join '; ')
    rules=@($rules); warnings=@($warnings); canConfirm=$canConfirm
  }
}
function Read-Events {
  $items = @(); $warnings = @()
  $sources = @(
    @{Name='Microsoft-Windows-TerminalServices-LocalSessionManager/Operational'; IDs=@(21,22,23,24,25,39,40)},
    @{Name='Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational'; IDs=@(1149)},
    @{Name='Security'; IDs=@(4624,4625,4634,4778,4779)}
  )
  foreach ($source in $sources) {
    try {
      $log = Get-WinEvent -ListLog $source.Name
      if (!$log.IsEnabled) { $warnings += ('Log disabled: ' + $source.Name); continue }
      $errors = @()
      $events = @(Get-WinEvent -FilterHashtable @{LogName=$source.Name; Id=$source.IDs; StartTime=(Get-Date).AddDays(-7)} -MaxEvents 150 -ErrorAction SilentlyContinue -ErrorVariable errors)
      foreach ($errorItem in $errors) { if ($errorItem.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') { $warnings += ($source.Name + ': ' + $errorItem.Exception.Message) } }
      foreach ($event in $events) {
        $xml = [xml]$event.ToXml(); $data = @{}
        foreach ($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="EventData"]/*[local-name()="Data"]'))) { $data[[string]$node.Name] = [string]$node.InnerText }
        foreach ($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="UserData"]/*/*'))) { $data[$node.LocalName] = [string]$node.InnerText }
        if ($source.Name -eq 'Security' -and $event.Id -in @(4624,4625,4634) -and $data['LogonType'] -notin @('10','3')) { continue }
        $user = @($data['TargetUserName'],$data['User'],$data['Param1'],$data['AccountName']) | Where-Object { $_ } | Select-Object -First 1
        $address = @($data['IpAddress'],$data['Address'],$data['Param3'],$data['ClientAddress']) | Where-Object { $_ } | Select-Object -First 1
        $items += [pscustomobject]@{
          time=$event.TimeCreated.ToUniversalTime().ToString('o'); id=[int]$event.Id; channel=$source.Name
          user=[string]$user; address=[string]$address; session=[string]$data['SessionID']
          detail=[string]$event.Message
        }
      }
    } catch { $warnings += ($source.Name + ': ' + $_.Exception.Message) }
  }
  [pscustomobject]@{ events=@($items | Sort-Object time -Descending | Select-Object -First 250); warnings=@($warnings + '1149 means authentication, not a completed desktop logon. Security type 3 can include non-RDP network logons. Missing events do not prove no activity.') } | ConvertTo-Json -Depth 5 -Compress
}

if ($Action -eq 'Events') { Read-Events; return }
# Global serialization includes recovery and confirmation in separate processes.
$mutex = New-Object Threading.Mutex($false, 'Global\WinToolBox-RdpSafety-v1')
$locked = $false
try {
  try { $locked = $mutex.WaitOne(15000) } catch [Threading.AbandonedMutexException] { $locked = $true }
  if (!$locked) { throw 'Another RDP safety operation is running.' }
  $state = Read-State
  if ($Action -eq 'Guard') {
    Assert-NoPending $state
    if (Test-Path -LiteralPath $policyFile) { throw 'An RDP whitelist is active. Use the protected RDP workflow; generic network changes are blocked to avoid bypassing it.' }
    return
  }
  if ($Action -eq 'Status') {
    try { Get-Health $state | ConvertTo-Json -Depth 6 -Compress }
    catch {
      # A broken CIM/firewall provider must not hide an armed recovery task.
      [pscustomobject]@{
        pending=(Is-Pending $state); transaction=$(if (Is-Pending $state) { $state.ID } else { '' })
        deadline=$(if (Is-Pending $state) { ([datetime]$state.Deadline).ToUniversalTime().ToString('o') } else { '' })
        phase=$(if ($state) { $state.Phase } else { 'unknown' }); kind=$(if ($state) { $state.Kind } else { '' })
        addresses=@(); port=0; listening=$false; service='unknown'; profiles='unknown'
        rules=@(); warnings=@($_.Exception.Message); canConfirm=$false
      } | ConvertTo-Json -Depth 5 -Compress
    }
    return
  }
  if ($Action -in @('Recover','Rollback','Confirm','Activate')) {
    if (!(Is-Pending $state)) {
      if ($Action -in @('Confirm','Rollback')) { throw 'No pending change remains; refresh its final status.' }
      return
    }
    if ($state.ID -ne $Transaction) { throw 'Stale transaction; no change was made.' }
    if ($Action -eq 'Recover') {
      if ([datetime]::UtcNow -ge [datetime]$state.Deadline) { Restore-State $state }
      return
    }
    if ($Action -eq 'Rollback') { Restore-State $state; return }
    if ($Action -eq 'Activate') {
      if ($state.Kind -ne 'port' -or $state.Phase -ne 'awaiting-confirmation') { return }
      if ([datetime]::UtcNow -ge [datetime]$state.Deadline) { Restore-State $state; return }
      try {
        $state.Applied = [datetime]::UtcNow; Save-State $state
        Restart-Service TermService -Force
        if (!(Test-Listening ([int]$state.Value))) { throw 'New listener is not ready.' }
        Write-Audit ($state.ID + ' new listener ready; waiting for reconnection')
      } catch { Write-Audit ($state.ID + ' activation failed: ' + $_.Exception.Message); Restore-State $state }
      return
    }
    if ([datetime]::UtcNow -ge [datetime]$state.Deadline -or $state.Phase -ne 'awaiting-confirmation') { throw 'Confirmation expired or recovery started.' }
    Assert-RecoveryTask $state
    if (!(Test-Reconnected $state)) { throw 'Reconnect with RDP after this change before confirming. Existing sessions and console clicks cannot confirm external access.' }
    if (!(Test-Listening (Get-Port))) { throw 'The RDP listener is not running.' }
    if ($state.Kind -eq 'port' -and (Get-Port) -ne [int]$state.Value) { throw 'RDP port changed unexpectedly.' }
    if ($state.Kind -eq 'nla' -and [int](Get-ItemProperty -LiteralPath $rdpKey -Name UserAuthentication).UserAuthentication -ne [int]$state.Value) { throw 'NLA changed unexpectedly.' }
    Assert-EffectiveAccess $state
    if ($state.Kind -eq 'whitelist') {
      foreach ($saved in @($state.Rules)) {
        $rule = Get-NetFirewallRule -PolicyStore ActiveStore -Name $saved.Name
        if (!(Same-Values @(($rule | Get-NetFirewallAddressFilter).RemoteAddress) $state.Addresses)) { throw 'Effective address filters differ; cannot confirm.' }
      }
      foreach ($candidate in @(Get-Candidates (Get-Port) 'ActiveStore')) {
        if ([string]$candidate.Rule.Action -eq 'Allow' -and !(Same-Values $candidate.Addresses $state.Addresses)) { throw ('Another effective rule bypasses the whitelist: ' + $candidate.Rule.Name) }
      }
      Save-State ([pscustomobject]@{Addresses=@($state.Addresses)}) $policyFile
    }
    if ([datetime]::UtcNow -ge [datetime]$state.Deadline) { throw 'Confirmation expired during verification; recovery remains armed.' }
    $state.Phase = 'confirmed'; Save-State $state
    Write-Audit ($state.ID + ' confirmed after RDP reconnection')
    Unregister-ScheduledTask -TaskName $activateName -Confirm:$false -ErrorAction SilentlyContinue
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    return
  }
  Assert-NoPending $state
  if ($Kind -notin @('port','whitelist','nla','firewall')) { throw 'Unsupported protected change.' }
  if ($SessionID -eq 0) { throw 'Interactive session required.' }
  if ([int](Get-CimInstance Win32_ComputerSystem).DomainRole -ge 4) { throw 'Domain controllers require domain-managed RDP policies; no changes were made.' }
  $policy = Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services' -ErrorAction SilentlyContinue
  if ($policy -and @($policy.PSObject.Properties.Name | Where-Object { $_ -in @('fDenyTSConnections','UserAuthentication','SecurityLayer') }).Count) { throw 'RDP is managed by policy. Change the policy through your administrator first.' }
  if ([int](Get-ItemProperty -LiteralPath $tsKey -Name fDenyTSConnections).fDenyTSConnections -ne 0) { throw 'Enable Remote Desktop from the console before using protected changes.' }
  if ($Kind -eq 'port' -and ($Value -notmatch '^\d+$' -or [int]$Value -lt 1024 -or [int]$Value -gt 65535)) { throw 'Protected RDP port must be between 1024 and 65535.' }
  if ($Kind -eq 'nla' -and $Value -notin @('0','1')) { throw 'Invalid NLA value.' }
  $oldPort = Get-Port
  $baseline = Reconnect-Baseline
  if (!(Test-Listening $oldPort)) { throw 'The original RDP listener cannot be verified. Repair it from the console before changing connectivity.' }
  if ($Kind -eq 'port' -and [int]$Value -eq $oldPort) { throw 'The RDP port is unchanged.' }
  if ($Kind -eq 'port' -and (@(Get-NetTCPConnection -State Listen -ErrorAction Stop | Where-Object LocalPort -eq ([int]$Value)).Count -or @(Get-NetUDPEndpoint -ErrorAction Stop | Where-Object LocalPort -eq ([int]$Value)).Count)) { throw 'The new port is already in use.' }
  $addresses = @('Any')
  if (Test-Path -LiteralPath $policyFile) { $addresses = @((Import-Clixml -LiteralPath $policyFile).Addresses) }
  if ($Kind -eq 'whitelist') {
    $addresses = @($Value -split ',')
    if (!$addresses.Count -or $addresses.Count -gt 64 -or $Value -match '[^0-9a-fA-F:.,/]') { throw 'Invalid whitelist.' }
  }
  $profiles = @(Get-NetFirewallProfile -PolicyStore ActiveStore)
  if ((Test-Path -LiteralPath $policyFile) -and @($profiles | Where-Object { !$_.Enabled -or [string]$_.DefaultInboundAction -ne 'Block' -or [string]$_.AllowLocalFirewallRules -eq 'False' }).Count) { throw 'The existing whitelist is not enforced by the effective firewall profiles.' }
  if ($Kind -eq 'firewall' -and !@((Get-Candidates $oldPort 'ActiveStore') | Where-Object { [string]$_.Rule.Action -eq 'Allow' }).Count) { throw 'No enabled RDP allow rule was found. Prepare a rule from the console before enabling the firewall.' }
  if ($Kind -eq 'whitelist' -and @($profiles | Where-Object { !$_.Enabled -or [string]$_.DefaultInboundAction -ne 'Block' -or [string]$_.AllowLocalFirewallRules -eq 'False' }).Count) { throw 'Whitelist requires all effective firewall profiles enabled, default inbound Block, and local rule merging permitted.' }
  $rules = @()
  if ($Kind -in @('whitelist','port')) {
    $oldCandidates = @(Get-Candidates $oldPort 'ActiveStore')
    foreach ($candidate in $oldCandidates) {
      $rule = $candidate.Rule
      if ([string]$rule.Action -ne 'Allow') { continue }
      if ([string]$rule.PolicyStoreSourceType -ne 'Local') { throw ('A matching managed allow rule must be changed by its administrator first: ' + $rule.Name) }
      $ports = @($candidate.Filter.LocalPort)
      if ($candidate.Service -ne 'TermService' -and ($ports.Count -ne 1 -or [string]$ports[0] -ne [string]$oldPort)) { throw ('Broad/ambiguous allow rule requires manual scoping first: ' + $rule.Name) }
      if ($Kind -eq 'whitelist') { $rules += [pscustomobject]@{Name=$rule.Name; Before=@($candidate.Addresses); After=@($addresses)} }
    }
    if ($Kind -eq 'port') {
      $allowCandidates = @($oldCandidates | Where-Object { [string]$_.Rule.Action -eq 'Allow' })
      if ($allowCandidates.Count -and !(Test-Path -LiteralPath $policyFile)) { $addresses = @($allowCandidates[0].Addresses) }
      foreach ($candidate in $allowCandidates) {
        if (!(Same-Values $candidate.Addresses $addresses)) { throw 'Existing RDP rules have different source scopes. Apply a consistent whitelist before changing the port.' }
      }
      foreach ($candidate in @(Get-Candidates ([int]$Value) 'ActiveStore')) {
        if ([string]$candidate.Rule.Action -eq 'Block') { throw ('A rule may block the new port: ' + $candidate.Rule.Name) }
        if ([string]$candidate.Rule.Action -eq 'Allow' -and !(Same-Values $candidate.Addresses $addresses)) { throw ('A conflicting allow rule already covers the new port: ' + $candidate.Rule.Name) }
      }
    }
  }
  $id = [guid]::NewGuid().ToString('N')
  $created = @()
  if ($Kind -in @('port','whitelist')) { $created = @(('WinToolBox-RdpSafe-' + $id + '-TCP'), ('WinToolBox-RdpSafe-' + $id + '-UDP')) }
  $state = [pscustomobject]@{
    ID=$id; Started=[datetime]::UtcNow; Applied=[datetime]::MaxValue; Deadline=[datetime]::UtcNow.AddMinutes(5); BaselineRecordID=$baseline
    Phase='armed'; Kind=$Kind; Value=$Value; OldPort=$oldPort
    OldNLA=[int](Get-ItemProperty -LiteralPath $rdpKey -Name UserAuthentication).UserAuthentication
    Rules=@($rules); Created=@($created); Addresses=@($addresses)
    OldPolicy=$(if (Test-Path -LiteralPath $policyFile) { Import-Clixml -LiteralPath $policyFile } else { $null })
    Profiles=@(Get-NetFirewallProfile | Select-Object Name,Enabled)
  }
  Save-State $state
  # Nothing below changes connectivity until the independent SYSTEM task is armed.
  try { Register-Recovery $state }
  catch {
    $state.Phase = 'rolled-back'; Save-State $state
    throw ('Recovery task could not be armed; no configuration changed: ' + $_.Exception.Message)
  }
  Write-Audit ($id + ' armed ' + $Kind)
  try {
    if ($Kind -in @('port','whitelist')) {
      $targetPort = $oldPort
      if ($Kind -eq 'port') { $targetPort = [int]$Value }
      for ($i=0; $i -lt 2; $i++) {
        $protocol = @('TCP','UDP')[$i]
        New-NetFirewallRule -PolicyStore PersistentStore -Name $created[$i] -DisplayName $created[$i] -Direction Inbound -Action Allow -Enabled True -Profile Any -Protocol $protocol -LocalPort $targetPort -RemoteAddress $addresses -Service TermService | Out-Null
      }
    }
    if ($Kind -eq 'whitelist') {
      foreach ($saved in $rules) { Set-NetFirewallRule -PolicyStore PersistentStore -Name $saved.Name -RemoteAddress $addresses | Out-Null }
    }
    if ($Kind -eq 'nla') { Set-ItemProperty -LiteralPath $rdpKey -Name UserAuthentication -Type DWord -Value ([int]$Value) }
    if ($Kind -eq 'firewall') { Set-NetFirewallProfile -Profile Domain,Private,Public -Enabled True }
    if ($Kind -eq 'port') { Set-ItemProperty -LiteralPath $rdpKey -Name PortNumber -Type DWord -Value ([int]$Value) }
    if ($Kind -ne 'port') { $state.Applied = [datetime]::UtcNow }
    $state.Phase = 'awaiting-confirmation'; Save-State $state
    if ($Kind -eq 'port') {
      $trigger = New-ScheduledTaskTrigger -Once -At (Get-Date).AddSeconds(10)
      $settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 2)
      Register-ScheduledTask -TaskName $activateName -Action (Task-Action 'Activate' $id) -Trigger $trigger -Settings $settings -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
    }
    Write-Audit ($id + ' applied; confirmation required')
  } catch {
    $failure = $_.Exception.Message
    try { Restore-State $state } catch { throw ($failure + '; recovery pending: ' + $_.Exception.Message) }
    throw ($failure + '; previous configuration restored')
  }
} finally {
  if ($locked) { $mutex.ReleaseMutex() }
  $mutex.Dispose()
}
