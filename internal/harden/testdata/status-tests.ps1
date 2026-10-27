param([string]$Script)
$ErrorActionPreference='Stop'
$global:rules=@()
$global:queries=0
$global:deny=$false
$global:notFound=$false
function Get-NetFirewallRule {
  [CmdletBinding()]
  param($PolicyStore,$Name)
  if ($PolicyStore -ne 'ActiveStore') { throw 'Must read effective rules' }
  $global:queries++
  if ($global:deny) { throw 'Injected provider access denied' }
  if ($global:notFound) { Write-Error -Message 'No matching rule' -ErrorId 'CmdletizationQuery_NotFound_Name'; return }
  return $global:rules
}
function Get-NetFirewallPortFilter {
  param([Parameter(ValueFromPipeline=$true)]$InputObject)
  process { return [pscustomobject]@{Protocol=$InputObject.Protocol;LocalPort=$InputObject.LocalPort} }
}
function Rule($port) {
  return [pscustomobject]@{Name=('WinToolbox-Block-WinRM-'+$port); Enabled='True';Direction='Inbound';Action='Block';Protocol='TCP';LocalPort=@([string]$port)}
}
foreach ($case in @('both','missing','not-found-error','partial','udp','outbound','disabled','allow','prefix-port','multiple-ports','unrelated-name')) {
  $global:queries=0
  $global:notFound=$false
  $global:rules=@((Rule 5985),(Rule 5986))
  $want5=$true; $want6=$true
  switch ($case) {
    'missing' { $global:rules=@(); $want5=$false; $want6=$false }
    'not-found-error' { $global:notFound=$true; $want5=$false; $want6=$false }
    'partial' { $global:rules=@((Rule 5985)); $want6=$false }
    'udp' { $global:rules[0].Protocol='UDP'; $want5=$false }
    'outbound' { $global:rules[0].Direction='Outbound'; $want5=$false }
    'disabled' { $global:rules[0].Enabled='False'; $want5=$false }
    'allow' { $global:rules[0].Action='Allow'; $want5=$false }
    'prefix-port' { $global:rules[0].LocalPort=@('59850'); $want5=$false }
    'multiple-ports' { $global:rules[0].LocalPort=@('80','5985') }
    'unrelated-name' { $global:rules[0].Name='WinToolbox-Block-WinRM-59850'; $want5=$false }
  }
  $result=(& $Script | ConvertFrom-Json)
  if ($result.port5985 -ne $want5 -or $result.port5986 -ne $want6 -or $global:queries -ne 1) { throw ('Invalid status: '+$case) }
}
$global:deny=$true
$global:notFound=$false
$failed=$false
try { & $Script } catch { $failed=$_.Exception.Message -like '*access denied*' }
if (!$failed) { throw 'Provider failures must not become missing rules' }
Write-Output 'WinRM read-only status scenarios passed'
