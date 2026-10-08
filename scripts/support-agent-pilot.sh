#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=scripts/lib/hermes-plugin.sh
source "$project_dir/scripts/lib/hermes-plugin.sh"

hermes_bin=${HERMES_BIN:-hermes}
hermes_profile=${HERMES_PROFILE:-supportpilot}
hermes_root=${HERMES_ROOT:-"$HOME/.hermes"}
profile_root=${HERMES_PROFILE_ROOT:-"$hermes_root/profiles/$hermes_profile"}
plugin_root=${ANTFLY_PLUGIN_ROOT:-"$profile_root/plugins/antfly-hermes-lite"}
plugin_data=$(antfly_resolve_plugin_data "$profile_root")
db=${ANTFLY_DB:-"$plugin_data/knowledge.aflite"}
backup=${ANTFLY_BACKUP:-"$plugin_data/knowledge.afb"}
manifest=${ANTFLY_CORPUS_MANIFEST:-"${db%.aflite}.manifest.json"}
suite=${SUPPORT_AGENT_SUITE:-}
retrieval_suite=${SUPPORT_RETRIEVAL_SUITE:-}
doctor_query=${SUPPORT_DOCTOR_QUERY:-}
model=${HERMES_MODEL:-gpt-5.6-sol}
provider=${HERMES_PROVIDER:-openai-codex}
skill_namespace=$(basename "$plugin_data")
support_skill=${HERMES_SUPPORT_SKILL:-"$skill_namespace:antfly-support"}
command=${1:-}

if [[ -f "$manifest" ]]; then
  if [[ -z "$doctor_query" ]]; then
    doctor_query=$(jq -er '.health_query' "$manifest")
  fi
  if [[ -z "$retrieval_suite" ]]; then
    retrieval_suite=$(antfly_resolve_plugin_file "$plugin_root" "$(jq -er '.evaluation.retrieval_suite' "$manifest")")
  fi
  if [[ -z "$suite" ]]; then
    conversation_ref=$(jq -r '.evaluation.conversation_suite // empty' "$manifest")
    if [[ -n "$conversation_ref" ]]; then
      suite=$(antfly_resolve_plugin_file "$plugin_root" "$conversation_ref")
    fi
  fi
elif [[ "$command" == "prepare" || -z "$command" ]]; then
  doctor_query=${doctor_query:-"password reset"}
  retrieval_suite=${retrieval_suite:-"$project_dir/evals/support-retrieval.jsonl"}
  suite=${suite:-"$project_dir/evals/support-agent-conversations.jsonl"}
else
  echo "error: active corpus manifest not found: $manifest" >&2
  echo "run the prepare command or promote a qualified manifest with the database" >&2
  exit 1
fi

usage() {
  cat <<'EOF'
Usage: scripts/support-agent-pilot.sh COMMAND [ARGUMENT]

Commands:
  prepare           Build the governed Support knowledge base in plugin data.
  verify            Run Antfly health, retrieval, policy, and citation checks.
  ask QUESTION      Run one grounded Hermes Support turn.
  conversations     Run and automatically check the conversation suite.

Environment overrides: HERMES_BIN, HERMES_PROFILE, HERMES_ROOT,
HERMES_PROFILE_ROOT, ANTFLY_PLUGIN_ROOT, ANTFLY_PLUGIN_DATA, ANTFLY_DB,
ANTFLY_BACKUP, ANTFLY_CORPUS_MANIFEST, HERMES_MODEL, HERMES_PROVIDER, SUPPORT_AGENT_SUITE,
SUPPORT_AGENT_OUTPUT, HERMES_SUPPORT_SKILL.
Retrieval overrides: SUPPORT_RETRIEVAL_SUITE, SUPPORT_DOCTOR_QUERY.
EOF
}

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "error: required file not found: $1" >&2
    exit 1
  fi
}

require_executable() {
  if [[ ! -x "$1" ]]; then
    echo "error: required executable not found: $1" >&2
    exit 1
  fi
}

prepare() {
  require_executable "$plugin_root/bin/antfly-hermes-setup"
  mkdir -p "$plugin_data"
  chmod 700 "$plugin_data"
  if [[ -f "$db" && -f "$backup" ]]; then
    if [[ ! -f "$manifest" ]]; then
      echo "error: database and backup exist without a corpus manifest: $plugin_data" >&2
      echo "qualify and promote the matching manifest; corpus identity cannot be inferred safely" >&2
      exit 1
    fi
    echo "support knowledge base already exists; leaving it unchanged"
    return
  fi
  if [[ -e "$db" || -e "$backup" ]]; then
    echo "error: only one of the database and backup exists; inspect the data directory before retrying: $plugin_data" >&2
    exit 1
  fi
  "$plugin_root/bin/antfly-hermes-setup" \
    --db "$db" \
    --input "$project_dir/examples/support-governed.jsonl" \
    --backup "$backup" \
    --manifest "$project_dir/examples/support-corpus-manifest.json" \
    --audience support \
    --max-visibility internal
}

