#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cli=${KUMBUKA_CLI:-}

# require_tools checks the CLI and browser tooling before generating previews.
require_tools() {
  if [ -z "$cli" ]; then
    echo 'KUMBUKA_CLI is required' >&2
    exit 1
  fi
  if [ ! -x "$cli" ]; then
    echo "kumbuka-cli is not executable: $cli" >&2
    exit 1
  fi
  if [ ! -x "$repository/node_modules/.bin/playwright" ]; then
    echo 'Playwright is not installed. Run npm ci first.' >&2
    exit 1
  fi
}

# selected_plugin reports whether a plugin is included in PREVIEW_PLUGINS.
selected_plugin() {
  plugin=$1
  [ -z "${PREVIEW_PLUGINS:-}" ] && return 0

  for selected in $(printf '%s' "$PREVIEW_PLUGINS" | tr ',' ' '); do
    [ "$selected" = "$plugin" ] && return 0
  done
  return 1
}

# build_packages builds requested plugins and strips unsafe preview capabilities.
build_packages() {
  printf '%s\n' 'Building plugin packages from the current checkout...'
  for manifest in "$repository"/*/plugin.yaml; do
    [ -f "$manifest" ] || continue
    plugin=$(basename "$(dirname "$manifest")")
    selected_plugin "$plugin" || continue
    "$repository/scripts/build/plugin.sh" "$plugin" "$dist" >/dev/null
  done

  # The static CLI exposes only render-safe capabilities; release files remain unchanged.
  set -- "$dist"/*.kumbukaplugin
  if [ -f "$1" ]; then
    "${SCRIPT_PYTHON:-$repository/bin/python-env/bin/python3}" "$repository/scripts/previews/package.py" "$@"
  fi
}

# prepare_sites generates temporary configuration and pages for each preview.
prepare_sites() {
  HOME="$cli_home" \
  XDG_CACHE_HOME="$cli_cache" \
  PREVIEW_REPOSITORY="$repository" \
  PREVIEW_DIST="$dist" \
  PREVIEW_WORK="$work" \
  PREVIEW_PLUGINS="${PREVIEW_PLUGINS:-}" \
    node "$repository/scripts/previews/prepare.mjs"
}

# render_sites builds each temporary site with the static Kumbuka CLI.
render_sites() {
  printf '%s\n' 'Rendering plugin previews with kumbuka-cli...'
  while IFS= read -r plugin; do
    [ -n "$plugin" ] || continue
    printf '  %s\n' "$plugin"
    HOME="$cli_home" \
    XDG_CACHE_HOME="$cli_cache" \
      "$cli" build \
        --config "$work/configs/$plugin.toml" \
        --plugins "$work/plugins/$plugin.toml" \
        --log-format text
  done <"$work/plugins.txt"
}

# capture_previews renders screenshots or checks browser assertions.
capture_previews() {
  if [ "${PREVIEW_SKIP_BROWSER_INSTALL:-0}" != '1' ]; then
    "$repository/node_modules/.bin/playwright" install chromium
  fi

  PREVIEW_REPOSITORY="$repository" \
  PREVIEW_SITE="$work/sites" \
  PREVIEW_BROWSER_CHANNEL="${PREVIEW_BROWSER_CHANNEL:-}" \
  PREVIEW_PLUGINS="${PREVIEW_PLUGINS:-}" \
  PREVIEW_ASSERT_ONLY="${PREVIEW_ASSERT_ONLY:-0}" \
    node "$repository/scripts/previews/capture.mjs"
}

# main uses an isolated working directory for all preview generation stages.
main() {
  require_tools
  work=$(mktemp -d "${TMPDIR:-/tmp}/kumbuka-plugin-previews.XXXXXX")
  trap 'rm -rf "$work"' 0
  trap 'exit 1' 1 2 3 15

  dist="$work/dist"
  cli_home="$work/home"
  cli_cache="$work/cache"
  mkdir -p "$dist" "$cli_home" "$cli_cache"

  build_packages
  prepare_sites
  render_sites
  capture_previews
}

main "$@"
