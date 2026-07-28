param([string]$Entrypoint = 'hub-windows')
$ErrorActionPreference = 'Stop'

$HerdrBin = if ($env:HERDR_BIN_PATH) { $env:HERDR_BIN_PATH } else { 'herdr' }
$PluginRoot = $env:HERDR_PLUGIN_ROOT
if (-not $PluginRoot) { throw 'HERDR_PLUGIN_ROOT is required' }
if ($PluginRoot.StartsWith('\\?\')) { $PluginRoot = $PluginRoot.Substring(4) }

$LogbookBin = Join-Path $PluginRoot 'bin\herdr-logbook.exe'

# Pane label reported by herdr; keep in sync with the [[panes]] titles in herdr-plugin.toml.
$Label = if ($Entrypoint -like 'hub*') { 'Herdr Logbook' } else { 'Capture Logbook' }

# Toggle: focus the pane this entrypoint already owns, and close it once it has focus.
$OpenPanes = @()
try {
    $PaneList = (& $HerdrBin pane list | ConvertFrom-Json)
    $OpenPanes = @($PaneList.result.panes | Where-Object { $_.label -eq $Label })
} catch {
    $OpenPanes = @()
}
if ($OpenPanes.Count -gt 0) {
    if ($OpenPanes | Where-Object { $_.focused }) {
        foreach ($Pane in $OpenPanes) {
            & $HerdrBin plugin pane close $Pane.pane_id | Out-Null
        }
    } else {
        & $HerdrBin plugin pane focus $OpenPanes[0].pane_id | Out-Null
    }
    exit 0
}

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
