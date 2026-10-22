#!/usr/bin/env bash
set -eu

herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_root="${HERDR_PLUGIN_ROOT:?HERDR_PLUGIN_ROOT is required}"
logbook_bin="$plugin_root/bin/herdr-logbook"
entrypoint="${1:-hub}"
placement=overlay
# Pane label reported by herdr; keep in sync with the [[panes]] titles in herdr-plugin.toml.
label="Capture Logbook"
if [ "$entrypoint" = "hub" ]; then
  placement=split
  label="Herdr Logbook"
fi

# Toggle: focus the pane this entrypoint already owns, and close it once it has focus.
# ponytail: matched on the pane label because `herdr pane list` reports no plugin id;
# an unparseable listing just falls through to opening a pane, as before.
pane_ids() { sed -n 's/.*"pane_id":"\([^"]*\)".*/\1/p'; }
labelled_panes() {
  "$herdr_bin" pane list "$@" 2>/dev/null |
    tr '{' '\n' |
    grep -F "\"label\":\"$label\"" || true
}
# `pane list` spans every workspace; only a pane in this one is toggled.
open_panes="$(labelled_panes)"
local_panes="$open_panes"
if [ -n "${HERDR_WORKSPACE_ID:-}" ]; then
  local_panes="$(labelled_panes --workspace "$HERDR_WORKSPACE_ID")"
fi
if [ -n "$local_panes" ]; then
  if printf '%s\n' "$local_panes" | grep -qF '"focused":true'; then
    for pane_id in $(printf '%s\n' "$local_panes" | pane_ids); do
      "$herdr_bin" plugin pane close "$pane_id" >/dev/null
    done
  else
    "$herdr_bin" plugin pane focus "$(printf '%s\n' "$local_panes" | pane_ids | head -n 1)" >/dev/null
  fi
  exit 0
fi
# Open only in other workspaces: there is one Logbook at a time, so move it here.
for pane_id in $(printf '%s\n' "$open_panes" | pane_ids); do
  "$herdr_bin" plugin pane close "$pane_id" >/dev/null
done

cwd="$("$logbook_bin" resolve-cwd)"

set -- plugin pane open \
  --plugin herdr-logbook \
  --entrypoint "$entrypoint" \
  --placement "$placement" \
  --focus

if [ "$entrypoint" = "hub" ]; then
  set -- "$@" --direction right
fi
if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  set -- "$@" --cwd "$cwd"
fi

exec "$herdr_bin" "$@"
