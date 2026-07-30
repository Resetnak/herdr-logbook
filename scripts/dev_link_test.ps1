$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$TestRoot = Join-Path ([IO.Path]::GetTempPath()) ("herdr-logbook-dev-link-" + [guid]::NewGuid())
$Zlutoucky = "repo " + [char]0x017e + "lu" + [char]0x0165 + "ou" + [char]0x010d + "k" + [char]0x00fd
$Fixture = Join-Path $TestRoot $Zlutoucky
$FakeBin = Join-Path $TestRoot 'fake-bin'
$GoArgs = Join-Path $TestRoot 'go.args'
$HerdrArgs = Join-Path $TestRoot 'herdr.args'
$PowerShell = (Get-Process -Id $PID).Path

try {
    New-Item -ItemType Directory -Force (Join-Path $Fixture 'scripts'), $FakeBin | Out-Null
    Copy-Item (Join-Path $RepoRoot 'scripts\dev-link.ps1') (Join-Path $Fixture 'scripts\dev-link.ps1')

    @'
$CommandArgs = @($args)
[System.IO.File]::WriteAllLines($env:DEV_GO_ARGS, $CommandArgs, [System.Text.Encoding]::UTF8)
if ($env:DEV_GO_FAIL -eq '1') { exit 23 }
$outputIndex = [Array]::IndexOf($CommandArgs, '-o')
if ($outputIndex -lt 0) { exit 2 }
$output = $CommandArgs[$outputIndex + 1]
New-Item -ItemType Directory -Force (Split-Path -Parent $output) | Out-Null
New-Item -ItemType File -Force $output | Out-Null
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'go-fake.ps1') -Encoding UTF8

    @'
@echo off
chcp 65001 >nul
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0go-fake.ps1" %*
exit /b %ERRORLEVEL%
'@ | Set-Content -Encoding ASCII -LiteralPath (Join-Path $FakeBin 'go.cmd')

    @'
$CommandArgs = @($args)
[System.IO.File]::WriteAllLines($env:DEV_HERDR_ARGS, $CommandArgs, [System.Text.Encoding]::UTF8)
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'herdr-fake.ps1') -Encoding UTF8

    @'
@echo off
chcp 65001 >nul
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0herdr-fake.ps1" %*
exit /b %ERRORLEVEL%
'@ | Set-Content -Encoding ASCII -LiteralPath (Join-Path $FakeBin 'herdr.cmd')

    $env:PATH = "$FakeBin;$env:PATH"
    $env:DEV_GO_ARGS = $GoArgs
    $env:DEV_HERDR_ARGS = $HerdrArgs
    $env:DEV_GO_FAIL = '0'

    $output = & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1') 2>&1
    if ($LASTEXITCODE -ne 0) {
        [Console]::WriteLine("dev-link.ps1 exited with code $LASTEXITCODE. Output:")
        $output | ForEach-Object { [Console]::WriteLine($_) }
        throw "dev-link.ps1 exited $LASTEXITCODE"
    }

    $ExpectedBinary = Join-Path $Fixture 'bin\herdr-logbook.exe'
    if (-not (Test-Path -LiteralPath $GoArgs)) {
        [Console]::WriteLine("GoArgs file ($GoArgs) does not exist.")
        throw "missing GoArgs"
    }
    if (-not (Test-Path -LiteralPath $HerdrArgs)) {
        [Console]::WriteLine("HerdrArgs file ($HerdrArgs) does not exist.")
        throw "missing HerdrArgs"
    }
    $ActualGo = ([System.IO.File]::ReadAllLines($GoArgs, [System.Text.Encoding]::UTF8)) -join ' '
    $ActualHerdr = ([System.IO.File]::ReadAllLines($HerdrArgs, [System.Text.Encoding]::UTF8)) -join ' '
    if ($ActualGo -ne "build -o $ExpectedBinary ./cmd/herdr-logbook") {
        [Console]::WriteLine("Expected go args: 'build -o $ExpectedBinary ./cmd/herdr-logbook'")
        [Console]::WriteLine("Actual go args:   '$ActualGo'")
        throw "unexpected go arguments: $ActualGo"
    }
    if ($ActualHerdr -ne "plugin link $Fixture --enabled") {
        [Console]::WriteLine("Expected herdr args: 'plugin link $Fixture --enabled'")
        [Console]::WriteLine("Actual herdr args:   '$ActualHerdr'")
        throw "unexpected herdr arguments: $ActualHerdr"
    }
    if (($output -join [Environment]::NewLine) -ne "built and linked $ExpectedBinary") {
        [Console]::WriteLine("Expected output: 'built and linked $ExpectedBinary'")
        [Console]::WriteLine("Actual output:   '$output'")
        throw "unexpected output: $output"
    }

    Remove-Item -Force -ErrorAction SilentlyContinue $HerdrArgs
    $env:DEV_GO_FAIL = '1'
    $oldPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $null = & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1') 2>&1
    $actualFailCode = $LASTEXITCODE
    $ErrorActionPreference = $oldPreference

    if ($actualFailCode -ne 23) {
        throw "build failure exit code = $actualFailCode, want 23"
    }
    if (Test-Path -LiteralPath $HerdrArgs) {
        throw 'herdr was called after build failure'
    }
}
catch {
    [Console]::WriteLine("EXCEPTION IN DEV_LINK_TEST: " + $_.Exception.ToString())
    [Console]::WriteLine("SCRIPT STACKTRACE: " + $_.ScriptStackTrace)
    throw
}
finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TestRoot
}
