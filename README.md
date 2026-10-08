# Antfly Lite Knowledge Connector for Hermes

Phase 0 prototype of a local-first knowledge connector. It packages a narrow
stdio MCP server for Hermes and stores the knowledge base in one Antfly Lite
`.aflite` database.

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

To exercise the complete Hermes Support-agent flow, including an isolated
profile, governed corpus, deterministic retrieval evaluation, and cited model
conversations, follow the [Support-agent pilot](docs/support-agent-pilot.md).
Once installed and authenticated, launch its local web UI with
`scripts/launch-support-dashboard.sh`.

## Model-visible tools

- `search_knowledge`: full-text search over approved knowledge chunks.
- `get_source`: retrieve one stored source by its stable identifier.
- `knowledge_status`: report database and retrieval capabilities.

All three tools are read-only. Database creation, ingestion, indexing, backup,
restore, and promotion are human-operated commands and are never exposed to the
model.

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
  --input examples/support-docs.jsonl \
  --backup ./knowledge.afb
```

The setup path indexes only records whose `state` is `approved`; other lifecycle
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
