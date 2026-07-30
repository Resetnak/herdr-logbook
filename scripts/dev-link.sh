#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
binary="$repo_root/bin/herdr-logbook"

mkdir -p "$repo_root/bin"
(
	cd "$repo_root"
	go build -o "$binary" ./cmd/herdr-logbook
)
herdr plugin link "$repo_root" --enabled
printf 'built and linked %s\n' "$binary"
