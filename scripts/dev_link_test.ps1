$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$TestRoot = Join-Path ([IO.Path]::GetTempPath()) ("herdr-logbook-dev-link-" + [guid]::NewGuid())
$Fixture = Join-Path $TestRoot 'repo žluťoučký'
$FakeBin = Join-Path $TestRoot 'fake-bin'
$GoArgs = Join-Path $TestRoot 'go.args'
$HerdrArgs = Join-Path $TestRoot 'herdr.args'
$PowerShell = (Get-Process -Id $PID).Path

try {
    New-Item -ItemType Directory -Force (Join-Path $Fixture 'scripts'), $FakeBin | Out-Null
    Copy-Item (Join-Path $RepoRoot 'scripts\dev-link.ps1') (Join-Path $Fixture 'scripts\dev-link.ps1')

    @'
$CommandArgs = @($args)
$CommandArgs | Set-Content -LiteralPath $env:DEV_GO_ARGS -Encoding UTF8
if ($env:DEV_GO_FAIL -eq '1') { exit 23 }
$outputIndex = [Array]::IndexOf($CommandArgs, '-o')
if ($outputIndex -lt 0) { exit 2 }
$output = $CommandArgs[$outputIndex + 1]
New-Item -ItemType Directory -Force (Split-Path -Parent $output) | Out-Null
New-Item -ItemType File -Force $output | Out-Null
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'go.ps1') -Encoding UTF8

    @'
$CommandArgs = @($args)
$CommandArgs | Set-Content -LiteralPath $env:DEV_HERDR_ARGS -Encoding UTF8
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'herdr.ps1') -Encoding UTF8

    $env:PATH = "$FakeBin;$env:PATH"
    $env:PATHEXT = ".PS1;$env:PATHEXT"
    $env:DEV_GO_ARGS = $GoArgs
    $env:DEV_HERDR_ARGS = $HerdrArgs
    $env:DEV_GO_FAIL = '0'

    $output = & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1')
    if ($LASTEXITCODE -ne 0) { throw "dev-link.ps1 exited $LASTEXITCODE" }

    $ExpectedBinary = Join-Path $Fixture 'bin\herdr-logbook.exe'
    $ActualGo = (Get-Content -LiteralPath $GoArgs -Encoding UTF8) -join ' '
    $ActualHerdr = (Get-Content -LiteralPath $HerdrArgs -Encoding UTF8) -join ' '
    if ($ActualGo -ne "build -o $ExpectedBinary ./cmd/herdr-logbook") {
        throw "unexpected go arguments: $ActualGo"
    }
    if ($ActualHerdr -ne "plugin link $Fixture --enabled") {
        throw "unexpected herdr arguments: $ActualHerdr"
    }
    if (($output -join [Environment]::NewLine) -ne "built and linked $ExpectedBinary") {
        throw "unexpected output: $output"
    }

    Remove-Item -Force -ErrorAction SilentlyContinue $HerdrArgs
    $env:DEV_GO_FAIL = '1'
    & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1') *> $null
    if ($LASTEXITCODE -ne 23) {
        throw "build failure exit code = $LASTEXITCODE, want 23"
    }
    if (Test-Path -LiteralPath $HerdrArgs) {
        throw 'herdr was called after build failure'
    }
}
finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TestRoot
}
