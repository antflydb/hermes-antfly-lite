#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
bundle="$project_dir/catalog/hermes-antfly"
version=$(jq -er '.version' "$bundle/plugin.json")
source_version=$(jq -er '.version' "$project_dir/plugin.json")

if [[ "$version" != "$source_version" ]]; then
  echo "error: catalog version $version does not match source version $source_version" >&2
  exit 1
fi
cmp "$project_dir/__init__.py" "$bundle/__init__.py"
cmp "$project_dir/config_schema.py" "$bundle/config_schema.py"
cmp "$project_dir/plugin.json" "$bundle/plugin.json"

python3 - "$bundle" <<'PY'
import hashlib
import json
import sys
from pathlib import Path

bundle = Path(sys.argv[1])
manifest = json.loads((bundle / "payloads" / "manifest.json").read_text(encoding="utf-8"))
for target, metadata in manifest["payloads"].items():
    archive = bundle / "payloads" / f"hermes-antfly-{manifest['version']}-{target}.tar.gz"
    actual = hashlib.sha256(archive.read_bytes()).hexdigest()
    if actual != metadata["sha256"]:
        raise SystemExit(f"checksum mismatch for {archive.name}")
PY

runtime_home=$(mktemp -d "${TMPDIR:-/tmp}/antfly-catalog-smoke.XXXXXX")
trap 'rm -rf "$runtime_home"' EXIT
printf '%s' '{"method":"status"}' \
  | ANTFLY_RUNTIME_DIR="$runtime_home/runtime" \
    "$bundle/bin/antfly-hermes-memory" --db "$runtime_home/memory.aflite" \
  | jq -e '.ok and .result.ready' >/dev/null
python3 "$project_dir/scripts/provider-smoke.py" --plugin-root "$bundle"

echo "Hermes universal catalog bundle smoke passed"
