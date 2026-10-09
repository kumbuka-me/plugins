#!/usr/bin/env bash
# Generate the first-party plugin update catalog from published GitHub releases.
set -euo pipefail

repository=${GITHUB_REPOSITORY:-kumbuka-me/plugins}
gh_command=${GH:-gh}
output=${1:-catalog.json}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/kumbuka-plugin-catalog.XXXXXX")
trap 'rm -rf "$tmp"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
entries="$tmp/entries.ndjson"
: >"$entries"

# decode_base64 supports the GNU and BSD base64 command variants.
decode_base64() {
  if printf '' | base64 --decode >/dev/null 2>&1; then
    base64 --decode
  else
    base64 -D
  fi
}

# manifest_field reads one top-level scalar field from a manifest at a release tag.
manifest_field() {
  local manifest=$1 field=$2
  printf '%s\n' "$manifest" | awk -v field="$field" '
    $1 == field ":" {
      sub(/^[^:]+:[[:space:]]*/, "")
      print
      exit
    }
  '
}

# package_checksum prefers GitHub's digest, falling back to the checksum asset.
package_checksum() {
  local digest=$1 checksum_url=$2 checksum
  checksum=${digest#sha256:}
  if [[ $checksum == "$digest" ]]; then
    checksum=$(curl --fail --silent --show-error --location "$checksum_url" | awk 'NR == 1 { print $1 }')
  fi
  [[ $checksum =~ ^[0-9a-fA-F]{64}$ ]] || return 1
  printf '%s' "$checksum" | tr '[:upper:]' '[:lower:]'
}

# append_release validates one published release and appends its catalog entry.
append_release() {
  local release=$1 tag plugin version manifest
  local api_version provider id name manifest_version
  local package_name checksum_name package_asset checksum_url package_url package_digest checksum released_at

  if [[ $(jq -r '.draft or .prerelease' <<<"$release") == true ]]; then
    return
  fi

  tag=$(jq -r '.tag_name' <<<"$release")
  if [[ ! $tag =~ ^([a-z0-9][a-z0-9-]*)/v((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))$ ]]; then
    printf 'Skipping unrelated release tag %s\n' "$tag" >&2
    return
  fi

  plugin=${BASH_REMATCH[1]}
  version=${BASH_REMATCH[2]}
  manifest=$(git show "$tag:$plugin/plugin.yaml" 2>/dev/null || true)
  if [[ -z $manifest ]]; then
    printf 'Skipping %s: manifest is unavailable at the release tag\n' "$tag" >&2
    return
  fi

  api_version=$(manifest_field "$manifest" api_version)
  provider=$(manifest_field "$manifest" provider)
  id=$(manifest_field "$manifest" id)
  name=$(manifest_field "$manifest" name)
  manifest_version=$(manifest_field "$manifest" version)
  if [[ ! $api_version =~ ^[1-9][0-9]*$ || $id != "me.kumbuka.$plugin" || $manifest_version != "$version" || -z $name || -z $provider ]]; then
    printf 'Skipping %s: release metadata does not match plugin.yaml\n' "$tag" >&2
    return
  fi

  package_name="$plugin-$version.kumbukaplugin"
  checksum_name="$package_name.sha256"
  package_asset=$(jq -c --arg name "$package_name" '.assets[]? | select(.name == $name)' <<<"$release" | head -n 1)
  checksum_url=$(jq -r --arg name "$checksum_name" '.assets[]? | select(.name == $name) | .browser_download_url' <<<"$release" | head -n 1)
  package_url=$(jq -r '.browser_download_url // empty' <<<"$package_asset")
  package_digest=$(jq -r '.digest // empty' <<<"$package_asset")

  if [[ -z $package_url || -z $checksum_url ]]; then
    printf 'Skipping %s: package or checksum release asset is missing\n' "$tag" >&2
    return
  fi

  if ! checksum=$(package_checksum "$package_digest" "$checksum_url"); then
    printf 'Skipping %s: package digest is invalid\n' "$tag" >&2
    return
  fi

  released_at=$(jq -r '.published_at // .created_at' <<<"$release")
  jq -cn \
    --arg id "$id" \
    --arg name "$name" \
    --arg provider "$provider" \
    --arg version "$version" \
    --argjson api_version "$api_version" \
    --arg released_at "$released_at" \
    --arg package_url "$package_url" \
    --arg sha256 "$checksum" \
    '{
      id: $id,
      name: $name,
      provider: $provider,
      version: $version,
      api_version: $api_version,
      released_at: $released_at,
      package_url: $package_url,
      sha256: $sha256
    }' >>"$entries"
}

# write_catalog sorts plugin releases by ID and semantic version.
write_catalog() {
  jq -s '
    sort_by(.id)
    | group_by(.id)
    | {
        schema_version: 1,
        plugins: map({
          id: .[0].id,
          name: .[0].name,
          provider: .[0].provider,
          versions: (
            sort_by(.version | split(".") | map(tonumber))
            | reverse
            | map({version, api_version, released_at, package_url, sha256})
          )
        })
      }
  ' "$entries" >"$tmp/catalog.json"

  mkdir -p "$(dirname "$output")"
  mv "$tmp/catalog.json" "$output"
}

# main fetches all releases before processing so API failures propagate.
main() {
  local encoded release
  "$gh_command" api --paginate "repos/$repository/releases?per_page=100" --jq '.[] | @base64' >"$tmp/releases.txt"
  while IFS= read -r encoded; do
    [[ -n $encoded ]] || continue
    release=$(printf '%s' "$encoded" | decode_base64)
    append_release "$release"
  done <"$tmp/releases.txt"
  write_catalog
}

main "$@"
