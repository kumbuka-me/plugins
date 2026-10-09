#!/usr/bin/env bash
# Interactive releases, compatible with macOS Bash 3.2 and Linux Bash.
set -euo pipefail
export LC_ALL=C

cd "$(dirname "$0")/../.."

remote=${RELEASE_REMOTE:-origin}
branch=${RELEASE_REF:-main}
make_command=${RELEASE_MAKE:-make}

plugins=()
versions=()
selected=()
bumps=()
next_versions=()
tags=()
manifests=()
changed=0
commits_started=0

fail() {
  printf 'release: %s\n' "$*" >&2
  exit 1
}

prompt() {
  answer=
  printf '%s' "$1"
  read -r answer || [[ -n $answer ]]
}

bump_name() {
  case $1 in
  1 | p | patch) printf 'patch\n' ;;
  2 | m | minor) printf 'minor\n' ;;
  3 | major) printf 'major\n' ;;
  *) return 1 ;;
  esac
}

next_version() {
  local major minor patch
  IFS=. read -r major minor patch <<<"$1"
  case $2 in
  patch) printf '%s.%s.%s\n' "$major" "$minor" "$((patch + 1))" ;;
  minor) printf '%s.%s.0\n' "$major" "$((minor + 1))" ;;
  major) printf '%s.0.0\n' "$((major + 1))" ;;
  esac
}

check_repository() {
  [[ -z $(git status --porcelain) ]] || fail 'working tree must be clean before releasing'
  [[ $(git branch --show-current) == "$branch" ]] || fail "release must run on $branch"
  git var GIT_AUTHOR_IDENT >/dev/null
  git var GIT_COMMITTER_IDENT >/dev/null
  git remote get-url "$remote" >/dev/null
}

