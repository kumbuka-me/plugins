#!/usr/bin/env bash
# Interactive releases, compatible with macOS Bash 3.2 and Linux Bash.
set -euo pipefail
export LC_ALL=C
cd "$(dirname "$0")/../.."
remote=${RELEASE_REMOTE:-origin}
branch=${RELEASE_REF:-main}
make_command=${RELEASE_MAKE:-make}
fail() { printf 'release: %s\n' "$*" >&2; exit 1; }
prompt() { printf '%s' "$1"; read -r answer || [[ -n $answer ]]; }
bump_name() {
  case $1 in
    1|p|patch) printf patch ;; 2|m|minor) printf minor ;; 3|major) printf major ;; *) return 1 ;;
  esac
}
next_version() {
  local major minor patch
  IFS=. read -r major minor patch <<< "$1"
  case $2 in
    patch) printf '%s.%s.%s' "$major" "$minor" "$((patch+1))" ;;
    minor) printf '%s.%s.0' "$major" "$((minor+1))" ;;
    major) printf '%s.0.0' "$((major+1))" ;;
  esac
}
[[ -z $(git status --porcelain) ]] || fail 'working tree must be clean before releasing'
git var GIT_AUTHOR_IDENT >/dev/null
[[ $(git branch --show-current) == "$branch" ]] || fail "release must run on $branch"
git remote get-url "$remote" >/dev/null
plugins=(); versions=(); selected=(); bumps=(); next=(); tags=(); manifests=()
for manifest in */plugin.yaml; do
  [[ -f $manifest ]] || continue
  plugin=${manifest%/plugin.yaml}
  plugins+=("$plugin")
  versions+=("$(./scripts/version/read.sh "$plugin")")
done
((${#plugins[@]})) || fail 'no plugin manifests found'
printf 'Select plugins:\n\n'
for ((i=0; i<${#plugins[@]}; i++)); do printf '  %2d) %-24s %s\n' "$((i+1))" "${plugins[i]}" "${versions[i]}"; done
printf "\nEnter numbers/ranges such as 2,5-7 or 'all'.\n"
while :; do
  prompt 'Plugins: '
  selected=(); valid=1
  if [[ $answer == all || $answer == '*' ]]; then
    for ((i=0; i<${#plugins[@]}; i++)); do selected+=("$i"); done
  else
    tokens=(); read -r -a tokens <<< "${answer//,/ }"
    chosen=' '
    for token in "${tokens[@]}"; do
      if [[ $token =~ ^([0-9]+)(-([0-9]+))?$ && ${#token} -le 12 ]]; then
        start=$((10#${BASH_REMATCH[1]})); end=${BASH_REMATCH[3]:-${BASH_REMATCH[1]}}; end=$((10#$end))
        if ((start<1 || end<start || end>${#plugins[@]})); then valid=0; break; fi
        for ((i=start-1; i<end; i++)); do chosen+="$i "; done
      else valid=0; break
      fi
    done
    for ((i=0; i<${#plugins[@]}; i++)); do [[ $chosen != *" $i "* ]] || selected+=("$i"); done
  fi
  if ((valid && ${#selected[@]})); then break; fi
  echo 'Invalid selection.'
done
printf '\nVersion bump: 1) patch  2) minor  3) major  4) choose per plugin\n'
while :; do
  prompt 'Bump: '
  if [[ $answer == 4 || $answer == custom ]]; then
    for i in "${selected[@]}"; do
      while :; do
        prompt "${plugins[i]} ${versions[i]} [patch/minor/major]: "
        if bump=$(bump_name "$answer"); then bumps+=("$bump"); break; fi
        echo 'Invalid bump.'
      done
    done
    break
  elif bump=$(bump_name "$answer"); then
    for i in "${selected[@]}"; do bumps+=("$bump"); done
    break
  fi
  echo 'Invalid bump.'
done
printf '\nRelease plan:\n'
for ((j=0; j<${#selected[@]}; j++)); do
  i=${selected[j]}; plugin=${plugins[i]}
  next+=("$(next_version "${versions[i]}" "${bumps[j]}")")
  tags+=("$plugin/v${next[j]}"); manifests+=("$plugin/plugin.yaml")
  printf '  %-24s %s -> %s  %s\n' "$plugin" "${versions[i]}" "${next[j]}" "${tags[j]}"
done
printf '\nRun tests/lint, build selected plugins, create one commit and tag per plugin, push %s to %s, then push each tag separately.\n' "$branch" "$remote"
while :; do
  prompt 'Create these releases? [y/N]: '
  case $answer in y|yes|Y|YES) break ;; ''|n|no|N|NO) echo 'Release cancelled.'; exit 0 ;; *) echo 'Please answer yes or no.' ;; esac
done
remote_tags=$(git ls-remote --tags "$remote")
for tag in "${tags[@]}"; do
  if git show-ref --verify --quiet "refs/tags/$tag"; then fail "tag $tag already exists locally"; fi
  if awk '{print $2}' <<< "$remote_tags" | grep -Fxq "refs/tags/$tag"; then fail "tag $tag already exists on $remote"; fi
done
changed=0; commits_started=0
cleanup() {
  status=$?
  if ((changed && !commits_started)); then git restore -- "${manifests[@]}" || status=1; fi
  if ((status && commits_started)); then echo 'Release stopped. Existing commits and tags were kept; inspect them before retrying.' >&2; fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
expected_changes() {
  local actual expected
  actual=$(git status --porcelain)
  expected=$(printf ' M %s\n' "${manifests[@]}" | sort)
  [[ $(printf '%s\n' "$actual" | sort) == "$expected" ]] || fail 'validation changed files other than the selected manifests, or a version change is missing'
}
changed=1
for ((j=0; j<${#selected[@]}; j++)); do
  ./scripts/version/bump.sh "${plugins[${selected[j]}]}" "${bumps[j]}"
done
expected_changes
"$make_command" test
"$make_command" lint
for i in "${selected[@]}"; do "$make_command" build-plugin "PLUGIN=${plugins[i]}"; done
expected_changes
commits_started=1
for ((j=0; j<${#selected[@]}; j++)); do
  git add -- "${manifests[j]}"
  git commit -m "chore: release ${plugins[${selected[j]}]} v${next[j]}"
  git tag "${tags[j]}"
done
[[ -z $(git status --porcelain) ]] || fail 'release commits created but working tree is not clean'
git push "$remote" "$branch"
for tag in "${tags[@]}"; do git push "$remote" "refs/tags/$tag"; done
printf '\nReleased:\n'; printf '  %s\n' "${tags[@]}"
