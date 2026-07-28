#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT

mkdir -p "$test_root/bin"
cat >"$test_root/bin/herdr-logbook" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = "resolve-cwd" ] || exit 2
EOF
cat >"$test_root/herdr" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "pane" ] && [ "${2:-}" = "list" ]; then
	printf '%s' "${HERDR_PANE_LIST:-}"
	exit 0
fi
printf '%s\n' "$@" >"$HERDR_ARGS_FILE"
EOF
chmod +x "$test_root/bin/herdr-logbook" "$test_root/herdr"

invoke() {
	local entrypoint="$1"
	local output="$test_root/$entrypoint.args"
	: >"$output"
	HERDR_ARGS_FILE="$output" \
		HERDR_PANE_LIST="${2:-}" \
		HERDR_BIN_PATH="$test_root/herdr" \
		HERDR_PLUGIN_ROOT="$test_root" \
		"$repo_root/scripts/open.sh" "$entrypoint"
	paste -sd ' ' "$output"
}

hub_focused='{"id":"cli:pane:list","result":{"panes":[{"cwd":"/work","focused":false,"pane_id":"wS:p1","scroll":{"offset_from_bottom":0}},{"cwd":"/work","focused":true,"label":"Herdr Logbook","pane_id":"wS:pF","revision":0,"scroll":{"offset_from_bottom":0}}],"type":"pane_list"}}'
hub_unfocused='{"id":"cli:pane:list","result":{"panes":[{"cwd":"/work","focused":true,"pane_id":"wS:p1","scroll":{"offset_from_bottom":0}},{"cwd":"/work","focused":false,"label":"Herdr Logbook","pane_id":"wS:pF","revision":0,"scroll":{"offset_from_bottom":0}}],"type":"pane_list"}}'

hub="$(invoke hub)"
capture="$(invoke capture)"
hub_close="$(invoke hub "$hub_focused")"
hub_focus="$(invoke hub "$hub_unfocused")"
capture_untouched="$(invoke capture "$hub_focused")"

[ "$hub" = "plugin pane open --plugin herdr-logbook --entrypoint hub --placement split --focus --direction right" ] || {
	echo "unexpected hub arguments: $hub" >&2
	exit 1
}
[ "$capture" = "plugin pane open --plugin herdr-logbook --entrypoint capture --placement overlay --focus" ] || {
	echo "unexpected capture arguments: $capture" >&2
	exit 1
}
[ "$hub_close" = "plugin pane close wS:pF" ] || {
	echo "a focused hub pane must be closed: $hub_close" >&2
	exit 1
}
[ "$hub_focus" = "plugin pane focus wS:pF" ] || {
	echo "an unfocused hub pane must be focused: $hub_focus" >&2
	exit 1
}
[ "$capture_untouched" = "plugin pane open --plugin herdr-logbook --entrypoint capture --placement overlay --focus" ] || {
	echo "capture must ignore an open hub pane: $capture_untouched" >&2
	exit 1
}
