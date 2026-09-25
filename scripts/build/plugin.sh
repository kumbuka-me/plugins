#!/bin/sh
set -eu

plugin=${1:?plugin directory required}
dist=${2:-dist}
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

plugin=${plugin%/}
manifest="$plugin/plugin.yaml"

if [ ! -f "$manifest" ]; then
  echo "unknown plugin: $plugin" >&2
  exit 1
fi

name=$(basename "$plugin")
version=$(./scripts/version/read.sh "$plugin")
sdk_version=$(go list -m -f '{{.Version}}' github.com/kumbuka-me/sdk)

if [ -z "$sdk_version" ]; then
  echo "unable to determine Kumbuka SDK version" >&2
  exit 1
fi

if [ -x "$plugin/generate.sh" ]; then
  "$plugin/generate.sh"
fi

./scripts/build/browser.sh "$plugin"
rm -rf "$plugin/dist"

(
  cd "$plugin"
  go run "github.com/kumbuka-me/sdk/cmd/kumbuka-plugin@$sdk_version" build
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
