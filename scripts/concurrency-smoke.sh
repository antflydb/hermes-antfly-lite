#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-locks.XXXXXX")
writer_pid=""
reader_pid=""

cleanup() {
  if [[ -n "$writer_pid" ]]; then kill "$writer_pid" 2>/dev/null || true; fi
  if [[ -n "$reader_pid" ]]; then kill "$reader_pid" 2>/dev/null || true; fi
  rm -rf "$smoke_dir"
}
trap cleanup EXIT

export DYLD_LIBRARY_PATH="$project_dir/bin${DYLD_LIBRARY_PATH:+:$DYLD_LIBRARY_PATH}"
export LD_LIBRARY_PATH="$project_dir/bin${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

db="$smoke_dir/knowledge.aflite"
"$project_dir/bin/antfly-hermes-ingest" --db "$db" --input "$project_dir/examples/support-docs.jsonl" >/dev/null

"$project_dir/bin/phase0-lockprobe" --db "$db" --mode writer --hold 3s >"$smoke_dir/writer.out" &
writer_pid=$!
sleep 0.25

if "$project_dir/bin/phase0-lockprobe" --db "$db" --mode writer >"$smoke_dir/second-writer.out" 2>&1; then
  echo "error: second writer unexpectedly opened the database" >&2
  exit 1
fi
grep -q 'busy' "$smoke_dir/second-writer.out"

"$project_dir/bin/phase0-lockprobe" --db "$db" --mode readonly --hold 1s >"$smoke_dir/reader.out" &
reader_pid=$!
sleep 0.25
"$project_dir/bin/phase0-lockprobe" --db "$db" --mode readonly >"$smoke_dir/second-reader.out"

wait "$reader_pid"
reader_pid=""
wait "$writer_pid"
writer_pid=""

grep -q 'opened=writer' "$smoke_dir/writer.out"
grep -q 'opened=readonly' "$smoke_dir/reader.out"
grep -q 'opened=readonly' "$smoke_dir/second-reader.out"
echo "phase 0 concurrency smoke passed"
