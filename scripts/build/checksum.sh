#!/bin/sh
set -eu

file=${1:?file required}
output="$file.sha256"
name=$(basename "$file")

if command -v shasum >/dev/null 2>&1; then
  hash=$(shasum -a 256 "$file" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
  hash=$(sha256sum "$file" | awk '{print $1}')
else
  echo "neither shasum nor sha256sum is available" >&2
  exit 1
fi

printf '%s  %s\n' "$hash" "$name" >"$output"
