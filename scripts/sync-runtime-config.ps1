$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$sourceConfig = Join-Path $repoRoot "rules\config.json"
$targetDir = Join-Path $repoRoot "build\bin\rules"
$targetConfig = Join-Path $targetDir "config.json"

if (!(Test-Path $sourceConfig)) {
    throw "Runtime config not found: $sourceConfig"
}

if (!(Test-Path $targetDir)) {
    New-Item -ItemType Directory -Path $targetDir | Out-Null
}

Copy-Item -Path $sourceConfig -Destination $targetConfig -Force
Write-Host "Synced runtime config to $targetConfig"
