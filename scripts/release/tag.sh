#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

usage() {
  echo "usage: $0 <plugin> | --all" >&2
  exit 2
}

if [ -n "$(git status --porcelain)" ]; then
  echo "working tree must be clean before creating plugin tags" >&2
  exit 1
fi

tag_plugin() {
  plugin=$1

  if [ ! -f "$plugin/plugin.yaml" ]; then
    echo "unknown plugin: $plugin" >&2
    exit 1
  fi

  version=$(./scripts/version/read.sh "$plugin")
  tag="$plugin/v$version"

  if git show-ref --verify --quiet "refs/tags/$tag"; then
    printf '%s already exists\n' "$tag"
    return
  fi

  git tag "$tag"
  printf 'Tagged %s\n' "$tag"
}

case $# in
1)
  if [ "$1" = "--all" ]; then
    found=0

    for manifest in */plugin.yaml; do
      [ -f "$manifest" ] || continue

      found=1
      tag_plugin "${manifest%/plugin.yaml}"
    done

    if [ "$found" -eq 0 ]; then
      echo "no plugin manifests found" >&2
      exit 1
    fi
  else
    tag_plugin "$1"
  fi
  ;;
*)
  usage
  ;;
esac
