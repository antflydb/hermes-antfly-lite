#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
hermes_bin=${HERMES_BIN:-hermes}
hermes_profile=${HERMES_PROFILE:-supportpilot}
dashboard_port=${HERMES_DASHBOARD_PORT:-0}

if [[ "${SUPPORT_DASHBOARD_SKIP_CONFIG:-0}" != "1" ]]; then
  HERMES_BIN="$hermes_bin" HERMES_PROFILE="$hermes_profile" \
    "$project_dir/scripts/configure-support-profile.sh"
fi

if [[ "${SUPPORT_DASHBOARD_SKIP_VERIFY:-0}" != "1" ]]; then
  HERMES_BIN="$hermes_bin" HERMES_PROFILE="$hermes_profile" \
    "$project_dir/scripts/support-agent-pilot.sh" verify
fi

if [[ "$dashboard_port" == "0" ]]; then
  echo "starting Support-agent dashboard on the next available localhost port"
else
  echo "starting Support-agent dashboard at http://127.0.0.1:$dashboard_port"
fi
exec "$hermes_bin" -p "$hermes_profile" dashboard \
  --isolated \
  --host 127.0.0.1 \
  --port "$dashboard_port" \
  "$@"
