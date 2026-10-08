#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
version=$(jq -er '.version' "$project_dir/plugin.json")
target=${1:-"$(go env GOOS)-$(go env GOARCH)"}
host_target="$(go env GOOS)-$(go env GOARCH)"

sha256() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$@"
  else
    sha256sum "$@"
  fi
}

if [[ "$target" != "$host_target" ]]; then
  echo "error: package target $target does not match locally built target $host_target" >&2
  exit 1
fi

case "$target" in
  darwin-arm64)
    native_library=libantfly.dylib
    expected_arch='arm64'
    ;;
  linux-amd64)
    native_library=libantfly.so
    expected_arch='x86-64|x86_64'
    ;;
  linux-arm64)
    native_library=libantfly.so
    expected_arch='aarch64|ARM aarch64'
    ;;
  *)
    echo "error: unsupported release target: $target" >&2
    exit 1
    ;;
esac

required_files=(
  bin/antfly-hermes-mcp
  bin/antfly-hermes-setup
  bin/antfly-hermes-ingest
  bin/antfly-hermes-maintain
  bin/antfly-hermes-eval
  bin/antfly-hermes-github
  bin/antfly-hermes-memory
  "bin/$native_library"
  plugin.json
  __init__.py
  config_schema.py
  mcp.json
  README.md
  LICENSE
  SOURCE_PROVENANCE.md
  SECURITY.md
  skills/antfly-support/SKILL.md
  examples/support-governed.jsonl
  examples/support-corpus-manifest.json
  evals/support-retrieval.jsonl
  evals/support-agent-conversations.jsonl
  evals/antfly-github-docs.jsonl
  evals/antfly-github-conversations.jsonl
  docs/knowledge-contract.md
  docs/security.md
  docs/operations.md
  docs/quickstart.md
  docs/troubleshooting.md
  docs/support-agent-pilot.md
  docs/github-ingestion.md
  docs/hermes-upstream-proposal.md
  docs/memory-provider.md
  scripts/support-agent-pilot.sh
  scripts/configure-support-profile.sh
  scripts/launch-support-dashboard.sh
  scripts/import-github-docs.sh
  scripts/memory-smoke.sh
  scripts/provider-smoke.py
  scripts/lib/hermes-plugin.sh
)
for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$project_dir/$relative_path" ]]; then
    echo "error: required release file is missing: $relative_path" >&2
    exit 1
  fi
done

for executable in antfly-hermes-mcp antfly-hermes-setup antfly-hermes-ingest antfly-hermes-maintain antfly-hermes-eval antfly-hermes-github antfly-hermes-memory; do
  description=$(file "$project_dir/bin/$executable")
  if [[ ! "$description" =~ $expected_arch ]]; then
    echo "error: bin/$executable does not match $target: $description" >&2
    exit 1
  fi
done
native_description=$(file "$project_dir/bin/$native_library")
if [[ ! "$native_description" =~ $expected_arch ]]; then
  echo "error: bin/$native_library does not match $target: $native_description" >&2
  exit 1
fi

package_name="antfly-hermes-lite-$version-$target"
artifact="$project_dir/dist/$package_name.tar.gz"
checksum_file="$artifact.sha256"
if [[ -e "$artifact" || -e "$checksum_file" ]]; then
  echo "error: release artifact already exists: $artifact" >&2
  echo "remove or archive the existing generated artifact before packaging again" >&2
  exit 1
fi

package_tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-hermes-package.XXXXXX")
trap 'rm -rf "$package_tmp_dir"' EXIT
package_root="$package_tmp_dir/$package_name"
mkdir -p "$package_root/bin" "$package_root/skills/antfly-support" "$package_root/examples" \
  "$package_root/evals" "$package_root/docs" "$package_root/scripts/lib" "$project_dir/dist"

cp "$project_dir/plugin.json" "$project_dir/mcp.json" "$project_dir/README.md" "$project_dir/__init__.py" \
  "$project_dir/config_schema.py" \
  "$project_dir/LICENSE" "$project_dir/SOURCE_PROVENANCE.md" "$project_dir/SECURITY.md" "$package_root/"
cp "$project_dir/skills/antfly-support/SKILL.md" "$package_root/skills/antfly-support/"
cp "$project_dir/examples/support-governed.jsonl" "$project_dir/examples/support-corpus-manifest.json" \
  "$package_root/examples/"
cp "$project_dir/evals/support-retrieval.jsonl" "$project_dir/evals/support-agent-conversations.jsonl" \
  "$project_dir/evals/antfly-github-docs.jsonl" "$project_dir/evals/antfly-github-conversations.jsonl" \
  "$package_root/evals/"
