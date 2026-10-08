# Antfly Lite Knowledge and Memory for Hermes

Production-oriented, local-first knowledge connector for Hermes. It packages a
narrow stdio MCP server for governed knowledge plus a native Hermes memory
provider. Knowledge and profile-local long-term memory use separate Antfly Lite
`.aflite` databases so corpus promotion never mutates conversational memory.

This repository is a pre-release Support golden path. The Go binding is pinned
to the Antfly v0.2.1 source commit with a verified module checksum. Native
packages are qualified for macOS arm64, Linux amd64, and Linux arm64.

The Phase 0 feasibility gate has passed: the plugin builds, Hermes validates and
enables it, Hermes launches the bundled stdio server, and all three tools
operate against the persistent Antfly Lite database. See
[`docs/phase-0.md`](docs/phase-0.md) for the evidence and remaining gates.

Start with the [quickstart](docs/quickstart.md). Native runtimes are distributed
as platform-specific release archives; a source checkout alone is not an
installed plugin.

The [product roadmap](docs/roadmap.md) covers the Apache-2.0 release gate,
Hermes catalog adoption, hosted Hermes support through an Antfly instance URL,
Lite-to-Cloud promotion, and the longer-term context-system work.

To exercise the complete Hermes Support-agent flow, including an isolated
profile, governed corpus, deterministic retrieval evaluation, and cited model
conversations, follow the [Support-agent pilot](docs/support-agent-pilot.md).
Once installed and authenticated, launch its local web UI with
`scripts/launch-support-dashboard.sh`.

To build a governed Antfly Lite knowledge base from commit-pinned GitHub files,
see [GitHub documentation ingestion](docs/github-ingestion.md).

To make Antfly Lite the active context and memory backend for a Hermes profile,
see the [native memory provider guide](docs/memory-provider.md). The initial
provider supplies scoped lexical recall, optional Antfly Inference hybrid
retrieval, automatic fact and completed-turn capture, curated-memory mirroring,
and pre-compression checkpoints. Its default lexical mode needs no API key,
model download, Docker, or separate service.

The repository also contains a universal, no-download Hermes catalog bundle in
`catalog/antfly-hermes-lite`. It selects a checksummed runtime for macOS arm64,
Linux amd64, or Linux arm64 and expands only that payload into profile-local
plugin data on first use.

## Model-visible knowledge tools

- `search_knowledge`: full-text search over approved knowledge chunks.
- `get_source`: retrieve one stored source by its stable identifier.
- `knowledge_status`: report database capabilities and active corpus provenance.

All three tools are read-only. Database creation, ingestion, indexing, backup,
restore, and promotion are human-operated commands and are never exposed to the
model.

The native provider also exposes `antfly_memory` with `remember`, `search`,
`forget`, and `status` actions. Automatic recall injects matched memories as
background evidence before each turn.

## Phase 0 local build

Build the Antfly C ABI from the pinned checkout:

```bash
cd ../antfly-agent-memory/zig
zig build capi -Doptimize=ReleaseFast
```

Build the prototype binaries on macOS:

```bash
mkdir -p bin
CGO_LDFLAGS="-L$PWD/../antfly-agent-memory/zig/zig-out/lib" \
  go build -o bin/antfly-hermes-mcp ./cmd/antfly-hermes-mcp
CGO_LDFLAGS="-L$PWD/../antfly-agent-memory/zig/zig-out/lib" \
  go build -o bin/antfly-hermes-ingest ./cmd/antfly-hermes-ingest
CGO_LDFLAGS="-L$PWD/../antfly-agent-memory/zig/zig-out/lib" \
  go build -o bin/antfly-hermes-setup ./cmd/antfly-hermes-ingest
CGO_LDFLAGS="-L$PWD/../antfly-agent-memory/zig/zig-out/lib" \
  go build -o bin/antfly-hermes-maintain ./cmd/antfly-hermes-maintain
cp ../antfly-agent-memory/zig/zig-out/lib/libantfly.dylib bin/
```

The portable MCP declaration points the dynamic loader at the plugin's `bin`
directory. For direct shell tests, run with:

```bash
export DYLD_LIBRARY_PATH="$PWD/../antfly-agent-memory/zig/zig-out/lib"
bin/antfly-hermes-ingest --db .phase0/knowledge.aflite \
  --input examples/support-docs.jsonl
bin/antfly-hermes-mcp --db .phase0/knowledge.aflite
```

The MCP process uses newline-delimited JSON-RPC on stdin/stdout. Logs go to
stderr so they cannot corrupt the protocol stream.

Run the repeatable local qualification checks with:

```bash
scripts/build-local.sh
scripts/validate.sh
scripts/smoke.sh
scripts/concurrency-smoke.sh
scripts/failure-smoke.sh
```

Create and test a platform-specific release archive after the local build:

```bash
scripts/package-release.sh
scripts/package-smoke.sh
```

The archive includes operator binaries, the native Antfly library, build
metadata, source provenance, license terms, and checksums. Packaged executables
use a loader-relative native-library path and run without the Antfly source
checkout.

## Knowledge-base setup and diagnostics

Create a knowledge base from approved JSONL records and write its first portable
backup:

```bash
bin/antfly-hermes-setup \
  --db ./knowledge.aflite \
  --input examples/support-governed.jsonl \
  --backup ./knowledge.afb \
  --manifest examples/support-corpus-manifest.json \
  --audience support \
  --max-visibility internal
```

The manifest records the corpus identity, provenance, health query, and
plugin-relative evaluation suites, and is installed beside the database as
`knowledge.manifest.json`. The setup path indexes only records whose `state` is
`approved`; other lifecycle
states, other audiences, excessive visibility, future-effective records, and
expired records are skipped before indexing. It creates the database and backup
with owner-only `0600` permissions. See the
[knowledge contract](docs/knowledge-contract.md) for the complete schema.

Run the production-readiness diagnostic with a broad query expected to match the
corpus:

```bash
bin/antfly-hermes-maintain doctor \
  --db ./knowledge.aflite \
  --backup ./knowledge.afb \
  --query "managed authentication"
```

`doctor` emits JSON and returns nonzero unless the C ABI matches, integrity is
valid, both artifacts have private permissions, the backup is current, and the
health query returns evidence. Restore into a new artifact with:

```bash
bin/antfly-hermes-setup \
  --db ./restored.aflite \
  --restore ./knowledge.afb
```

Run the Support retrieval and citation acceptance suite with:

```bash
scripts/support-eval.sh
```

The governed fixture covers public, internal, restricted, expired,
superseded, future-effective, cross-audience, contradictory, and injected
content. See [security](docs/security.md) and [operations](docs/operations.md)
for the deployment boundaries and lifecycle.

## Data ownership

The MCP sidecar opens the database read-only. Ingestion owns writer access and
must run separately. Use `.aflite` as the live database and `.afb` as the
portable backup format.
