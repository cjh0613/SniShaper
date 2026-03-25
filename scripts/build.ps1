param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$WailsArgs
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

Push-Location $repoRoot
try {
    & wails build @WailsArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    & powershell -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "sync-runtime-config.ps1")
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}
