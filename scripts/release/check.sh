#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/common.sh"

# main reports whether the latest tag of each plugin has a GitHub release.
main() {
  latest=$(plugin_release_tags)
  if [ -z "$latest" ]; then
    echo 'No plugin release tags found.'
    return
  fi

  releases=$(plugin_published_releases)
  tab=$(printf '\t')
  while IFS="$tab" read -r plugin tag; do
    if published_release "$releases" "$tag"; then
      status='OK'
    else
      status='MISSING RELEASE'
    fi
    printf '%-22s %-10s %s\n' "$plugin" "${tag#*/}" "$status"
  done <<EOF_TAGS
$latest
EOF_TAGS
}

main "$@"
