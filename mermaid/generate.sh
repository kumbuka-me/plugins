#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
bundle="$root/assets/mermaid.min.js"

if [ -f "$bundle" ] && head -n 1 "$bundle" | grep -Fqx '// Kumbuka Mermaid bundle: 13.0.0-alpha.3'; then
  exit 0
fi

"$root/update-assets.sh"