cp "$project_dir/docs/knowledge-contract.md" "$project_dir/docs/security.md" \
  "$project_dir/docs/operations.md" "$project_dir/docs/quickstart.md" \
  "$project_dir/docs/troubleshooting.md" "$project_dir/docs/support-agent-pilot.md" \
  "$project_dir/docs/github-ingestion.md" "$project_dir/docs/hermes-upstream-proposal.md" \
  "$project_dir/docs/memory-provider.md" "$package_root/docs/"
cp "$project_dir/scripts/support-agent-pilot.sh" "$project_dir/scripts/configure-support-profile.sh" \
  "$project_dir/scripts/launch-support-dashboard.sh" "$project_dir/scripts/import-github-docs.sh" \
  "$project_dir/scripts/memory-smoke.sh" \
  "$project_dir/scripts/provider-smoke.py" \
  "$package_root/scripts/"
cp "$project_dir/scripts/lib/hermes-plugin.sh" "$package_root/scripts/lib/"
cp "$project_dir/bin/antfly-hermes-mcp" "$project_dir/bin/antfly-hermes-ingest" \
  "$project_dir/bin/antfly-hermes-setup" "$project_dir/bin/antfly-hermes-maintain" \
  "$project_dir/bin/antfly-hermes-eval" "$project_dir/bin/antfly-hermes-github" \
  "$project_dir/bin/antfly-hermes-memory" \
  "$project_dir/bin/$native_library" "$package_root/bin/"
chmod 0755 "$package_root/bin/antfly-hermes-mcp" "$package_root/bin/antfly-hermes-ingest" \
  "$package_root/bin/antfly-hermes-setup" "$package_root/bin/antfly-hermes-maintain" \
  "$package_root/bin/antfly-hermes-eval" "$package_root/scripts/support-agent-pilot.sh" \
  "$package_root/bin/antfly-hermes-github" "$package_root/scripts/configure-support-profile.sh" \
  "$package_root/bin/antfly-hermes-memory" \
  "$package_root/scripts/launch-support-dashboard.sh" "$package_root/scripts/import-github-docs.sh" \
  "$package_root/scripts/memory-smoke.sh"

connector_commit=$(git -C "$project_dir" rev-parse --verify HEAD 2>/dev/null || printf 'uncommitted')
antfly_lite_version=$(cd "$project_dir" && go list -m -f '{{.Version}}' github.com/antflydb/antfly/go/pkg/antflylite)
native_sha=$(sha256 "$package_root/bin/$native_library" | awk '{print $1}')
jq -n \
  --arg version "$version" \
  --arg target "$target" \
  --arg connector_commit "$connector_commit" \
  --arg antfly_lite_module "$antfly_lite_version" \
  --arg native_library "$native_library" \
  --arg native_sha256 "$native_sha" \
  '{version:$version,target:$target,connector_commit:$connector_commit,antfly_lite_module:$antfly_lite_module,native_library:$native_library,native_sha256:$native_sha256}' \
  > "$package_root/build-metadata.json"

(
  cd "$package_root"
  sha256 \
    LICENSE README.md SECURITY.md SOURCE_PROVENANCE.md __init__.py config_schema.py build-metadata.json mcp.json plugin.json \
    bin/antfly-hermes-eval bin/antfly-hermes-github bin/antfly-hermes-ingest bin/antfly-hermes-maintain bin/antfly-hermes-mcp bin/antfly-hermes-memory \
    bin/antfly-hermes-setup "bin/$native_library" evals/antfly-github-conversations.jsonl \
    evals/antfly-github-docs.jsonl evals/support-agent-conversations.jsonl \
    evals/support-retrieval.jsonl examples/support-corpus-manifest.json examples/support-governed.jsonl \
    docs/github-ingestion.md docs/hermes-upstream-proposal.md docs/knowledge-contract.md docs/memory-provider.md \
    docs/operations.md docs/quickstart.md docs/security.md docs/support-agent-pilot.md \
    docs/troubleshooting.md scripts/configure-support-profile.sh scripts/import-github-docs.sh scripts/lib/hermes-plugin.sh \
    scripts/memory-smoke.sh \
    scripts/provider-smoke.py \
    scripts/launch-support-dashboard.sh scripts/support-agent-pilot.sh skills/antfly-support/SKILL.md > SHA256SUMS
)

COPYFILE_DISABLE=1 tar -czf "$artifact" -C "$package_tmp_dir" "$package_name"
(
  cd "$project_dir/dist"
  sha256 "$(basename "$artifact")" > "$(basename "$checksum_file")"
)

echo "$artifact"
