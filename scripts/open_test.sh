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
printf '%s\n' "$@" >"$HERDR_ARGS_FILE"
EOF
chmod +x "$test_root/bin/herdr-logbook" "$test_root/herdr"

invoke() {
	local entrypoint="$1"
	local output="$test_root/$entrypoint.args"
	HERDR_ARGS_FILE="$output" \
		HERDR_BIN_PATH="$test_root/herdr" \
		HERDR_PLUGIN_ROOT="$test_root" \
		"$repo_root/scripts/open.sh" "$entrypoint"
	paste -sd ' ' "$output"
}

hub="$(invoke hub)"
capture="$(invoke capture)"

[ "$hub" = "plugin pane open --plugin herdr-logbook --entrypoint hub --placement split --focus --direction right" ] || {
	echo "unexpected hub arguments: $hub" >&2
	exit 1
}
[ "$capture" = "plugin pane open --plugin herdr-logbook --entrypoint capture --placement overlay --focus" ] || {
	echo "unexpected capture arguments: $capture" >&2
	exit 1
}
