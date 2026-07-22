#!/usr/bin/env bash
set -eu

herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_root="${HERDR_PLUGIN_ROOT:?HERDR_PLUGIN_ROOT is required}"
logbook_bin="$plugin_root/bin/herdr-logbook"
cwd="$("$logbook_bin" resolve-cwd)"
entrypoint="${1:-hub}"
placement=overlay
if [ "$entrypoint" = "hub" ]; then
  placement=split
fi

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
