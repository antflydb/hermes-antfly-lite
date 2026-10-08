#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
antfly_dir=${ANTFLY_SOURCE_DIR:-"$project_dir/../antfly-agent-memory"}

if [[ -n "${ANTFLY_NATIVE_LIB_DIR:-}" ]]; then
  if [[ ! -f "$ANTFLY_NATIVE_LIB_DIR/libantfly.dylib" && ! -f "$ANTFLY_NATIVE_LIB_DIR/libantfly.so" ]]; then
    echo "error: Antfly native library not found: $ANTFLY_NATIVE_LIB_DIR" >&2
    exit 1
  fi
else
  if [[ ! -d "$antfly_dir/zig" ]]; then
    echo "error: Antfly source checkout not found: $antfly_dir" >&2
    exit 1
  fi
  (
    cd "$antfly_dir/zig"
    zig_args=(build capi -Doptimize=ReleaseFast)
    if [[ -n "${ANTFLY_ZIG_MAXRSS:-}" ]]; then
      zig_args+=(--maxrss "$ANTFLY_ZIG_MAXRSS")
    fi
    zig "${zig_args[@]}"
  )
fi

ANTFLY_SOURCE_DIR="$antfly_dir" ANTFLY_NATIVE_LIB_DIR="${ANTFLY_NATIVE_LIB_DIR:-}" \
  "$project_dir/scripts/build-local.sh"
"$project_dir/scripts/validate.sh"
"$project_dir/scripts/smoke.sh"
"$project_dir/scripts/concurrency-smoke.sh"
"$project_dir/scripts/failure-smoke.sh"
"$project_dir/scripts/doctor-smoke.sh"
"$project_dir/scripts/memory-smoke.sh"
python3 "$project_dir/scripts/provider-smoke.py"
"$project_dir/scripts/catalog-smoke.sh"
"$project_dir/scripts/support-eval.sh"
"$project_dir/scripts/package-release.sh"
"$project_dir/scripts/package-smoke.sh"
