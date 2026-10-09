#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
RELEASE_WORKFLOW=${RELEASE_WORKFLOW:-release.yml}
RELEASE_REF=${RELEASE_REF:-main}
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/common.sh"

# dispatch_release creates the workflow for one missing plugin release.
dispatch_release() {
  plugin=$1
  version=${2#*/v}
  printf '%-22s %-10s CREATE\n' "$plugin" "v$version"
  "$GH" workflow run "$RELEASE_WORKFLOW" \
    --repo "$GITHUB_REPOSITORY" \
    --ref "$RELEASE_REF" \
    -f "plugin=$plugin" \
    -f "version=$version"
}

# main dispatches missing releases for the most recent tag of each plugin.
main() {
  latest=$(plugin_release_tags)
  if [ -z "$latest" ]; then
    echo 'No plugin release tags found.'
    return
  fi

  releases=$(plugin_published_releases)
  count=0
  tab=$(printf '\t')
  while IFS="$tab" read -r plugin tag; do
    if published_release "$releases" "$tag"; then
      continue
    fi
    dispatch_release "$plugin" "$tag"
    count=$((count + 1))
  done <<EOF_TAGS
$latest
EOF_TAGS

  if [ "$count" -eq 0 ]; then
    echo 'All latest plugin tags already have releases.'
  else
    printf 'Dispatched %s release workflow(s).\n' "$count"
  fi
}

main "$@"
