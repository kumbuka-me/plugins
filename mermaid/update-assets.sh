#!/bin/sh
set -eu

# Mermaid v13 is ESM-only. Build an isolated browser bundle during plugin packaging.
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
cp "$SCRIPT_DIR/entry.mjs" "$workdir/entry.mjs"
mkdir -p "$MERMAID_OUTPUT_DIR"
"$workdir/node_modules/.bin/esbuild" "$workdir/entry.mjs" \
  --bundle \
  --platform=browser \
  --format=iife \
  --target=es2022 \
  --minify \
  --legal-comments=eof \
  --outfile="$MERMAID_OUTPUT_DIR/mermaid.min.js.tmp"

# Mark the output to distinguish it from a stale v12 bundle.
{
  printf '// Kumbuka Mermaid bundle: %s\n' "$MERMAID_VERSION"
  cat "$MERMAID_OUTPUT_DIR/mermaid.min.js.tmp"
} > "$MERMAID_OUTPUT_DIR/mermaid.min.js"
rm "$MERMAID_OUTPUT_DIR/mermaid.min.js.tmp"
cp "$workdir/node_modules/mermaid/LICENSE" "$MERMAID_OUTPUT_DIR/LICENSE"
