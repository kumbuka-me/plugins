#!/bin/sh
set -eu

GH=${GH:-gh}
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
RELEASE_WORKFLOW=${RELEASE_WORKFLOW:-release.yml}
RELEASE_REF=${RELEASE_REF:-main}

tags="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/tags" --jq '.[].name')"
releases="$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/releases" --jq '.[].tag_name')"
count=0
for plugin in $(printf '%s\n' "$tags" | cut -d/ -f1 | sort -u); do
  tag=$(printf '%s\n' "$tags" | grep "^${plugin}/v" | sort -V | tail -1)
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
done
if [ "$count" -eq 0 ]; then
  echo "All latest plugin tags already have releases."
else
  echo "Dispatched $count release workflow(s)."
fi
