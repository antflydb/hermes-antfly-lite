#!/usr/bin/env bash
set -euo pipefail

hermes_bin=${HERMES_BIN:-hermes}
hermes_profile=${HERMES_PROFILE:-supportpilot}
model=${HERMES_MODEL:-gpt-5.6-sol}
provider=${HERMES_PROVIDER:-openai-codex}
support_skill=${HERMES_SUPPORT_SKILL:-agent-plugin-antfly-hermes-lite-b5bf37d8:antfly-support}

if ! command -v "$hermes_bin" >/dev/null 2>&1 && [[ ! -x "$hermes_bin" ]]; then
  echo "error: Hermes executable not found: $hermes_bin" >&2
  exit 1
fi

"$hermes_bin" -p "$hermes_profile" plugins show antfly-hermes-lite >/dev/null
"$hermes_bin" -p "$hermes_profile" config set model.provider "$provider" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set model.default "$model" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set skills.auto_load "[\"$support_skill\"]" >/dev/null
"$hermes_bin" -p "$hermes_profile" config set platform_toolsets.cli '["antfly-knowledge"]' >/dev/null
"$hermes_bin" -p "$hermes_profile" config set agent.max_turns 8 >/dev/null
"$hermes_bin" -p "$hermes_profile" config check >/dev/null

echo "configured Hermes profile '$hermes_profile' for the Antfly Support agent"
