#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
RELEASE_WORKFLOW=${RELEASE_WORKFLOW:-release.yml}
RELEASE_REF=${RELEASE_REF:-main}
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/common.sh"

tags="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/tags" --jq '.[].name')"
releases="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/releases" --jq '.[].tag_name')"
latest=$(printf '%s\n' "$tags" | latest_plugin_tags)
if [ -z "$latest" ]; then
  echo "No plugin release tags found."
  exit 0
fi

count=0
tab=$(printf '\t')
while IFS="$tab" read -r plugin tag; do
  if printf '%s\n' "$releases" | grep -Fqx "$tag"; then
    continue
  fi
  version="${tag#*/v}"
  printf "%-22s %-10s CREATE\n" "$plugin" "v$version"
  "$GH" workflow run "$RELEASE_WORKFLOW" \
    --repo "$GITHUB_REPOSITORY" \
    --ref "$RELEASE_REF" \
    -f "plugin=$plugin" \
    -f "version=$version"
  count=$((count + 1))
done <<EOF_TAGS
$latest
EOF_TAGS

if [ "$count" -eq 0 ]; then
  echo "All latest plugin tags already have releases."
else
  echo "Dispatched $count release workflow(s)."
fi
