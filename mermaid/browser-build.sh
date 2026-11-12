#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
bundle="$root/assets/plugin.js"
version='13.0.0-alpha.3'

# Drop artifacts produced by the superseded two-script and static-export builds.
rm -f "$root/assets/mermaid.min.js" "$root/assets/static-exports.json"

# Avoid an npm install on every browser build unless sources or the pinned alpha change.
if [ -s "$bundle" ] && \
   head -n 1 "$bundle" | grep -Fqx "// Kumbuka Mermaid browser bundle: $version" && \
   [ "$bundle" -nt "$root/browser.ts" ] && \
   [ "$bundle" -nt "$root/update-assets.sh" ]; then
  exit 0
fi

"$root/update-assets.sh"
