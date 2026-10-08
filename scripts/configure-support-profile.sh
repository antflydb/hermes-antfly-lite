#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=scripts/lib/hermes-plugin.sh
source "$project_dir/scripts/lib/hermes-plugin.sh"

hermes_bin=${HERMES_BIN:-hermes}
hermes_profile=${HERMES_PROFILE:-supportpilot}
hermes_root=${HERMES_ROOT:-"$HOME/.hermes"}
profile_root=${HERMES_PROFILE_ROOT:-"$hermes_root/profiles/$hermes_profile"}
model=${HERMES_MODEL:-gpt-5.6-sol}
provider=${HERMES_PROVIDER:-openai-codex}

if ! command -v "$hermes_bin" >/dev/null 2>&1 && [[ ! -x "$hermes_bin" ]]; then
  echo "error: Hermes executable not found: $hermes_bin" >&2
  exit 1
fi

"$hermes_bin" -p "$hermes_profile" plugins show antfly-hermes-lite >/dev/null
plugin_data=$(antfly_resolve_plugin_data "$profile_root")
skill_namespace=$(basename "$plugin_data")
support_skill=${HERMES_SUPPORT_SKILL:-"$skill_namespace:antfly-support"}
"$hermes_bin" -p "$hermes_profile" config set model.provider "$provider" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set model.default "$model" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set skills.auto_load "[\"$support_skill\"]" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set platform_toolsets.cli '["antfly-knowledge"]' >/dev/null
"$hermes_bin" -p "$hermes_profile" config set agent.max_turns 8 >/dev/null
"$hermes_bin" -p "$hermes_profile" config check >/dev/null

echo "configured Hermes profile '$hermes_profile' for the Antfly Support agent"
