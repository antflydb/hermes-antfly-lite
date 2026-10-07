#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-failures.XXXXXX")
trap 'rm -rf "$smoke_dir"' EXIT

export DYLD_LIBRARY_PATH="$project_dir/bin${DYLD_LIBRARY_PATH:+:$DYLD_LIBRARY_PATH}"
export LD_LIBRARY_PATH="$project_dir/bin${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

missing="$smoke_dir/missing.aflite"
if "$project_dir/bin/antfly-hermes-mcp" --db "$missing" </dev/null >"$smoke_dir/missing.out" 2>"$smoke_dir/missing.err"; then
  echo "error: missing database unexpectedly opened" >&2
  exit 1
fi
rg -qi 'not found' "$smoke_dir/missing.err"
test ! -e "$missing"

corrupt="$smoke_dir/corrupt.aflite"
cp "$project_dir/examples/support-docs.jsonl" "$corrupt"
if "$project_dir/bin/antfly-hermes-mcp" --db "$corrupt" </dev/null >"$smoke_dir/corrupt.out" 2>"$smoke_dir/corrupt.err"; then
  echo "error: corrupt database unexpectedly opened" >&2
  exit 1
fi
rg -qi 'invalid|truncated|internal' "$smoke_dir/corrupt.err"

valid_db="$smoke_dir/valid.aflite"
valid_backup="$smoke_dir/valid.afb"
incompatible_backup="$smoke_dir/incompatible.afb"
incompatible_target="$smoke_dir/incompatible.aflite"
"$project_dir/bin/antfly-hermes-setup" \
  --db "$valid_db" \
  --input "$project_dir/examples/support-docs.jsonl" \
  --backup "$valid_backup" >/dev/null
cp "$valid_backup" "$incompatible_backup"
# Portable backup format version is the little-endian u32 at byte offset 8.
printf '\143\000\000\000' | dd of="$incompatible_backup" bs=1 seek=8 conv=notrunc 2>/dev/null
if "$project_dir/bin/antfly-hermes-setup" \
  --db "$incompatible_target" \
  --restore "$incompatible_backup" \
  >"$smoke_dir/incompatible.out" 2>"$smoke_dir/incompatible.err"; then
  echo "error: incompatible backup unexpectedly restored" >&2
  exit 1
fi
rg -qi 'format/version compatibility' "$smoke_dir/incompatible.err"
test ! -e "$incompatible_target"

echo "phase 0 failure smoke passed"
