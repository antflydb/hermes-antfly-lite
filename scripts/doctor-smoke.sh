#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
doctor_test_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-doctor.XXXXXX")
trap 'rm -rf "$doctor_test_dir"' EXIT

db="$doctor_test_dir/knowledge.aflite"
backup="$doctor_test_dir/knowledge.afb"
restored="$doctor_test_dir/restored.aflite"
report="$doctor_test_dir/doctor.json"

"$project_dir/bin/antfly-hermes-setup" \
  --db "$db" \
  --input "$project_dir/examples/support-docs.jsonl" \
  --backup "$backup" >/dev/null
"$project_dir/bin/antfly-hermes-maintain" doctor \
  --db "$db" --backup "$backup" --query "managed authentication" > "$report"
jq -e '.ready and .abi.valid and .database.integrity and .backup.fresh and .retrieval.healthy and (.warnings | length == 0)' "$report" >/dev/null

chmod 0644 "$db"
if "$project_dir/bin/antfly-hermes-maintain" doctor \
  --db "$db" --backup "$backup" --query "managed authentication" > "$report"; then
  echo "error: doctor accepted an overly broad database mode" >&2
  exit 1
fi
jq -e '(.ready | not) and (.warnings | index("database is accessible to group or other users") != null)' "$report" >/dev/null
chmod 0600 "$db"

touch -t 202001010000 "$backup"
if "$project_dir/bin/antfly-hermes-maintain" doctor \
  --db "$db" --backup "$backup" --query "managed authentication" > "$report"; then
  echo "error: doctor accepted a stale backup" >&2
  exit 1
fi
jq -e '(.ready | not) and (.backup.fresh | not)' "$report" >/dev/null

"$project_dir/bin/antfly-hermes-setup" --db "$restored" --restore "$backup" >/dev/null
test "$(stat -f '%Sp' "$restored" 2>/dev/null || stat -c '%A' "$restored")" = "-rw-------"

echo "setup and doctor smoke passed"
