#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
repository=""
source_ref="main"
output=""
audience="support"
visibility="public"
paths=()

usage() {
  cat <<'EOF'
Usage: scripts/import-github-docs.sh --repo OWNER/REPO --output FILE [OPTIONS]

Options:
  --ref REF                 Branch, tag, or commit (default: main)
  --path PATH               Repository-relative file/directory; repeatable
  --audience AUDIENCE       Governed audience (default: support)
  --visibility VISIBILITY   public, internal, or restricted (default: public)

The output is governed JSONL ready for antfly-hermes-setup. GitHub credentials
are used only by gh/git during the temporary checkout and are never serialized.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo) repository=${2:-}; shift 2 ;;
    --ref) source_ref=${2:-}; shift 2 ;;
    --output) output=${2:-}; shift 2 ;;
    --path) paths+=("${2:-}"); shift 2 ;;
    --audience) audience=${2:-}; shift 2 ;;
    --visibility) visibility=${2:-}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ ! "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  echo "error: --repo must be OWNER/REPO" >&2
  exit 2
fi
if [[ -z "$output" || ! "$source_ref" =~ ^[A-Za-z0-9._/-]+$ || "$source_ref" == -* || \
  "$source_ref" == /* || "$source_ref" == */ || "$source_ref" == *..* ]]; then
  echo "error: --output is required and --ref must be a safe Git ref" >&2
  exit 2
fi
if [[ ${#paths[@]} -eq 0 ]]; then
  echo "error: at least one --path is required" >&2
  exit 2
fi
for path in "${paths[@]}"; do
  if [[ -z "$path" || "$path" == /* || "$path" == .. || "$path" == ../* ]]; then
    echo "error: paths must be non-empty and repository-relative: $path" >&2
    exit 2
  fi
done
if [[ -e "$output" ]]; then
  echo "error: output already exists: $output" >&2
  exit 1
fi
if [[ ! -x "$project_dir/bin/antfly-hermes-github" ]]; then
  echo "error: importer binary is missing; run scripts/build-local.sh first" >&2
  exit 1
fi
command -v gh >/dev/null || { echo "error: gh is required" >&2; exit 1; }
command -v git >/dev/null || { echo "error: git is required" >&2; exit 1; }

commit=$(gh api "repos/$repository/commits/$source_ref" --jq '.sha')
updated_at=$(gh api "repos/$repository/commits/$commit" --jq '.commit.committer.date')
if [[ ! "$commit" =~ ^[0-9a-f]{40}$ ]]; then
  echo "error: GitHub returned an invalid commit SHA" >&2
  exit 1
fi

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/antfly-github-import.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT
checkout="$work_dir/repository"
gh repo clone "$repository" "$checkout" -- --filter=blob:none --no-checkout --depth=1 >/dev/null
git -C "$checkout" fetch --quiet --depth=1 origin "$commit"
git -C "$checkout" checkout --quiet "$commit" -- "${paths[@]}"

args=(
  --repo-dir "$checkout"
  --repo "$repository"
  --commit "$commit"
  --updated-at "$updated_at"
  --output "$output"
  --audience "$audience"
  --visibility "$visibility"
)
for path in "${paths[@]}"; do
  args+=(--path "$path")
done
"$project_dir/bin/antfly-hermes-github" "${args[@]}"
