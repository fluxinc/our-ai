#!/bin/sh
# Stable documentation-site entrypoint; the canonical installer stays at the
# repository root so release and local verification use one implementation.
set -eu

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
latest="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/fluxinc/my-cli/releases/latest)"
tag="${latest##*/}"
[ -n "$tag" ] || { echo "Error: could not determine the latest My AI release" >&2; exit 1; }
curl -fsSL "https://raw.githubusercontent.com/fluxinc/my-cli/${tag}/install.sh" -o "$tmp"
sh "$tmp" "$@"
