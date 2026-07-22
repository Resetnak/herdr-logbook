param([string]$Entrypoint = 'hub-windows')
$ErrorActionPreference = 'Stop'

$HerdrBin = if ($env:HERDR_BIN_PATH) { $env:HERDR_BIN_PATH } else { 'herdr' }
$PluginRoot = $env:HERDR_PLUGIN_ROOT
if (-not $PluginRoot) { throw 'HERDR_PLUGIN_ROOT is required' }
if ($PluginRoot.StartsWith('\\?\')) { $PluginRoot = $PluginRoot.Substring(4) }

$LogbookBin = Join-Path $PluginRoot 'bin\herdr-logbook.exe'
$Cwd = (& $LogbookBin resolve-cwd | Select-Object -First 1)
$Arguments = @(
    'plugin', 'pane', 'open',
    '--plugin', 'herdr-logbook',
    '--entrypoint', $Entrypoint,
    '--placement', 'overlay',
    '--focus'
)
if ($Cwd -and (Test-Path -LiteralPath $Cwd -PathType Container)) {
    $Arguments += @('--cwd', $Cwd)
}

& $HerdrBin @Arguments
exit $LASTEXITCODE
