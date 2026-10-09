#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/hermes-antfly.XXXXXX")
trap 'rm -rf "$smoke_dir"' EXIT

export DYLD_LIBRARY_PATH="$project_dir/bin${DYLD_LIBRARY_PATH:+:$DYLD_LIBRARY_PATH}"
export LD_LIBRARY_PATH="$project_dir/bin${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

db="$smoke_dir/knowledge.aflite"
backup="$smoke_dir/knowledge.afb"
responses="$smoke_dir/responses.jsonl"
restored="$smoke_dir/restored.aflite"

"$project_dir/bin/antfly-hermes-ingest" \
  --db "$db" \
  --input "$project_dir/examples/support-docs.jsonl" \
  --backup "$backup"

printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"phase0-smoke","version":"0.0.1"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"managed authentication","limit":6}}}' \
  '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_source","arguments":{"id":"support:authentication"}}}' \
  '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"knowledge_status","arguments":{}}}' \
  | "$project_dir/bin/antfly-hermes-mcp" --db "$db" > "$responses"

jq -e -s '
  length == 5 and
  .[0].result.serverInfo.name == "hermes-antfly" and
  ([.[1].result.tools[].name] | sort) == (["get_source", "knowledge_status", "search_knowledge"] | sort) and
  (.[2].result.content[0].text | contains("Managed authentication")) and
  (.[3].result.content[0].text | contains("support:authentication")) and
  (.[4].result.content[0].text | contains("aflite"))
' "$responses" >/dev/null

# Reopen the database in a new process and prove the evidence persists.
printf '%s\n' \
  '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"deployment validation backup","limit":6}}}' \
  | "$project_dir/bin/antfly-hermes-mcp" --db "$db" \
  | jq -e 'select(.id == 6) | .result.content[0].text | contains("Deployment procedure")' >/dev/null

test -s "$db"
test -s "$backup"

"$project_dir/bin/antfly-hermes-maintain" restore --backup "$backup" --db "$restored"
"$project_dir/bin/antfly-hermes-maintain" check --db "$restored" \
  | jq -e '.valid == true and .record_count > 0' >/dev/null
printf '%s\n' \
  '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"managed authentication","limit":6}}}' \
  | "$project_dir/bin/antfly-hermes-mcp" --db "$restored" \
  | jq -e 'select(.id == 7) | .result.content[0].text | contains("Managed authentication")' >/dev/null

echo "phase 0 smoke passed"
