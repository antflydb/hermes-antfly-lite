#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
version=$(jq -er '.version' "$project_dir/plugin.json")
target="$(go env GOOS)-$(go env GOARCH)"
package_name="antfly-hermes-lite-$version-$target"
artifact=${1:-"$project_dir/dist/$package_name.tar.gz"}

verify_sha256() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c "$1"
  else
    sha256sum --check "$1"
  fi
}

if [[ ! -f "$artifact" ]]; then
  echo "error: package archive not found: $artifact" >&2
  exit 1
fi

package_test_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-package-smoke.XXXXXX")
trap 'rm -rf "$package_test_dir"' EXIT
tar -xzf "$artifact" -C "$package_test_dir"
package_root="$package_test_dir/$package_name"
if [[ ! -d "$package_root" ]]; then
  echo "error: archive did not contain expected root: $package_name" >&2
  exit 1
fi

(
  cd "$package_root"
  verify_sha256 SHA256SUMS
)
test -x "$package_root/scripts/support-agent-pilot.sh"
test -x "$package_root/scripts/configure-support-profile.sh"
test -x "$package_root/scripts/launch-support-dashboard.sh"
test -x "$package_root/scripts/import-github-docs.sh"
test -x "$package_root/bin/antfly-hermes-github"
jq -ce . "$package_root/evals/support-agent-conversations.jsonl" >/dev/null
jq -ce . "$package_root/evals/antfly-github-conversations.jsonl" >/dev/null
jq -ce . "$package_root/evals/antfly-github-docs.jsonl" >/dev/null

version_output=$("$package_root/bin/antfly-hermes-mcp" --version)
if [[ "$version_output" != *"version=$version"* || "$version_output" != *"target=$target"* ]]; then
  echo "error: unexpected packaged build identity: $version_output" >&2
  exit 1
fi

data_dir="$package_test_dir/data"
mkdir -p "$data_dir"
"$package_root/bin/antfly-hermes-setup" \
  --db "$data_dir/knowledge.aflite" \
  --input "$project_dir/examples/support-docs.jsonl" \
  --backup "$data_dir/knowledge.afb"
"$package_root/bin/antfly-hermes-maintain" doctor \
  --db "$data_dir/knowledge.aflite" \
  --backup "$data_dir/knowledge.afb" \
  --query "managed authentication" \
  | jq -e '.ready and .abi.valid and .database.integrity and .backup.fresh and .retrieval.healthy' >/dev/null

governed_db="$data_dir/governed.aflite"
"$package_root/bin/antfly-hermes-setup" \
  --db "$governed_db" \
  --input "$package_root/examples/support-governed.jsonl" \
  --audience support \
  --max-visibility internal >/dev/null
"$package_root/bin/antfly-hermes-eval" \
  --db "$governed_db" \
  --suite "$package_root/evals/support-retrieval.jsonl" \
  | jq -e '.passed and .expected_recall == 1 and .forbidden_leakage == 0 and .citation_validity == 1' >/dev/null

responses="$package_test_dir/responses.jsonl"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"package-smoke","version":"0.0.1"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"managed authentication customer sign-in","limit":3}}}' \
  | "$package_root/bin/antfly-hermes-mcp" --db "$data_dir/knowledge.aflite" > "$responses"

if ! grep -q 'support:authentication' "$responses"; then
  echo "error: packaged MCP search did not return the expected source" >&2
  exit 1
fi

echo "packaged connector smoke passed: $package_name"
