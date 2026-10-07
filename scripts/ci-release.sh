#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
antfly_dir=${ANTFLY_SOURCE_DIR:-"$project_dir/../antfly-agent-memory"}

if [[ ! -d "$antfly_dir/zig" ]]; then
  echo "error: Antfly source checkout not found: $antfly_dir" >&2
  exit 1
fi

(
  cd "$antfly_dir/zig"
  zig build capi -Doptimize=ReleaseFast
)

ANTFLY_SOURCE_DIR="$antfly_dir" "$project_dir/scripts/build-local.sh"
"$project_dir/scripts/validate.sh"
"$project_dir/scripts/smoke.sh"
"$project_dir/scripts/concurrency-smoke.sh"
"$project_dir/scripts/failure-smoke.sh"
"$project_dir/scripts/doctor-smoke.sh"
"$project_dir/scripts/support-eval.sh"
"$project_dir/scripts/package-release.sh"
"$project_dir/scripts/package-smoke.sh"
