#!/bin/sh
set -eu

found=0
for manifest in */plugin.yaml; do
  [ -f "$manifest" ] || continue
  found=1
  plugin=${manifest%/plugin.yaml}
  provider=$(awk '$1 == "provider:" { print $2; exit }' "$manifest")
  id=$(awk '$1 == "id:" { print $2; exit }' "$manifest")
  version=$(./scripts/plugin-version.sh "$plugin")

  if [ "$provider" != "Kumbuka" ]; then
    echo "$manifest: provider must be Kumbuka" >&2
    exit 1
  fi
  if [ "$id" != "me.kumbuka.$plugin" ]; then
    echo "$manifest: id must be me.kumbuka.$plugin" >&2
    exit 1
  fi
  case "$version" in
    *[!0-9.]*|.*|*..*|*.)
      echo "$manifest: invalid version $version" >&2
      exit 1
      ;;
  esac
done

if [ "$found" -eq 0 ]; then
  echo "no plugin manifests found" >&2
  exit 1
fi
