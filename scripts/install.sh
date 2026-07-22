#!/usr/bin/env bash
set -euo pipefail

plugin_root="${HERDR_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
version="$(awk -F '"' '/^version = / { print $2; exit }' "$plugin_root/herdr-plugin.toml")"
case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

asset="herdr-memory_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/Resetnak/herdr-logbook/releases/download/v${version}"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT
curl --fail --location --silent --show-error --output "$temporary/$asset" "$base_url/$asset"
curl --fail --location --silent --show-error --output "$temporary/checksums.txt" "$base_url/checksums.txt"
expected="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$temporary/checksums.txt")"
if [ -z "$expected" ]; then
  echo "checksum for $asset is missing" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$temporary/$asset" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "$temporary/$asset" | awk '{print $1}')"
fi
if [ "$actual" != "$expected" ]; then
  echo "checksum verification failed for $asset" >&2
  exit 1
fi
mkdir -p "$plugin_root/bin"
tar -xzf "$temporary/$asset" -C "$plugin_root/bin" herdr-memory
chmod 0755 "$plugin_root/bin/herdr-memory"
