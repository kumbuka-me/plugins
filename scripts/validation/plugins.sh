#!/bin/sh
set -eu

found=0
for manifest in */plugin.yaml; do
  [ -f "$manifest" ] || continue
  found=1
  plugin=${manifest%/plugin.yaml}
  provider=$(awk '$1 == "provider:" { print $2; exit }' "$manifest")
  id=$(awk '$1 == "id:" { print $2; exit }' "$manifest")
  ./scripts/version/read.sh "$plugin" >/dev/null

  if [ "$provider" != "Kumbuka" ]; then
    echo "$manifest: provider must be Kumbuka" >&2
    exit 1
  fi
  if [ "$id" != "me.kumbuka.$plugin" ]; then
    echo "$manifest: id must be me.kumbuka.$plugin" >&2
    exit 1
  fi
  if [ ! -f "$plugin/preview.md" ]; then
    echo "$plugin/preview.md: plugin preview source is required" >&2
    exit 1
  fi
  if [ ! -f "$plugin/assets/preview.png" ]; then
    echo "$plugin/assets/preview.png: plugin preview is required" >&2
    exit 1
  fi
done

if [ "$found" -eq 0 ]; then
  echo "no plugin manifests found" >&2
  exit 1
fi
