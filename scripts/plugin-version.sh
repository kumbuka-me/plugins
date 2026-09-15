#!/bin/sh
set -eu

plugin=${1:?plugin directory required}
manifest="$plugin/plugin.yaml"

if [ ! -f "$manifest" ]; then
  echo "missing plugin manifest: $manifest" >&2
  exit 1
fi

version=$(
  awk '
    $1 == "version:" { version = $2; count++ }
    END {
      if (count != 1 || version == "") exit 1
      print version
    }
  ' "$manifest"
) || {
  echo "plugin manifest must contain exactly one version: $manifest" >&2
  exit 1
}

printf '%s\n' "$version"