load_plugins() {
  local manifest plugin
  for manifest in */plugin.yaml; do
    [[ -f $manifest ]] || continue
    plugin=${manifest%/plugin.yaml}
    plugins+=("$plugin")
    versions+=("$(./scripts/version/read.sh "$plugin")")
  done
  ((${#plugins[@]} > 0)) || fail 'no plugin manifests found'
}

# parse_selection accepts indices and ranges, deduplicating in manifest order.
parse_selection() {
  local token start end index
  local -a tokens=() chosen=()
  selected=()

  if [[ $answer == all || $answer == '*' ]]; then
    for ((index = 0; index < ${#plugins[@]}; index++)); do
      selected+=("$index")
    done
    return 0
  fi

  read -r -a tokens <<<"${answer//,/ }"
  ((${#tokens[@]} > 0)) || return 1

  for token in "${tokens[@]}"; do
    [[ ${#token} -le 12 && $token =~ ^([0-9]+)(-([0-9]+))?$ ]] || return 1
    start=$((10#${BASH_REMATCH[1]}))
    end=$((10#${BASH_REMATCH[3]:-${BASH_REMATCH[1]}}))
    ((start >= 1 && end >= start && end <= ${#plugins[@]})) || return 1

    for ((index = start - 1; index < end; index++)); do
      chosen[index]=1
    done
  done

  for ((index = 0; index < ${#plugins[@]}; index++)); do
    [[ ${chosen[index]:-} == 1 ]] && selected+=("$index")
  done
  ((${#selected[@]} > 0))
}

choose_plugins() {
  local index
  printf 'Select plugins:\n\n'
  for ((index = 0; index < ${#plugins[@]}; index++)); do
    printf '  %2d) %-24s %s\n' "$((index + 1))" "${plugins[index]}" "${versions[index]}"
  done
  printf "\nEnter numbers/ranges such as 2,5-7 or 'all'.\n"

  while :; do
    prompt 'Plugins: '
    if parse_selection; then
      return
    fi
    printf 'Invalid selection.\n'
  done
}

choose_bumps() {
  local index bump
  printf '\nVersion bump: 1) patch  2) minor  3) major  4) choose per plugin\n'

  while :; do
    prompt 'Bump: '
    if [[ $answer == 4 || $answer == custom ]]; then
      for index in "${selected[@]}"; do
        while :; do
          prompt "${plugins[index]} ${versions[index]} [patch/minor/major]: "
          if bump=$(bump_name "$answer"); then
            bumps+=("$bump")
            break
          fi
          printf 'Invalid bump.\n'
        done
      done
      return
    fi

    if bump=$(bump_name "$answer"); then
      for index in "${selected[@]}"; do
        bumps+=("$bump")
      done
      return
    fi
    printf 'Invalid bump.\n'
  done
}

show_plan() {
  local item index plugin version
  printf '\nRelease plan:\n'

  for ((item = 0; item < ${#selected[@]}; item++)); do
    index=${selected[item]}
    plugin=${plugins[index]}
    version=$(next_version "${versions[index]}" "${bumps[item]}")
    next_versions+=("$version")
    tags+=("$plugin/v$version")
    manifests+=("$plugin/plugin.yaml")
    printf '  %-24s %s -> %s  %s\n' "$plugin" "${versions[index]}" "$version" "${tags[item]}"
  done

  printf '\nRun tests/lint, build selected plugins, create one commit and tag per plugin, push %s to %s, then push each tag separately.\n' "$branch" "$remote"
}

confirm_release() {
  while :; do
    prompt 'Create these releases? [y/N]: '
    case $answer in
    y | yes | Y | YES) return ;;
    '' | n | no | N | NO)
      printf 'Release cancelled.\n'
      exit 0
      ;;
    *) printf 'Please answer yes or no.\n' ;;
    esac
  done
}

check_tags() {
  local tag remote_tags
  remote_tags=$(git ls-remote --tags "$remote" | awk '{print $2}')

  for tag in "${tags[@]}"; do
    if git show-ref --verify --quiet "refs/tags/$tag"; then
      fail "tag $tag already exists locally"
    fi
    if grep -Fxq "refs/tags/$tag" <<<"$remote_tags"; then
      fail "tag $tag already exists on $remote"
    fi
  done
}

# Before the first commit, restore only the manifests modified by this wizard.
cleanup() {
  local status=$?
  if ((changed && !commits_started)); then
    git restore -- "${manifests[@]}" || status=1
  fi
  if ((status != 0 && commits_started)); then
    printf '%s\n' 'Release stopped. Existing commits and tags were kept; inspect them before retrying.' >&2
  fi
  exit "$status"
}

# Verify validations and builds did not modify other tracked or untracked files.
expected_changes() {
  local actual expected
  actual=$(git status --porcelain)
  expected=$(printf ' M %s\n' "${manifests[@]}" | sort)
  [[ $(printf '%s\n' "$actual" | sort) == "$expected" ]] ||
    fail 'validation changed files other than the selected manifests, or a version change is missing'
}

validate_releases() {
  local item index
  changed=1
  for ((item = 0; item < ${#selected[@]}; item++)); do
    index=${selected[item]}
    ./scripts/version/bump.sh "${plugins[index]}" "${bumps[item]}"
  done

  expected_changes
  "$make_command" test
  "$make_command" lint
  for index in "${selected[@]}"; do
    "$make_command" build-plugin "PLUGIN=${plugins[index]}"
  done
  expected_changes
}

publish_releases() {
  local item index tag
  commits_started=1
  for ((item = 0; item < ${#selected[@]}; item++)); do
    index=${selected[item]}
    git add -- "${manifests[item]}"
    git commit -m "chore: release ${plugins[index]} v${next_versions[item]}"
    git tag "${tags[item]}"
  done

  [[ -z $(git status --porcelain) ]] || fail 'release commits created but working tree is not clean'
  git push "$remote" "$branch"
  for tag in "${tags[@]}"; do
    git push "$remote" "refs/tags/$tag"
  done
  printf '\nReleased:\n'
  printf '  %s\n' "${tags[@]}"
}

main() {
  check_repository
  load_plugins
  choose_plugins
  choose_bumps
  show_plan
  confirm_release
  check_tags

  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  validate_releases
  publish_releases
}

main "$@"
