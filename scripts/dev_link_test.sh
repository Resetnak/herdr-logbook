#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT

[ -f "$repo_root/scripts/dev-link.ps1" ] || {
	echo "missing scripts/dev-link.ps1" >&2
	exit 1
}
[ -f "$repo_root/scripts/dev_link_test.ps1" ] || {
	echo "missing scripts/dev_link_test.ps1" >&2
	exit 1
}

fixture="$test_root/repo žluťoučký"
fake_bin="$test_root/fake-bin"
mkdir -p "$fixture/scripts" "$fake_bin"
cp "$repo_root/scripts/dev-link.sh" "$fixture/scripts/dev-link.sh"

cat >"$fake_bin/go" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$@" >"$DEV_GO_ARGS"
if [ "${DEV_GO_FAIL:-0}" = "1" ]; then
	exit 23
fi
while [ "$#" -gt 0 ]; do
	if [ "$1" = "-o" ]; then
		shift
		mkdir -p "$(dirname "$1")"
		: >"$1"
		break
	fi
	shift
done
EOF

cat >"$fake_bin/herdr" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$@" >"$DEV_HERDR_ARGS"
EOF

chmod +x "$fake_bin/go" "$fake_bin/herdr"

go_args="$test_root/go.args"
herdr_args="$test_root/herdr.args"
output="$(
	PATH="$fake_bin:$PATH" \
	DEV_GO_ARGS="$go_args" \
	DEV_HERDR_ARGS="$herdr_args" \
		bash "$fixture/scripts/dev-link.sh"
)"

expected_binary="$fixture/bin/herdr-logbook"
actual_go="$(paste -sd ' ' "$go_args")"
actual_herdr="$(paste -sd ' ' "$herdr_args")"

[ "$actual_go" = "build -o $expected_binary ./cmd/herdr-logbook" ] || {
	echo "unexpected go arguments: $actual_go" >&2
	exit 1
}
[ "$actual_herdr" = "plugin link $fixture --enabled" ] || {
	echo "unexpected herdr arguments: $actual_herdr" >&2
	exit 1
}
[ "$output" = "built and linked $expected_binary" ] || {
	echo "unexpected output: $output" >&2
	exit 1
}

rm -f "$herdr_args"
status=0
PATH="$fake_bin:$PATH" \
DEV_GO_ARGS="$go_args" \
DEV_HERDR_ARGS="$herdr_args" \
DEV_GO_FAIL=1 \
	bash "$fixture/scripts/dev-link.sh" >/dev/null 2>&1 || status=$?

[ "$status" -eq 23 ] || {
	echo "build failure exit code = $status, want 23" >&2
	exit 1
}
[ ! -e "$herdr_args" ] || {
	echo "herdr was called after build failure" >&2
	exit 1
}
