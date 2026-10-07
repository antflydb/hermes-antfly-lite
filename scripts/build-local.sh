#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
antfly_dir=${ANTFLY_SOURCE_DIR:-"$project_dir/../antfly-agent-memory"}
native_lib_dir=${ANTFLY_NATIVE_LIB_DIR:-"$antfly_dir/zig/zig-out/lib"}
version=$(jq -er '.version' "$project_dir/plugin.json")
connector_commit=$(git -C "$project_dir" rev-parse --verify HEAD 2>/dev/null || printf 'uncommitted')
antfly_lite_version=$(cd "$project_dir" && go list -m -f '{{.Version}}' github.com/antflydb/antfly/go/pkg/antflylite)
target="$(go env GOOS)-$(go env GOARCH)"
ldflags="-s -w -X github.com/antflydb/hermes-antfly-lite/internal/buildinfo.Version=$version -X github.com/antflydb/hermes-antfly-lite/internal/buildinfo.Commit=$connector_commit -X github.com/antflydb/hermes-antfly-lite/internal/buildinfo.AntflyLite=$antfly_lite_version -X github.com/antflydb/hermes-antfly-lite/internal/buildinfo.Target=$target"
case "$(go env GOOS)" in
  darwin) runtime_linker_flags='-Wl,-rpath,@loader_path' ;;
  linux) runtime_linker_flags='-Wl,-rpath,$ORIGIN' ;;
  *)
    echo "error: unsupported build operating system: $(go env GOOS)" >&2
    exit 1
    ;;
esac
cgo_linker_flags="-L$native_lib_dir $runtime_linker_flags"

if [[ ! -f "$native_lib_dir/libantfly.dylib" && ! -f "$native_lib_dir/libantfly.so" ]]; then
  echo "error: build Antfly first with: (cd $antfly_dir/zig && zig build capi -Doptimize=ReleaseFast)" >&2
  exit 1
fi

mkdir -p "$project_dir/bin"
(
  cd "$project_dir"
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/antfly-hermes-mcp ./cmd/antfly-hermes-mcp
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/antfly-hermes-ingest ./cmd/antfly-hermes-ingest
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/antfly-hermes-setup ./cmd/antfly-hermes-ingest
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/antfly-hermes-maintain ./cmd/antfly-hermes-maintain
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/antfly-hermes-eval ./cmd/antfly-hermes-eval
  CGO_LDFLAGS="$cgo_linker_flags" go build -trimpath -buildvcs=false -ldflags "$ldflags" -o bin/phase0-lockprobe ./cmd/phase0-lockprobe
)

if [[ -f "$native_lib_dir/libantfly.dylib" ]]; then
  cp "$native_lib_dir/libantfly.dylib" "$project_dir/bin/"
fi
if [[ -f "$native_lib_dir/libantfly.so" ]]; then
  cp "$native_lib_dir/libantfly.so" "$project_dir/bin/"
fi

echo "built version=$version target=$target antfly_lite=$antfly_lite_version in $project_dir/bin"
