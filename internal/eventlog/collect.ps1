$ErrorActionPreference='Stop'
Set-StrictMode -Version 2
$start=(Get-Date).AddDays(-$days)
$pairs=@(
 @{Provider='Microsoft-Windows-Kernel-General';IDs=@(1,12,13)},
 @{Provider='Microsoft-Windows-Kernel-Power';IDs=@(41)},
 @{Provider='EventLog';IDs=@(6005,6006,6008,6009)},
 @{Provider='User32';IDs=@(1074)},
 @{Provider='Microsoft-Windows-WER-SystemErrorReporting';IDs=@(1001)},
 @{Provider='BugCheck';IDs=@(1001)},
 @{Provider='Microsoft-Windows-WHEA-Logger';IDs=@(17,18,19,20,46,47)},
 @{Provider='volmgr';IDs=@(46,161)},
 @{Provider='disk';IDs=@(7,11,51,153)},
 @{Provider='Ntfs';IDs=@(55,98)},
 @{Provider='storahci';IDs=@(129)},
 @{Provider='stornvme';IDs=@(129)},
 @{Provider='storport';IDs=@(129)},
 @{Provider='Service Control Manager';IDs=@(7031,7034)},
 @{Provider='Microsoft-Windows-WindowsUpdateClient';IDs=@(19)},
 @{Provider='Microsoft-Windows-Eventlog';IDs=@(104)}
)
$clauses=@($pairs | ForEach-Object {
 $idClause=(@($_.IDs | ForEach-Object {'EventID='+$_}) -join ' or ')
 "(Provider[@Name='"+$_.Provider+"'] and ("+$idClause+"))"
})
$stamp=$start.ToUniversalTime().ToString('o')
$xpath="*[System[TimeCreated[@SystemTime >= '"+$stamp+"'] and ("+($clauses -join ' or ')+")]]"
$log=Get-WinEvent -ListLog System
if (!$log.IsEnabled) {throw 'System event log is disabled.'}
$errors=@(); $warnings=@()
$records=@(Get-WinEvent -LogName System -FilterXPath $xpath -MaxEvents ($limit+1) -ErrorAction SilentlyContinue -ErrorVariable errors)
foreach($entry in $errors) {if($entry.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') {throw $entry}}
$truncated=$records.Count -gt $limit
$events=@(foreach($record in @($records | Select-Object -First $limit)) {
 $xmlText=$record.ToXml(); $xml=[xml]$xmlText; $data=@(); $index=0
 foreach($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="EventData"]/*[local-name()="Data"]'))) {
   $name=$node.GetAttribute('Name'); if(!$name){$name='param'+(++$index)}
   $data+=[pscustomobject]@{name=[string]$name;value=[string]$node.InnerText}
 }
 foreach($node in @($xml.SelectNodes('/*[local-name()="Event"]/*[local-name()="UserData"]/*/*'))) {$data+=[pscustomobject]@{name=[string]$node.LocalName;value=[string]$node.InnerText}}
 $message=''; try {$message=[string]$record.Message} catch {}
 if([string]::IsNullOrWhiteSpace($message)){$warnings+='Message resource unavailable or empty: '+$record.RecordId}
 $limited=$message.Length -gt 8192 -or $xmlText.Length -gt 32768
 if($message.Length -gt 8192){$message=$message.Substring(0,8192)}
 if($xmlText.Length -gt 32768){$xmlText=$xmlText.Substring(0,32768)}
 [pscustomobject]@{recordId=[string]$record.RecordId;time=$record.TimeCreated.ToUniversalTime().ToString('o');provider=[string]$record.ProviderName;id=[int]$record.Id;level=[int]$record.Level;computer=[string]$record.MachineName;message=$message;xml=$xmlText;data=@($data);limited=$limited}
})
[pscustomobject]@{events=@($events);warnings=@($warnings);truncated=$truncated;collectedAt=[datetime]::UtcNow.ToString('o');startTime=$start.ToUniversalTime().ToString('o')} | ConvertTo-Json -Depth 8 -Compress
