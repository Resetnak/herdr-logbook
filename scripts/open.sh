#!/usr/bin/env bash
set -eu

herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_root="${HERDR_PLUGIN_ROOT:?HERDR_PLUGIN_ROOT is required}"
memory_bin="$plugin_root/bin/herdr-memory"
cwd="$("$memory_bin" resolve-cwd)"
entrypoint="${1:-hub}"

set -- plugin pane open \
  --plugin herdr-memory \
  --entrypoint "$entrypoint" \
  --placement overlay \
  --focus

if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  set -- "$@" --cwd "$cwd"
fi

exec "$herdr_bin" "$@"
