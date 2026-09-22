#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

usage() {
  echo "usage: $0 [<plugin> <patch|minor|major> | --all <patch|minor|major>]" >&2
  exit 2
}

split_version() {
  current=$1
  major=${current%%.*}
  rest=${current#*.}
  minor=${rest%%.*}
  patch=${rest#*.}

  case "$major" in
  '' | *[!0-9]*) return 1 ;;
  esac

  case "$minor" in
  '' | *[!0-9]*) return 1 ;;
  esac

  case "$patch" in
  '' | *[!0-9]* | *.*) return 1 ;;
  esac

  return 0
}

next_version() {
  current=$1
  bump=$2

  if ! split_version "$current"; then
    echo "invalid plugin version: $current (expected MAJOR.MINOR.PATCH)" >&2
    exit 1
  fi

  case "$bump" in
  patch)
    printf '%s.%s.%s\n' "$major" "$minor" "$((patch + 1))"
    ;;
  minor)
    printf '%s.%s.0\n' "$major" "$((minor + 1))"
    ;;
  major)
    printf '%s.0.0\n' "$((major + 1))"
    ;;
  *)
    echo "invalid bump: $bump (expected patch, minor, or major)" >&2
    exit 1
    ;;
  esac
}

write_version() {
  plugin=$1
  version=$2
  manifest="$plugin/plugin.yaml"

  tmp=$(mktemp "${TMPDIR:-/tmp}/kumbuka-plugin-version.XXXXXX")
  trap 'rm -f "$tmp"' EXIT INT TERM

  awk -v version="$version" '
    $1 == "version:" && !updated {
      print "version: " version
      updated = 1
      next
    }

    { print }

    END {
      if (!updated) exit 1
    }
  ' "$manifest" >"$tmp"

  mv "$tmp" "$manifest"
  trap - EXIT INT TERM
}

bump_plugin() {
  plugin=$1
  bump=$2
  manifest="$plugin/plugin.yaml"

  if [ ! -f "$manifest" ]; then
    echo "unknown plugin: $plugin" >&2
    exit 1
  fi

  current=$(./scripts/version/read.sh "$plugin")
  next=$(next_version "$current" "$bump")

  write_version "$plugin" "$next"

  printf '%s: %s -> %s\n' "$plugin" "$current" "$next"
}

version_all() {
  bump=$1

  case "$bump" in
  patch | minor | major) ;;
  *) usage ;;
  esac

  found=0

  for manifest in */plugin.yaml; do
    [ -f "$manifest" ] || continue

    found=1
    bump_plugin "${manifest%/plugin.yaml}" "$bump"
  done

  if [ "$found" -eq 0 ]; then
    echo "no plugin manifests found" >&2
    exit 1
  fi
}

wizard() {
  set -- */plugin.yaml

  if [ ! -f "$1" ]; then
    echo "no plugin manifests found" >&2
    exit 1
  fi

  printf 'Select plugin:\n\n'

  index=1
  for manifest; do
    plugin=${manifest%/plugin.yaml}
    version=$(./scripts/version/read.sh "$plugin")

    printf '  %2d) %-24s %s\n' "$index" "$plugin" "$version"

    index=$((index + 1))
  done

  printf '\nPlugin: '
  IFS= read -r choice

  case "$choice" in
  '' | *[!0-9]*)
    echo "invalid plugin selection: $choice" >&2
    exit 1
    ;;
  esac

  index=1
  selected=

  for manifest; do
    if [ "$index" -eq "$choice" ]; then
      selected=${manifest%/plugin.yaml}
      break
    fi

    index=$((index + 1))
  done

  if [ -z "$selected" ]; then
    echo "invalid plugin selection: $choice" >&2
    exit 1
  fi

  current=$(./scripts/version/read.sh "$selected")

  patch_version=$(next_version "$current" patch)
  minor_version=$(next_version "$current" minor)
  major_version=$(next_version "$current" major)

  printf '\nCurrent: %s v%s\n\n' "$selected" "$current"
  printf 'Select bump:\n\n'
  printf '  1) patch -> %s\n' "$patch_version"
  printf '  2) minor -> %s\n' "$minor_version"
  printf '  3) major -> %s\n' "$major_version"
  printf '\nBump: '

  IFS= read -r choice

  case "$choice" in
  1) bump=patch ;;
  2) bump=minor ;;
  3) bump=major ;;
  *)
    echo "invalid bump selection: $choice" >&2
    exit 1
    ;;
  esac

  bump_plugin "$selected" "$bump"
}

case $# in
0)
  wizard
  ;;
2)
  if [ "$1" = "--all" ]; then
    version_all "$2"
  else
    bump_plugin "$1" "$2"
  fi
  ;;
*)
  usage
  ;;
esac
