#!/bin/sh
set -eu

plugin=${1:?plugin directory required}
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

plugin=${plugin%/}
source="$plugin/browser.ts"
output="$plugin/assets/plugin.js"

if [ ! -f "$source" ]; then
  exit 0
fi

tsc="$root/node_modules/.bin/tsc"
if [ ! -x "$tsc" ]; then
  echo "TypeScript is not installed; run npm ci" >&2
  exit 1
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/kumbuka-plugin-browser.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

"$tsc" \
  "$source" \
  --target ES2022 \
  --lib ES2022,DOM,DOM.Iterable \
  --strict \
  --outDir "$tmp"

generated="$tmp/browser.js"
if [ ! -f "$generated" ]; then
  echo "TypeScript did not produce $generated" >&2
  exit 1
fi

mkdir -p "$plugin/assets"
mv "$generated" "$output"
