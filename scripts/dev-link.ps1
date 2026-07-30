$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$BinaryPath = Join-Path $RepoRoot 'bin\herdr-logbook.exe'

New-Item -ItemType Directory -Force (Split-Path -Parent $BinaryPath) | Out-Null
Push-Location $RepoRoot
try {
    & go build -o $BinaryPath ./cmd/herdr-logbook
    $BuildExit = $LASTEXITCODE
}
finally {
    Pop-Location
}
if ($BuildExit -ne 0) { exit $BuildExit }

& herdr plugin link $RepoRoot --enabled
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Output "built and linked $BinaryPath"
