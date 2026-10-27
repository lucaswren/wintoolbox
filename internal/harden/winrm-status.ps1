$ErrorActionPreference = 'Stop'
$queryErrors = @()
# Query the effective store once; do not launch another process per port.
$rules = @(Get-NetFirewallRule -PolicyStore ActiveStore -Name 'WinToolbox-Block-WinRM-*' -ErrorAction SilentlyContinue -ErrorVariable queryErrors)
foreach ($item in $queryErrors) {
  if ($item.FullyQualifiedErrorId -notlike 'CmdletizationQuery_NotFound_Name*') { throw $item }
}
$result = @{port5985=$false;port5986=$false}
foreach ($port in @(5985,5986)) {
  foreach ($rule in @($rules | Where-Object { $_.Name -eq ('WinToolbox-Block-WinRM-' + $port) })) {
    if ([string]$rule.Enabled -ne 'True' -or [string]$rule.Direction -ne 'Inbound' -or [string]$rule.Action -ne 'Block') { continue }
    $filter = $rule | Get-NetFirewallPortFilter -ErrorAction Stop
    if ([string]$filter.Protocol -notin @('TCP','6')) { continue }
    if (@($filter.LocalPort) -contains [string]$port) { $result['port' + $port] = $true }
  }
}
$result | ConvertTo-Json -Compress
