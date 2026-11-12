#!/bin/sh
set -eu

# Mermaid v13 is ESM-only. Bundle its lazy imports into one classic plugin script.
# A single script works in Kumbuka's opaque sandbox and on static hosts without CORS.
MERMAID_VERSION=13.0.0-alpha.3
ESBUILD_VERSION=0.28.2
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
MERMAID_OUTPUT_DIR=${MERMAID_OUTPUT_DIR:-"$SCRIPT_DIR/assets"}
NPM=${NPM:-npm}

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT INT TERM

# Only plugin authors/build CI need npm; the released plugin contains the bundle.
"$NPM" install \
  --prefix "$workdir" \
  --no-save \
  --no-package-lock \
  --ignore-scripts \
  --no-audit \
  --no-fund \
  "mermaid@$MERMAID_VERSION" "esbuild@$ESBUILD_VERSION"

# Resolve mermaid from the temporary node_modules, not from a site's dependencies.
cp "$SCRIPT_DIR/browser.ts" "$workdir/browser.ts"
mkdir -p "$MERMAID_OUTPUT_DIR"
"$workdir/node_modules/.bin/esbuild" "$workdir/browser.ts" \
  --bundle \
  --platform=browser \
  --format=iife \
  --target=es2022 \
  --minify \
  --legal-comments=eof \
  --outfile="$MERMAID_OUTPUT_DIR/plugin.js.tmp"

# Bundle the same Mermaid engine for the Go/WASM plugin's embedded JS runtime.
# It has no DOM dependency, and must not import Node or fetch at render time.
cp "$SCRIPT_DIR/headless.ts" "$workdir/headless.ts"
"$workdir/node_modules/.bin/esbuild" "$workdir/headless.ts" \
  --bundle \
  --platform=neutral \
  --format=iife \
  --target=es2022 \
  --minify \
  --legal-comments=eof \
  --outfile="$MERMAID_OUTPUT_DIR/headless.js.tmp"
{
  printf '// Kumbuka Mermaid headless bundle: %s\n' "$MERMAID_VERSION"
  cat "$MERMAID_OUTPUT_DIR/headless.js.tmp"
} > "$MERMAID_OUTPUT_DIR/headless.js.new"
mv "$MERMAID_OUTPUT_DIR/headless.js.new" "$MERMAID_OUTPUT_DIR/headless.js"
rm "$MERMAID_OUTPUT_DIR/headless.js.tmp"

# Mark the output so an old independent renderer cannot be reused accidentally.
{
  printf '// Kumbuka Mermaid browser bundle: %s\n' "$MERMAID_VERSION"
  cat "$MERMAID_OUTPUT_DIR/plugin.js.tmp"
} > "$MERMAID_OUTPUT_DIR/plugin.js.new"
mv "$MERMAID_OUTPUT_DIR/plugin.js.new" "$MERMAID_OUTPUT_DIR/plugin.js"
rm "$MERMAID_OUTPUT_DIR/plugin.js.tmp"
cp "$workdir/node_modules/mermaid/LICENSE" "$MERMAID_OUTPUT_DIR/LICENSE"
