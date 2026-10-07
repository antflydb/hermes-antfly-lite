#!/usr/bin/env bash
set -euo pipefail

artifact=${1:-}
if [[ -z "$artifact" || ! -f "$artifact" ]]; then
  echo "usage: linux-package-smoke.sh <release.tar.gz>" >&2
  exit 2
fi

smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-linux-package.XXXXXX")
trap 'rm -rf "$smoke_dir"' EXIT
package_name=$(basename "$artifact" .tar.gz)
tar -xzf "$artifact" -C "$smoke_dir"
package_root="$smoke_dir/$package_name"
cd "$package_root"
sha256sum --check SHA256SUMS

mkdir data
bin/antfly-hermes-setup \
  --db data/knowledge.aflite \
  --input examples/support-governed.jsonl \
  --backup data/knowledge.afb \
  --audience support \
  --max-visibility internal >/dev/null
bin/antfly-hermes-maintain doctor \
  --db data/knowledge.aflite \
  --backup data/knowledge.afb \
  --query "password reset" > data/doctor.json
grep -q '"ready":true' data/doctor.json
bin/antfly-hermes-eval \
  --db data/knowledge.aflite \
  --suite evals/support-retrieval.jsonl > data/eval.json
grep -q '"passed": true' data/eval.json

printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"forgotten password reset","limit":3}}}' \
  | bin/antfly-hermes-mcp --db data/knowledge.aflite > data/mcp.jsonl
grep -q 'support:password-reset' data/mcp.jsonl

echo "Linux release package smoke passed: $package_name"
