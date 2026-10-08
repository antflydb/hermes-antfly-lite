#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
runtime="$project_dir/bin/antfly-hermes-memory"
test -x "$runtime"

smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-memory.XXXXXX")
trap 'rm -rf "$smoke_dir"' EXIT
db="$smoke_dir/memory.aflite"

remembered=$(printf '%s' '{"method":"remember","text":"The user prefers concise weekly status reports","kind":"preference","scope":"user","importance":0.9,"context":{"agent_id":"support","user_id":"alice","session_id":"s1"}}' \
  | "$runtime" --db "$db")
memory_id=$(jq -er '.result.id' <<<"$remembered")

printf '%s' '{"method":"search","query":"concise weekly status","limit":6,"context":{"agent_id":"support","user_id":"alice","session_id":"s2"}}' \
  | "$runtime" --db "$db" \
  | jq -e --arg id "$memory_id" '.ok and (.result.hits | length) == 1 and .result.hits[0].id == $id' >/dev/null

printf '%s' '{"method":"search","query":"concise weekly status","limit":6,"context":{"agent_id":"support","user_id":"bob","session_id":"s3"}}' \
  | "$runtime" --db "$db" \
  | jq -e '.ok and (.result.hits | length) == 0' >/dev/null

printf '%s' '{"method":"forget_text","text":"The user prefers concise weekly status reports","kind":"preference","scope":"user","context":{"agent_id":"support","user_id":"bob","session_id":"s3"}}' \
  | "$runtime" --db "$db" \
  | jq -e '.ok and (.result.deleted | not)' >/dev/null

printf '%s' '{"method":"forget_text","text":"The user prefers concise weekly status reports","kind":"preference","scope":"user","context":{"agent_id":"support","user_id":"alice","session_id":"s2"}}' \
  | "$runtime" --db "$db" \
  | jq -e --arg id "$memory_id" '.ok and .result.deleted and .result.id == $id' >/dev/null

printf '%s' '{"method":"remember","text":"Keeps a feline companion","kind":"fact","scope":"user","context":{"user_id":"alice"},"embedding":[1,0]}' \
  | "$runtime" --db "$db" \
  | jq -e '.ok' >/dev/null
printf '%s' '{"method":"remember","text":"Schedules quarterly finance reviews","kind":"fact","scope":"user","context":{"user_id":"alice"},"embedding":[0,1]}' \
  | "$runtime" --db "$db" \
  | jq -e '.ok' >/dev/null
printf '%s' '{"method":"search","query":"household pet","limit":2,"context":{"user_id":"alice"},"embedding":[0.99,0.01]}' \
  | "$runtime" --db "$db" \
  | jq -e '.ok and .result.hits[0].text == "Keeps a feline companion"' >/dev/null

echo "Antfly memory runtime smoke passed"