verify() {
  require_executable "$plugin_root/bin/antfly-hermes-maintain"
  require_executable "$plugin_root/bin/antfly-hermes-eval"
  require_file "$db"
  require_file "$backup"
  "$plugin_root/bin/antfly-hermes-maintain" doctor \
    --db "$db" \
    --backup "$backup" \
    --query "$doctor_query" \
    | jq -e '.ready' >/dev/null
  "$plugin_root/bin/antfly-hermes-eval" \
    --db "$db" \
    --suite "$retrieval_suite" \
    | jq -e '.passed and .cases == 8 and .cases_passed == 8 and .expected_recall == 1 and .forbidden_leakage == 0 and .citation_validity == 1' >/dev/null
  echo "support knowledge verification passed"
}

run_turn() {
  local prompt=$1
  # Hermes currently prints its portable-MCP early-validation warning to stdout
  # before the stream-json init event. Keep this runner's stdout valid JSONL.
  "$hermes_bin" -p "$hermes_profile" chat \
    --query "$prompt" \
    --oneshot \
    --provider "$provider" \
    --model "$model" \
    --toolsets antfly-knowledge \
    --skills "$support_skill" \
    --format stream-json \
    --max-turns 8 \
    --run-budget 120 \
    --source support-pilot \
    | sed -n '/^{/p'
}

conversations() {
  require_file "$suite"
  require_file "$db"
  local output=${SUPPORT_AGENT_OUTPUT:-"$project_dir/pilot-results/support-agent-conversations.json"}
  local output_dir
  output_dir=$(dirname "$output")
  mkdir -p "$output_dir"
  : > "$output"

  local failures=0
  while IFS= read -r case_json; do
    [[ -n "$case_json" ]] || continue
    local case_id prompt events stderr_file response
    case_id=$(jq -er '.id' <<<"$case_json")
    prompt=$(jq -er '.prompt' <<<"$case_json")
    events=$(mktemp "${TMPDIR:-/tmp}/antfly-hermes-events.XXXXXX")
    stderr_file=$(mktemp "${TMPDIR:-/tmp}/antfly-hermes-stderr.XXXXXX")

    if ! run_turn "$prompt" >"$events" 2>"$stderr_file"; then
      jq -cn --arg id "$case_id" --arg error "$(<"$stderr_file")" \
        '{case_id:$id,passed:false,error:$error}' >>"$output"
      echo "FAIL $case_id: Hermes turn failed" >&2
      failures=$((failures + 1))
      rm -f "$events" "$stderr_file"
      continue
    fi

    response=$(jq -sr 'map(select(.type == "result")) | last.text // ""' "$events")
    if jq -se --arg response "$response" --argjson case "$case_json" '
      ($response | ascii_downcase) as $answer
      | ($case.required_terms // []) as $required
      | ($case.required_urls // []) as $urls
      | ($case.required_any_terms // []) as $any
      | ($case.forbidden_terms // []) as $forbidden
      | all($required[]; . as $term | $answer | contains($term | ascii_downcase))
      and all($urls[]; . as $url | $answer | contains($url | ascii_downcase))
      and (($any | length) == 0 or any($any[]; . as $term | $answer | contains($term | ascii_downcase)))
      and all($forbidden[]; . as $term | ($answer | contains($term | ascii_downcase) | not))
    ' "$events" >/dev/null \
      && jq -se 'any(.[]; .type == "tool_use" and .name == "mcp__antfly_knowledge__search_knowledge")' "$events" >/dev/null; then
      jq -cn --arg id "$case_id" --arg response "$response" \
        '{case_id:$id,passed:true,response:$response}' >>"$output"
      echo "PASS $case_id"
    else
      jq -cn --arg id "$case_id" --arg response "$response" \
        --arg error "response or retrieval assertion failed" \
        '{case_id:$id,passed:false,response:$response,error:$error}' >>"$output"
      echo "FAIL $case_id: response or retrieval assertion failed" >&2
      failures=$((failures + 1))
    fi
    rm -f "$events" "$stderr_file"
  done < "$suite"

  jq -s '{cases:length,passed:map(select(.passed))|length,failed:map(select(.passed|not))|length,results:.}' \
    "$output" >"$output.tmp"
  mv "$output.tmp" "$output"
  echo "conversation report: $output"
  [[ "$failures" -eq 0 ]]
}

case "$command" in
  prepare)
    prepare
    ;;
  verify)
    verify
    ;;
  ask)
    shift
    [[ $# -gt 0 ]] || { usage >&2; exit 2; }
    run_turn "$*"
    ;;
  conversations)
    conversations
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
