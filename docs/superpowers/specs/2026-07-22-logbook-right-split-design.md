# Logbook Right-Side Split Design

## Scope

The macOS/Linux `open` action opens the Logbook Hub in a focused split on the right. The `capture` action remains an overlay. Windows behavior is unchanged because the requested surface is `herdr plugin action invoke open --plugin herdr-logbook`.

## Design

Reuse Herdr's native plugin-pane placement. `scripts/open.sh` selects `split` with direction `right` for the `hub` entrypoint and keeps `overlay` for `capture`. The `hub` pane declaration in `herdr-plugin.toml` also defaults to `split`, so direct pane opening agrees with the action launcher.

Herdr's plugin-pane API does not expose a split ratio. The design therefore accepts Herdr's native split size instead of adding a brittle open-then-resize sequence solely to force an exact 30 percent width.

## Error Handling

Existing working-directory resolution, focus behavior, and command exit propagation remain unchanged. No fallback layout or custom pane management is added.

## Verification

A launcher test records the arguments passed to a stub Herdr binary and proves that `hub` requests a right split while `capture` remains an overlay. Existing shell syntax checks and the full Go test suite remain required.
