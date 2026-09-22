#!/bin/sh
set -eu

plugin=${1:?plugin directory required}
source="$plugin/browser.ts"

if [ ! -f "$source" ]; then
  exit 0
fi

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
tsc="$root/node_modules/.bin/tsc"

if [ ! -x "$tsc" ]; then
  echo "TypeScript is not installed; run npm ci" >&2
  exit 1
fi

mkdir -p "$plugin/assets"
"$tsc" \
  "$source" \
  --target ES2022 \
  --lib ES2022,DOM,DOM.Iterable \
  --strict \
  --outDir "$plugin/assets"

if [ ! -f "$plugin/assets/browser.js" ]; then
  echo "TypeScript did not produce $plugin/assets/browser.js" >&2
  exit 1
fi

mv "$plugin/assets/browser.js" "$plugin/assets/plugin.js"
