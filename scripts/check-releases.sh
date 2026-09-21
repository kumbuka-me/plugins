#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}

tags="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/tags" --jq '.[].name')"
releases="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/releases" --jq '.[].tag_name')"
for plugin in $(printf '%s\n' "$tags" | cut -d/ -f1 | sort -u); do
  tag=$(printf '%s\n' "$tags" | grep "^${plugin}/v" | sort -V | tail -1)
  if printf '%s\n' "$releases" | grep -Fqx "$tag"; then
    printf "%-22s %-10s OK\n" "$plugin" "${tag#*/}"
  else
    printf "%-22s %-10s MISSING RELEASE\n" "$plugin" "${tag#*/}"
  fi
done
