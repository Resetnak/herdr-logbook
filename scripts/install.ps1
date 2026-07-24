param()
$ErrorActionPreference = 'Stop'

$PluginRoot = if ($env:HERDR_PLUGIN_ROOT) { $env:HERDR_PLUGIN_ROOT } else { Split-Path -Parent $PSScriptRoot }
$ManifestPath = if ($PluginRoot) { Join-Path $PluginRoot 'herdr-plugin.toml' } else { $null }
if ($ManifestPath -and (Test-Path -LiteralPath $ManifestPath)) {
    $Manifest = Get-Content -LiteralPath $ManifestPath -Raw
    $Bin = Join-Path $PluginRoot 'bin'
} else {
    # Standalone install (irm | iex): no plugin checkout around the script.
    $Manifest = (Invoke-WebRequest -UseBasicParsing -Uri 'https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/herdr-plugin.toml').Content
    $Bin = Join-Path $env:USERPROFILE '.local\bin'
}
if ($Manifest -notmatch '(?m)^version\s*=\s*"([^"]+)"') { throw 'Plugin version is missing from herdr-plugin.toml' }
$Version = $Matches[1]
$Architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
switch ($Architecture) {
    'x64' { $Arch = 'amd64' }
    'arm64' { $Arch = 'arm64' }
    default { throw "Unsupported architecture: $Architecture" }
}
$Asset = "herdr-logbook_${Version}_windows_${Arch}.zip"
$BaseUrl = "https://github.com/Resetnak/herdr-logbook/releases/download/v${Version}"
$Temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("herdr-logbook-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $Temporary | Out-Null
try {
    $Archive = Join-Path $Temporary $Asset
    $Checksums = Join-Path $Temporary 'checksums.txt'
    Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/$Asset" -OutFile $Archive
    Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/checksums.txt" -OutFile $Checksums
    $Line = Get-Content -LiteralPath $Checksums | Where-Object { $_ -match "\s$([regex]::Escape($Asset))$" } | Select-Object -First 1
    if (-not $Line) { throw "Checksum for $Asset is missing" }
    $Expected = ($Line -split '\s+')[0].ToLowerInvariant()
    $Actual = (Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) { throw "Checksum verification failed for $Asset" }
    New-Item -ItemType Directory -Force -Path $Bin | Out-Null
    Expand-Archive -LiteralPath $Archive -DestinationPath $Bin -Force
    Write-Output "installed herdr-logbook $Version to $Bin"
} finally {
    Remove-Item -LiteralPath $Temporary -Recurse -Force -ErrorAction SilentlyContinue
}
