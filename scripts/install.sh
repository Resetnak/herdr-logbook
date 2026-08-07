#!/usr/bin/env bash
set -euo pipefail

plugin_root="${HERDR_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
manifest="$plugin_root/herdr-plugin.toml"
if [ -f "$manifest" ]; then
  version="$(awk -F '"' '/^version = / { print $2; exit }' "$manifest")"
  install_dir="$plugin_root/bin"
else
  # Standalone install (curl | bash): there is no plugin checkout around the
  # script, so resolve the newest published release and drop the binary into a
  # PATH-friendly directory instead of a plugin root. The manifest on main is
  # not used here: a version bump that has landed before its tag and assets
  # exist would point at a release that is not downloadable yet.
  latest="$(curl --fail --location --silent --show-error --output /dev/null \
    --write-out '%{url_effective}' \
    https://github.com/Resetnak/herdr-logbook/releases/latest)"
  version="${latest##*/}"
  version="${version#v}"
  install_dir="${HERDR_LOGBOOK_BIN_DIR:-$HOME/.local/bin}"
fi
if [ -z "$version" ]; then
  echo "could not determine the herdr-logbook version" >&2
  exit 1
fi
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

asset="herdr-logbook_${version}_${os}_${arch}.tar.gz"
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
mkdir -p "$install_dir"
tar -xzf "$temporary/$asset" -C "$install_dir" herdr-logbook
chmod 0755 "$install_dir/herdr-logbook"
echo "installed herdr-logbook ${version} to $install_dir/herdr-logbook"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "note: $install_dir is not on your PATH" ;;
esac
