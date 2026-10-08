# GitHub documentation ingestion

The GitHub importer creates governed JSONL from a commit-pinned checkout. Antfly
Lite remains local: `gh` and `git` fetch source files into a temporary directory,
the converter writes normalized records, and the setup command builds a new
`.aflite` artifact. GitHub credentials are never written to the JSONL or database.

## Import files

Build the release binaries, authenticate `gh` when the repository is private,
then select explicit files or directories:

```bash
scripts/import-github-docs.sh \
  --repo antflydb/antfly \
  --ref main \
  --output /tmp/antfly-docs.jsonl \
  --manifest /tmp/antfly-docs.manifest.json \
  --corpus-id antfly-technical-docs \
  --health-query "Antfly Lite embedded local database" \
  --retrieval-suite evals/antfly-github-docs.jsonl \
  --conversation-suite evals/antfly-github-conversations.jsonl \
  --path docs/introduction.mdx \
  --path docs/guides/quickstart.mdx \
  --path docs/guides/lite.mdx
```

The importer resolves `--ref` to an immutable commit SHA before checkout. It
accepts UTF-8 Markdown, MDX, text, and reStructuredText; rejects escaping paths,
symlinks, oversized files, malformed repository identities, and existing output
paths; chunks text below the connector's 16 KiB record limit; and writes the
output and manifest with `0600` permissions.

Every record includes a stable path-derived ID, commit-pinned GitHub blob URL,
repository, commit SHA, source path, chunk position, audience, visibility,
lifecycle state, and commit timestamp. Re-importing a path at another commit
keeps its record IDs stable while updating its provenance URL and content.

## Build and qualify a candidate

Never write into the live database. Build a candidate and backup under new
filenames:

```bash
bin/antfly-hermes-setup \
  --db /tmp/knowledge-candidate.aflite \
  --input /tmp/antfly-docs.jsonl \
  --backup /tmp/knowledge-candidate.afb \
  --manifest /tmp/antfly-docs.manifest.json \
  --audience support \
  --max-visibility public

bin/antfly-hermes-maintain doctor \
  --db /tmp/knowledge-candidate.aflite \
  --backup /tmp/knowledge-candidate.afb \
  --query "Antfly Lite embedded local database"

bin/antfly-hermes-eval \
  --db /tmp/knowledge-candidate.aflite \
  --suite evals/antfly-github-docs.jsonl
```

Promotion requires stopped readers and writers. Archive the current `.aflite`
and `.afb`, archive the current manifest, then copy the qualified candidate into
the plugin data directory as `knowledge.aflite`, `knowledge.afb`, and
`knowledge.manifest.json`. Rerun `doctor` and the retrieval suite against those
final paths, then restart Hermes. Retain the archived artifacts for rollback.

The dashboard launcher discovers the profile-specific plugin-data namespace and
uses the active manifest's health query and evaluation suites automatically. An
environment override is only needed for an intentional one-off test.

## Synchronization policy

Scheduled synchronization should resolve a new commit, build an entirely new
candidate, and run the same gates. A failed fetch, conversion, health check, or
evaluation leaves the live database untouched. Automatic promotion should only
be enabled after the repository paths, content owners, audience, visibility,
and required evaluation cases are reviewed.
