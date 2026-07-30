$ErrorActionPreference = 'Stop'

try {
    $RepoRoot = Split-Path -Parent $PSScriptRoot
    $TestRoot = Join-Path ([IO.Path]::GetTempPath()) ("herdr-logbook-dev-link-" + [guid]::NewGuid())
    $Zlutoucky = "repo " + [char]0x017e + "lu" + [char]0x0165 + "ou" + [char]0x010d + "k" + [char]0x00fd
    $Fixture = Join-Path $TestRoot $Zlutoucky
    $FakeBin = Join-Path $TestRoot 'fake-bin'
    $GoArgs = Join-Path $TestRoot 'go.args'
    $HerdrArgs = Join-Path $TestRoot 'herdr.args'
    $PowerShell = (Get-Process -Id $PID).Path

    New-Item -ItemType Directory -Force (Join-Path $Fixture 'scripts'), $FakeBin | Out-Null
    Copy-Item (Join-Path $RepoRoot 'scripts\dev-link.ps1') (Join-Path $Fixture 'scripts\dev-link.ps1')

    $ShimGo = Join-Path $FakeBin 'shim.go'
    $ShimCode = @'
package main

import (
	"os"
	"path/filepath"
	"strings"
)

func main() {
	base := strings.ToLower(filepath.Base(os.Args[0]))
	name := strings.TrimSuffix(base, ".exe")
	if name == "go" {
		argsFile := os.Getenv("DEV_GO_ARGS")
		if argsFile != "" {
			_ = os.WriteFile(argsFile, []byte(strings.Join(os.Args[1:], "\n")), 0644)
		}
		if os.Getenv("DEV_GO_FAIL") == "1" {
			os.Exit(23)
		}
		for i, arg := range os.Args {
			if arg == "-o" && i+1 < len(os.Args) {
				out := os.Args[i+1]
				_ = os.MkdirAll(filepath.Dir(out), 0755)
				_ = os.WriteFile(out, nil, 0644)
				break
			}
		}
	} else if name == "herdr" {
		argsFile := os.Getenv("DEV_HERDR_ARGS")
		if argsFile != "" {
			_ = os.WriteFile(argsFile, []byte(strings.Join(os.Args[1:], "\n")), 0644)
		}
	}
}
'@
    [System.IO.File]::WriteAllText($ShimGo, $ShimCode, [System.Text.Encoding]::UTF8)

    $FakeGoExe = Join-Path $FakeBin 'go.exe'
    $FakeHerdrExe = Join-Path $FakeBin 'herdr.exe'
    Push-Location $FakeBin
    try {
        & go build -o go.exe shim.go
    }
    finally {
        Pop-Location
    }
    Copy-Item $FakeGoExe $FakeHerdrExe

    $env:PATH = "$FakeBin;$env:PATH"
    $env:DEV_GO_ARGS = $GoArgs
    $env:DEV_HERDR_ARGS = $HerdrArgs
    $env:DEV_GO_FAIL = '0'

    $output = & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1') 2>&1
    if ($LASTEXITCODE -ne 0) {
        [Console]::WriteLine("dev-link.ps1 exited with code $LASTEXITCODE. Output:")
        $output | ForEach-Object { [Console]::WriteLine($_) }
        [Console]::Out.Flush()
        exit 1
    }

    $ExpectedBinary = Join-Path $Fixture 'bin\herdr-logbook.exe'
    if (-not (Test-Path -LiteralPath $GoArgs)) {
        [Console]::WriteLine("GoArgs file ($GoArgs) does not exist.")
        [Console]::Out.Flush()
        exit 1
    }
    if (-not (Test-Path -LiteralPath $HerdrArgs)) {
        [Console]::WriteLine("HerdrArgs file ($HerdrArgs) does not exist.")
        [Console]::Out.Flush()
        exit 1
    }
    $ActualGo = ([System.IO.File]::ReadAllLines($GoArgs, [System.Text.Encoding]::UTF8)) -join ' '
    $ActualHerdr = ([System.IO.File]::ReadAllLines($HerdrArgs, [System.Text.Encoding]::UTF8)) -join ' '
    if ($ActualGo -ne "build -o $ExpectedBinary ./cmd/herdr-logbook") {
        [Console]::WriteLine("Expected go args: 'build -o $ExpectedBinary ./cmd/herdr-logbook'")
        [Console]::WriteLine("Actual go args:   '$ActualGo'")
        [Console]::Out.Flush()
        exit 1
    }
    if ($ActualHerdr -ne "plugin link $Fixture --enabled") {
        [Console]::WriteLine("Expected herdr args: 'plugin link $Fixture --enabled'")
        [Console]::WriteLine("Actual herdr args:   '$ActualHerdr'")
        [Console]::Out.Flush()
        exit 1
    }
    if (($output -join [Environment]::NewLine) -ne "built and linked $ExpectedBinary") {
        [Console]::WriteLine("Expected output: 'built and linked $ExpectedBinary'")
        [Console]::WriteLine("Actual output:   '$output'")
        [Console]::Out.Flush()
        exit 1
    }

    Remove-Item -Force -ErrorAction SilentlyContinue $HerdrArgs
    $env:DEV_GO_FAIL = '1'
    $oldPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $null = & $PowerShell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Fixture 'scripts\dev-link.ps1') 2>&1
    $actualFailCode = $LASTEXITCODE
    $ErrorActionPreference = $oldPreference

    if ($actualFailCode -ne 23) {
        [Console]::WriteLine("build failure exit code = $actualFailCode, want 23")
        [Console]::Out.Flush()
        exit 1
    }
    if (Test-Path -LiteralPath $HerdrArgs) {
        [Console]::WriteLine('herdr was called after build failure')
        [Console]::Out.Flush()
        exit 1
    }
    [Console]::WriteLine("DEV LINK TEST PASSED SUCCESSFULLY")
    [Console]::Out.Flush()
}
catch {
    [Console]::WriteLine("EXCEPTION IN DEV_LINK_TEST: " + $_.Exception.ToString())
    [Console]::WriteLine("SCRIPT STACKTRACE: " + $_.ScriptStackTrace)
    [Console]::Out.Flush()
    exit 1
}
finally {
    if ($TestRoot -and (Test-Path -LiteralPath $TestRoot)) {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TestRoot
    }
}
