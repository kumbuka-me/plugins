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
mkdir -p "$dist"

printf '%s\n' "Building plugin packages from the current checkout..."
for manifest in "$repository"/*/plugin.yaml; do
  [ -f "$manifest" ] || continue
  plugin=$(basename "$(dirname "$manifest")")
  "$repository/scripts/build-plugin.sh" "$plugin" "$dist" >/dev/null
done

mkdir -p "$work/home" "$work/cache"
export HOME="$work/home"
export XDG_CACHE_HOME="$work/cache"

PREVIEW_REPOSITORY="$repository" \
PREVIEW_DIST="$dist" \
PREVIEW_WORK="$work" \
  node "$repository/scripts/previews/prepare.mjs"

printf '%s\n' "Rendering plugin previews with kumbuka-cli..."
"$cli" build \
  --config "$work/kumbuka-site.toml" \
  --plugins "$work/.kumbukaplugins" \
  --log-format text

if [ "${PREVIEW_SKIP_BROWSER_INSTALL:-0}" != "1" ]; then
  "$repository/node_modules/.bin/playwright" install chromium
fi

PREVIEW_REPOSITORY="$repository" \
PREVIEW_SITE="$work/site" \
PREVIEW_BROWSER_CHANNEL="${PREVIEW_BROWSER_CHANNEL:-}" \
  node "$repository/scripts/previews/capture.mjs"
