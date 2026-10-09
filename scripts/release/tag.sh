#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

# usage reports the supported tagging modes.
usage() {
  echo "usage: $0 <plugin> | --all" >&2
  exit 2
}

# tag_plugin tags the current manifest version unless it is already tagged.
tag_plugin() (
  plugin=$1
  manifest="$plugin/plugin.yaml"
  if [ ! -f "$manifest" ]; then
    echo "unknown plugin: $plugin" >&2
    exit 1
  fi

  version=$(./scripts/version/read.sh "$plugin")
  tag="$plugin/v$version"
  if git show-ref --verify --quiet "refs/tags/$tag"; then
    printf '%s already exists\n' "$tag"
    exit 0
  fi

  git tag "$tag"
  printf 'Tagged %s\n' "$tag"
)

# tag_all tags every plugin in the current checkout.
tag_all() {
  found=0
  for manifest in */plugin.yaml; do
    [ -f "$manifest" ] || continue
    found=1
    tag_plugin "${manifest%/plugin.yaml}"
  done
  if [ "$found" -eq 0 ]; then
    echo 'no plugin manifests found' >&2
    exit 1
  fi
}

# main verifies the repository before creating any tags.
main() {
  if [ -n "$(git status --porcelain)" ]; then
    echo 'working tree must be clean before creating plugin tags' >&2
    exit 1
  fi

  [ "$#" -eq 1 ] || usage
  if [ "$1" = '--all' ]; then
    tag_all
  else
    tag_plugin "$1"
  fi
}

main "$@"
