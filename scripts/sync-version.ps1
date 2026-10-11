# Sync version metadata from version.json (single source of truth).
$ErrorActionPreference = 'Stop'
$Root = Split-Path $PSScriptRoot -Parent
$cfgPath = Join-Path $Root 'version.json'
if (-not (Test-Path $cfgPath)) {
  throw "Missing version.json at repo root"
}

$cfg = Get-Content $cfgPath -Raw -Encoding UTF8 | ConvertFrom-Json
$name = [string]$cfg.name
$ver = [string]$cfg.version
$display = "v$ver"

function Set-FileUtf8([string]$path, [string]$content) {
  [System.IO.File]::WriteAllText($path, $content, [System.Text.UTF8Encoding]::new($false))
}

# internal/ui/dto.go
$dtoPath = Join-Path $Root 'internal\ui\dto.go'
$dto = Get-Content $dtoPath -Raw -Encoding UTF8
$dto = [regex]::Replace($dto, 'AppName\s+=\s+"[^"]*"', "AppName    = `"$name`"")
$dto = [regex]::Replace($dto, 'AppVersion\s+=\s+"[^"]*"', "AppVersion = `"$display`"")
$dto = [regex]::Replace($dto, 'Version\s+string\s+`json:"version"`\s+// v[\d.]+', "Version string ``json:`"version`"`` // $display")
Set-FileUtf8 $dtoPath $dto

# frontend/src/constants.js
$constPath = Join-Path $Root 'frontend\src\constants.js'
$const = Get-Content $constPath -Raw -Encoding UTF8
$const = [regex]::Replace($const, "export const APP_NAME = '[^']*'", "export const APP_NAME = '$name'")
$const = [regex]::Replace($const, "export const APP_VERSION = '[^']*'", "export const APP_VERSION = '$display'")
Set-FileUtf8 $constPath $const

# wails.json
$wailsPath = Join-Path $Root 'wails.json'
$wails = Get-Content $wailsPath -Raw -Encoding UTF8
$wails = [regex]::Replace($wails, '"productName":\s*"[^"]*"', "`"productName`": `"$name`"")
$wails = [regex]::Replace($wails, '"productVersion":\s*"[^"]*"', "`"productVersion`": `"$ver`"")
Set-FileUtf8 $wailsPath $wails

# frontend/package.json
$pkgPath = Join-Path $Root 'frontend\package.json'
$pkg = Get-Content $pkgPath -Raw -Encoding UTF8
$pkg = [regex]::Replace($pkg, '"version":\s*"[^"]*"', "`"version`": `"$ver`"")
Set-FileUtf8 $pkgPath $pkg

# Keep all language editions aligned (header line only).
foreach ($readmeName in @('README.md', 'README.zh-Hans.md', 'README.zh-Hant.md')) {
  $readmePath = Join-Path $Root $readmeName
  if (-not (Test-Path -LiteralPath $readmePath)) { continue }
  $readme = Get-Content -LiteralPath $readmePath -Raw -Encoding UTF8
  $readme = [regex]::Replace($readme, '(?m)^(Version: |版本：)v[\d.]+', ('${1}' + $display))
  Set-FileUtf8 $readmePath $readme
}

Write-Host "Synced version $display from version.json" -ForegroundColor DarkGray

# Keep both lockfile metadata entries aligned without touching dependency versions.
$lockPath = Join-Path $Root 'frontend\package-lock.json'
node -e 'const fs=require("fs"); const [version,path]=process.argv.slice(1); const lock=JSON.parse(fs.readFileSync(path,"utf8")); if(lock.version!==version || lock.packages[""].version!==version){ lock.version=version; lock.packages[""].version=version; fs.writeFileSync(path,JSON.stringify(lock,null,2)+"\n"); }' $ver $lockPath
if ($LASTEXITCODE -ne 0) { throw 'Failed to synchronize lockfile metadata' }
gofmt -w $dtoPath
if ($LASTEXITCODE -ne 0) { throw 'Failed to format version DTO' }
