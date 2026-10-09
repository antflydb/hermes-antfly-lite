#!/usr/bin/env bash

antfly_resolve_plugin_data() {
  local profile_root=$1
  if [[ -n "${ANTFLY_PLUGIN_DATA:-}" ]]; then
    printf '%s\n' "$ANTFLY_PLUGIN_DATA"
    return
  fi

  local data_root="$profile_root/plugin-data"
  local matches=()
  if [[ -d "$data_root" ]]; then
    while IFS= read -r path; do
      matches+=("$path")
    done < <(find "$data_root" -mindepth 1 -maxdepth 1 -type d \
      -name 'agent-plugin-hermes-antfly-*' -print | sort)
  fi
  if [[ ${#matches[@]} -ne 1 ]]; then
    echo "error: expected exactly one Antfly plugin-data directory under $data_root; found ${#matches[@]}" >&2
    echo "run the Hermes plugin doctor once, or set ANTFLY_PLUGIN_DATA explicitly" >&2
    return 1
  fi
  printf '%s\n' "${matches[0]}"
}

antfly_resolve_plugin_file() {
  local plugin_root=$1
  local relative=$2
  if [[ -z "$relative" || "$relative" == /* || "$relative" == .. || "$relative" == ../* || \
    "$relative" == */.. || "$relative" == */../* ]]; then
    echo "error: corpus suite path must be plugin-relative: $relative" >&2
    return 1
  fi
  local resolved="$plugin_root/$relative"
  if [[ ! -f "$resolved" ]]; then
    echo "error: corpus suite is missing from the installed plugin: $resolved" >&2
    return 1
  fi
  printf '%s\n' "$resolved"
}
