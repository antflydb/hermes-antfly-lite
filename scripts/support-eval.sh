#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
eval_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-support-eval.XXXXXX")
trap 'rm -rf "$eval_dir"' EXIT

db="$eval_dir/support.aflite"
backup="$eval_dir/support.afb"
setup_output="$eval_dir/setup.txt"
report="$eval_dir/report.json"
public_db="$eval_dir/support-public.aflite"
restricted_db="$eval_dir/support-restricted.aflite"

"$project_dir/bin/antfly-hermes-setup" \
  --db "$db" \
  --input "$project_dir/examples/support-governed.jsonl" \
  --backup "$backup" \
  --audience support \
  --max-visibility internal > "$setup_output"

grep -q 'approved=6' "$setup_output"
grep -q 'skipped=6' "$setup_output"
"$project_dir/bin/antfly-hermes-maintain" doctor \
  --db "$db" --backup "$backup" --query "password reset" \
  | jq -e '.ready' >/dev/null
"$project_dir/bin/antfly-hermes-eval" \
  --db "$db" \
  --suite "$project_dir/evals/support-retrieval.jsonl" > "$report"
jq -e '.passed and .cases == 8 and .cases_passed == 8 and .expected_recall == 1 and .forbidden_leakage == 0 and .citation_validity == 1' "$report" >/dev/null

"$project_dir/bin/antfly-hermes-setup" \
  --db "$public_db" \
  --input "$project_dir/examples/support-governed.jsonl" \
  --audience support \
  --max-visibility public > "$eval_dir/public-setup.txt"
grep -q 'approved=3' "$eval_dir/public-setup.txt"
grep -q 'skipped=9' "$eval_dir/public-setup.txt"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"SSO login loop tenant callback","limit":6}}}' \
  | "$project_dir/bin/antfly-hermes-mcp" --db "$public_db" \
  | jq -e '.result.structuredContent.total_hits == 0' >/dev/null

"$project_dir/bin/antfly-hermes-setup" \
  --db "$restricted_db" \
  --input "$project_dir/examples/support-governed.jsonl" \
  --audience support \
  --max-visibility restricted > "$eval_dir/restricted-setup.txt"
grep -q 'approved=7' "$eval_dir/restricted-setup.txt"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"restricted executive escalation contacts routing","limit":6}}}' \
  | "$project_dir/bin/antfly-hermes-mcp" --db "$restricted_db" \
  | jq -e '.result.structuredContent.hits | any(.id == "support:vip-escalation")' >/dev/null

echo "support retrieval evaluation passed"
