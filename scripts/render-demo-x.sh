#!/usr/bin/env bash
# render-demo-x.sh — regenerates assets/herdr-logbook-x.gif and .mp4 from
# assets/demo-x.tape. Run from anywhere:  ./scripts/render-demo-x.sh
#
# Prerequisites: vhs (brew install vhs), ffmpeg, go, git, bash.
# Recording against a stale bin/ silently ships an old UI, so this builds first.
set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

for tool in vhs ffmpeg go git; do
	command -v "$tool" >/dev/null || { echo "render-demo-x: $tool is not installed" >&2; exit 1; }
done

go build -o bin/herdr-logbook ./cmd/herdr-logbook
vhs assets/demo-x.tape

for out in assets/herdr-logbook-x.gif assets/herdr-logbook-x.mp4; do
	printf '%s  %s  %s\n' \
		"$out" \
		"$(ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0:s=x "$out")" \
		"$(du -h "$out" | cut -f1)"
done
