$ErrorActionPreference='Stop'
Set-StrictMode -Version 2
$start=(Get-Date).AddDays(-$days)
$stamp=$start.ToUniversalTime().ToString('o')
$sources=@(
 @{Name='Microsoft-Windows-TerminalServices-LocalSessionManager/Operational';Query='(EventID=21 or EventID=22 or EventID=23 or EventID=24 or EventID=25 or EventID=39 or EventID=40)'},
 @{Name='Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational';Query='EventID=1149'},
 @{Name='Security';Query='(EventID=4624 or EventID=4625 or EventID=4634 or EventID=4778 or EventID=4779 or EventID=1102)'}
)
$items=@(); $warnings=@(); $truncated=$false; $available=0; $partial=$false
foreach ($source in $sources) {
 try {
  $log=Get-WinEvent -ListLog $source.Name
  if (!$log.IsEnabled) {$warnings+=('Log disabled: '+$source.Name);continue}
  $security=''
  if ($source.Name -eq 'Security') {
   $security=" and ((System[EventID=4624 or EventID=4634] and EventData[Data[@Name='LogonType']='10']) or (System[EventID=4625] and EventData[Data[@Name='LogonType']='10' or Data[@Name='LogonType']='3']) or System[EventID=4778 or EventID=4779 or EventID=1102])"
  }
  $query="*[System[TimeCreated[@SystemTime >= '"+$stamp+"'] and ("+$source.Query+")]"+$security+"]"
  $errors=@()
  $events=@(Get-WinEvent -LogName $source.Name -FilterXPath $query -MaxEvents ($limit+1) -ErrorAction SilentlyContinue -ErrorVariable errors)
  foreach ($item in $errors) {if ($item.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') {throw $item}}
  $available++
  if ($events.Count -gt $limit) {$truncated=$true}
  foreach ($event in @($events | Select-Object -First $limit)) {
   try {
    $xmlText=$event.ToXml(); $xml=[xml]$xmlText; $data=@{}; $fields=@(); $limited=$false
    foreach ($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="EventData"]/*[local-name()="Data"]'))) {
     $name=$node.GetAttribute('Name'); if (!$name) {$name='param'+($fields.Count+1)}
     $data[$name]=[string]$node.InnerText
    }
    foreach ($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="UserData"]/*/*'))) {$data[$node.LocalName]=[string]$node.InnerText}
    # Defense in depth for providers/mocks that do not honor the XPath.
    if ($source.Name -eq 'Security' -and $event.Id -in @(4624,4634) -and $data['LogonType'] -ne '10') {continue}
    if ($source.Name -eq 'Security' -and $event.Id -eq 4625 -and $data['LogonType'] -notin @('3','10')) {continue}
    foreach ($name in @($data.Keys | Sort-Object)) {
     if ($fields.Count -ge 64) {$limited=$true;break}
     $value=[string]$data[$name]; if ($value.Length -gt 2048) {$value=$value.Substring(0,2048);$limited=$true}
     $fields+=[pscustomobject]@{name=[string]$name;value=$value}
    }
    $message='';try {$message=[string]$event.Message} catch {}
    if ([string]::IsNullOrWhiteSpace($message)) {$warnings+=('Message resource unavailable: '+$source.Name)}
    if ($message.Length -gt 8192) {$message=$message.Substring(0,8192);$limited=$true}
    if ($xmlText.Length -gt 32768) {$xmlText=$xmlText.Substring(0,32768);$limited=$true}
    $user=[string](@($data['TargetUserName'],$data['AccountName'],$data['User'],$data['Param1']) | Where-Object {$_} | Select-Object -First 1)
    $domain=[string](@($data['TargetDomainName'],$data['AccountDomain'],$data['Param2']) | Where-Object {$_ -and $_ -ne '-'} | Select-Object -First 1)
    if ($domain -and $user -and !$user.Contains('\')) {$user=$domain+'\'+$user}
    $address=[string](@($data['IpAddress'],$data['ClientAddress'],$data['Address'],$data['Param3']) | Where-Object {$_} | Select-Object -First 1)
    $session=[string](@($data['SessionID'],$data['SessionName']) | Where-Object {$_} | Select-Object -First 1)
    $items+=[pscustomobject]@{time=$event.TimeCreated.ToUniversalTime().ToString('o');id=[int]$event.Id;recordId=[string]$event.RecordId;provider=[string]$event.ProviderName;channel=$source.Name;user=$user;address=$address;session=$session;logonType=[string]$data['LogonType'];status=[string]$data['Status'];subStatus=[string]$data['SubStatus'];detail=$message;xml=$xmlText;data=@($fields);limited=$limited}
   } catch {$partial=$true;$warnings+=('Unreadable event: '+$source.Name+'/'+$event.RecordId+': '+$_.Exception.Message)}
  }
 } catch {$warnings+=($source.Name+': '+$_.Exception.Message)}
}
if (!$available) {throw ('No event channels could be read: '+($warnings -join '; '))}
if ($items.Count -gt $limit) {$truncated=$true}
[pscustomobject]@{events=@($items | Sort-Object time -Descending | Select-Object -First $limit);warnings=@($warnings | Select-Object -Unique);truncated=$truncated;partial=($partial -or $available -lt $sources.Count);collectedAt=[datetime]::UtcNow.ToString('o');startTime=$start.ToUniversalTime().ToString('o')} | ConvertTo-Json -Depth 8 -Compress
