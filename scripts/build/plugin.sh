#!/bin/sh
set -eu

plugin=${1:?plugin directory required}
dist=${2:-dist}
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

plugin=${plugin%/}
manifest="$plugin/plugin.yaml"
kumbuka_plugin="$root/bin/kumbuka-plugin"

if [ ! -f "$manifest" ]; then
  echo "unknown plugin: $plugin" >&2
  exit 1
fi

if [ ! -x "$kumbuka_plugin" ]; then
  echo "kumbuka-plugin is not installed: $kumbuka_plugin" >&2
  echo "run make download" >&2
  exit 1
fi

name=$(basename "$plugin")
version=$(./scripts/version/read.sh "$plugin")

if [ -x "$plugin/generate.sh" ]; then
  "$plugin/generate.sh"
fi

./scripts/build/browser.sh "$plugin"
rm -rf "$plugin/dist"

(
  cd "$plugin"
  "$kumbuka_plugin" build
)

source="$plugin/dist/$name.kumbukaplugin"
artifact="$dist/$name-$version.kumbukaplugin"

if [ ! -f "$source" ]; then
  echo "plugin build did not produce $source" >&2
  exit 1
fi

mkdir -p "$dist"
mv "$source" "$artifact"
rm -rf "$plugin/dist"
./scripts/build/checksum.sh "$artifact"
printf '%s\n' "$artifact"
