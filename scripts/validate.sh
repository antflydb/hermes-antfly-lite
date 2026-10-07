#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
cd "$project_dir"

jq empty plugin.json mcp.json
test "$(jq -r '.name' plugin.json)" = "antfly-hermes-lite"
test "$(jq -r '.mcpServers["antfly-knowledge"].type' mcp.json)" = "stdio"
test "$(jq -r '.mcpServers["antfly-knowledge"].command' mcp.json)" = "./bin/antfly-hermes-mcp"
go mod verify
if grep -Eq '^replace[[:space:]]' go.mod; then
  echo "error: release module must not contain a replace directive" >&2
  exit 1
fi
go test ./internal/mcp ./internal/policy ./internal/evidence
go vet ./internal/mcp ./internal/policy ./internal/evidence
bash -n scripts/*.sh
test -x scripts/build-local.sh
test -x scripts/smoke.sh
test -x scripts/concurrency-smoke.sh
test -x scripts/failure-smoke.sh
test -x scripts/package-release.sh
test -x scripts/package-smoke.sh
test -x scripts/doctor-smoke.sh
test -x scripts/support-eval.sh
test -x scripts/ci-release.sh
test -x scripts/linux-package-smoke.sh

if rg -n 'antflydb_[A-Za-z0-9_-]+' . --glob '!bin/**'; then
  echo "error: possible Antfly API key literal" >&2
  exit 1
fi

echo "static validation passed"
