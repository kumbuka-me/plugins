#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cli=${KUMBUKA_CLI:-}

if [ -z "$cli" ]; then
  echo "KUMBUKA_CLI is required" >&2
  exit 1
fi
if [ ! -x "$cli" ]; then
  echo "kumbuka-cli is not executable: $cli" >&2
  exit 1
fi
if [ ! -x "$repository/node_modules/.bin/playwright" ]; then
  echo "Playwright is not installed. Run npm ci first." >&2
  exit 1
fi

work=$(mktemp -d "${TMPDIR:-/tmp}/kumbuka-plugin-previews.XXXXXX")
trap 'rm -rf "$work"' EXIT INT TERM

dist="$work/dist"
cli_home="$work/home"
cli_cache="$work/cache"
mkdir -p "$dist" "$cli_home" "$cli_cache"

printf '%s\n' "Building plugin packages from the current checkout..."
for manifest in "$repository"/*/plugin.yaml; do
  [ -f "$manifest" ] || continue
  plugin=$(basename "$(dirname "$manifest")")
  "$repository/scripts/build/plugin.sh" "$plugin" "$dist" >/dev/null
done

# The static CLI intentionally exposes only render-safe capabilities. Rewrite
# temporary preview packages to that subset without changing release manifests.
set -- "$dist"/*.kumbukaplugin
if [ -f "$1" ]; then
  (cd "$repository" && go run ./scripts/previews/package "$@")
fi

HOME="$cli_home" \
XDG_CACHE_HOME="$cli_cache" \
PREVIEW_REPOSITORY="$repository" \
PREVIEW_DIST="$dist" \
PREVIEW_WORK="$work" \
  node "$repository/scripts/previews/prepare.mjs"

printf '%s\n' "Rendering plugin previews with kumbuka-cli..."
while IFS= read -r plugin; do
  [ -n "$plugin" ] || continue
  printf '  %s\n' "$plugin"
  HOME="$cli_home" \
  XDG_CACHE_HOME="$cli_cache" \
    "$cli" build \
      --config "$work/configs/$plugin.toml" \
      --plugins "$work/plugins/$plugin.toml" \
      --log-format text
done < "$work/plugins.txt"

if [ "${PREVIEW_SKIP_BROWSER_INSTALL:-0}" != "1" ]; then
  "$repository/node_modules/.bin/playwright" install chromium
fi

PREVIEW_REPOSITORY="$repository" \
PREVIEW_SITE="$work/sites" \
PREVIEW_BROWSER_CHANNEL="${PREVIEW_BROWSER_CHANNEL:-}" \
  node "$repository/scripts/previews/capture.mjs"
