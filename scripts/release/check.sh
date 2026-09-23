#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/common.sh"

tags="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/tags" --jq '.[].name')"
releases="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/releases" --jq '.[].tag_name')"
latest=$(printf '%s\n' "$tags" | latest_plugin_tags)
if [ -z "$latest" ]; then
  echo "No plugin release tags found."
  exit 0
fi

tab=$(printf '\t')
while IFS="$tab" read -r plugin tag; do
  if printf '%s\n' "$releases" | grep -Fqx "$tag"; then
    printf "%-22s %-10s OK\n" "$plugin" "${tag#*/}"
  else
    printf "%-22s %-10s MISSING RELEASE\n" "$plugin" "${tag#*/}"
  fi
done <<EOF_TAGS
$latest
EOF_TAGS
