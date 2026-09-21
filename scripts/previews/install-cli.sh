#!/bin/sh
set -eu

version=${1:?kumbuka-cli version required}
target=${2:?target path required}

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *)
    echo "Unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

archive_name="kumbuka-cli_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/kumbuka-me/cli/releases/download/v${version}"

temporary=$(mktemp -d "${TMPDIR:-/tmp}/kumbuka-cli.XXXXXX")
trap 'rm -rf "$temporary"' EXIT INT TERM

archive="$temporary/$archive_name"
checksums="$temporary/checksums.txt"

curl --fail --location --silent --show-error \
  --output "$archive" \
  "$base_url/$archive_name"
curl --fail --location --silent --show-error \
  --output "$checksums" \
  "$base_url/kumbuka-cli_${version}_checksums.txt"

expected=$(awk -v file="$archive_name" '
  $2 == file || $2 == "*" file {
    print $1
    exit
  }
' "$checksums")

if [ -z "$expected" ]; then
  echo "Checksum for $archive_name was not published" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$archive" | awk '{ print $1 }')
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "Checksum mismatch for $archive_name" >&2
  echo "expected: $expected" >&2
  echo "actual:   $actual" >&2
  exit 1
fi

tar -xzf "$archive" -C "$temporary"

binary=$(find "$temporary" -type f -name kumbuka-cli -print | head -n 1)
if [ -z "$binary" ]; then
  echo "$archive_name did not contain kumbuka-cli" >&2
  exit 1
fi

mkdir -p "$(dirname "$target")"
cp "$binary" "$target"
chmod 0755 "$target"
printf '%s\n' "Installed kumbuka-cli v$version to $target"
