#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
cd "$project_dir"

jq empty plugin.json mcp.json examples/support-corpus-manifest.json
test "$(jq -r '.name' plugin.json)" = "antfly-hermes-lite"
test "$(jq -r '.mcpServers["antfly-knowledge"].type' mcp.json)" = "stdio"
test "$(jq -r '.mcpServers["antfly-knowledge"].command' mcp.json)" = "./bin/antfly-hermes-mcp"
go mod verify
if grep -Eq '^replace[[:space:]]' go.mod; then
  echo "error: release module must not contain a replace directive" >&2
  exit 1
fi
go test ./internal/mcp ./internal/policy ./internal/evidence ./internal/githubdocs ./internal/corpus ./cmd/antfly-hermes-github
go vet ./internal/mcp ./internal/policy ./internal/evidence ./internal/githubdocs ./internal/corpus
bash -n scripts/*.sh
source scripts/lib/hermes-plugin.sh
helper_test_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-helper.XXXXXX")
trap 'rm -rf "$helper_test_dir"' EXIT
mkdir -p "$helper_test_dir/plugin-data/agent-plugin-antfly-hermes-lite-test1234"
test "$(ANTFLY_PLUGIN_DATA= antfly_resolve_plugin_data "$helper_test_dir")" = \
  "$helper_test_dir/plugin-data/agent-plugin-antfly-hermes-lite-test1234"
test -x scripts/build-local.sh
test -x scripts/smoke.sh
test -x scripts/concurrency-smoke.sh
test -x scripts/failure-smoke.sh
test -x scripts/package-release.sh
test -x scripts/package-smoke.sh
test -x scripts/doctor-smoke.sh
test -x scripts/support-eval.sh
test -x scripts/support-agent-pilot.sh
test -x scripts/configure-support-profile.sh
test -x scripts/launch-support-dashboard.sh
test -x scripts/import-github-docs.sh
test -x scripts/ci-release.sh
test -x scripts/fetch-antfly-native.sh
test -x scripts/linux-package-smoke.sh

if grep -RInE 'antflydb_[A-Za-z0-9_-]+' . \
  --exclude-dir=.git --exclude-dir=.antfly-native --exclude-dir=bin --exclude-dir=dist; then
  echo "error: possible Antfly API key literal" >&2
  exit 1
fi

echo "static validation passed"
