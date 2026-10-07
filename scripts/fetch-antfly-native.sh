#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
target=${1:-"$(go env GOOS)-$(go env GOARCH)"}
output_dir=${ANTFLY_NATIVE_ROOT:-"$project_dir/.antfly-native"}
release_version=v0.2.1
release_base="https://github.com/antflydb/antfly/releases/download/$release_version"

case "$target" in
  darwin-arm64)
    asset=antfly_0.2.1_Darwin_arm64.tar.gz
    expected_sha256=d169bd4dfdee1cb007092770e62181061207649463a56a3ef02cbeb431aa1e62
    native_library=libantfly.dylib
    ;;
  linux-amd64)
    asset=antfly_0.2.1_Linux_x86_64_gnu.tar.gz
    expected_sha256=2378190c86966626e5a6a101918f66f0f175c8f7482a85415fc446a2a09427d7
    native_library=libantfly.so
    ;;
  linux-arm64)
    asset=antfly_0.2.1_Linux_arm64_gnu.tar.gz
    expected_sha256=b66c9684e2998d5c4fea79b15dc80ec317fbeeaf9188e5d64f463895a2924884
    native_library=libantfly.so
    ;;
  *)
    echo "error: unsupported Antfly native target: $target" >&2
    exit 1
    ;;
esac

download_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-native.XXXXXX")
trap 'rm -rf "$download_dir"' EXIT
archive="$download_dir/$asset"

curl --fail --location --silent --show-error --retry 3 \
  --output "$archive" "$release_base/$asset"
if command -v shasum >/dev/null 2>&1; then
  actual_sha256=$(shasum -a 256 "$archive" | awk '{print $1}')
else
  actual_sha256=$(sha256sum "$archive" | awk '{print $1}')
fi
if [[ "$actual_sha256" != "$expected_sha256" ]]; then
  echo "error: checksum mismatch for $asset" >&2
  echo "expected=$expected_sha256 actual=$actual_sha256" >&2
  exit 1
fi

mkdir -p "$output_dir"
tar -xzf "$archive" -C "$output_dir" ./include ./lib
if [[ ! -f "$output_dir/lib/$native_library" ]]; then
  echo "error: $asset does not contain lib/$native_library" >&2
  exit 1
fi

echo "prepared=$output_dir target=$target release=$release_version sha256=$actual_sha256"
