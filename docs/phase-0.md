# Phase 0 qualification record

Status: core feasibility passed; release packaging remains in progress.

Pinned inputs:

- Antfly checkout: `antfly-agent-memory`, revision `0ee81e8d2` based on v0.2.1.
- Hermes: upstream revision `bf23bccef38f4a8b061ca54cfbda13d9d1e9720c`.
- Go: 1.26.2 on darwin/arm64.
- Zig: 0.16.0.
- Agent Plugins format: 1.0.0.

## Hypothesis

A portable Agent Plugin with a Go stdio MCP sidecar is the smallest production
shape. The sidecar embeds Antfly Lite through its Go/C ABI binding, opens the
knowledge artifact read-only, and exposes only narrow evidence tools.

## Required evidence

- [x] Build and load the pinned `libantfly` C ABI.
- [x] Create, close, reopen, and search a `.aflite` knowledge database.
- [x] MCP initialize and tool discovery succeed.
- [x] Only read-only knowledge tools are exposed.
- [x] Search and source lookup work after process restart.
- [x] Missing and corrupt database cases fail clearly.
- [x] An incompatible database case fails clearly.
- [x] One writer and supported reader combinations are recorded.
- [x] Backup, restore, and snapshot paths are exercised.
- [ ] Normal-Antfly promotion is exercised.
- [x] Hermes validates, discovers, and loads the portable plugin.
- [x] Platform packaging and native library loading are decided.

## Evidence log

### 2026-10-07 — first embedded MCP smoke

- Built `libantfly.dylib` from the pinned checkout with `zig build capi
  -Doptimize=ReleaseFast`.
- Built the Go ingestion and stdio MCP binaries against the pinned local Go
  binding and C ABI.
- Ingested three JSONL records: two approved records were indexed and one
  superseded record was skipped before indexing.
- `Check()` reported a valid `.aflite` database and `BackupToFile()` produced a
  non-empty `.afb` archive.
- MCP initialize and `tools/list` returned exactly `search_knowledge`,
  `get_source`, and `knowledge_status`.
- Search returned the approved managed-authentication evidence; direct lookup
  returned its stable source; status identified `aflite` storage.
- A second MCP process reopened the same database and retrieved the persisted
  deployment procedure.
- The upstream Go binding suite passed with the real C ABI, including writer,
  read-only, status-only, snapshot, restore, integrity, and concurrency cases:
  `go test -tags antflylite_capi ./...`.

### 2026-10-07 — failure and concurrency qualification

- A missing database caused the read-only MCP process to fail without creating
  a new file.
- A deliberately corrupt database failed to open rather than returning empty or
  misleading results.
- A live writer excluded a second writer with a clear busy result.
- Multiple read-only MCP processes opened and searched the database while the
  writer remained live.
- A portable `.afb` backup restored into a new `.aflite` database; integrity and
  retrieval both passed against the restored copy.

### 2026-10-07 — real Hermes integration

- Hermes' official `plugins validate` command accepted the plugin. Its security
  scan emitted the expected caution for the bundled native `libantfly.dylib`;
  no manifest, skill, or MCP validation errors remained.
- The plugin was copied into an isolated Hermes profile and enabled through the
  Hermes CLI.
- Hermes expanded `PLUGIN_ROOT` and `PLUGIN_DATA` so the executable and native
  library came from the plugin installation while `knowledge.aflite` remained
  in the profile's persistent plugin-data directory.
- Hermes' own MCP client launched the stdio sidecar and discovered exactly
  `search_knowledge`, `get_source`, and `knowledge_status`.
- Calls through the live Hermes session successfully searched for the approved
  authentication record, retrieved it by stable source ID, and reported
  `aflite` storage status. The subprocess then shut down cleanly.

### 2026-10-07 — reproducible dependency and macOS package

- Replaced the development-only Go `replace` directive with immutable Antfly
  Lite module version `v0.0.0-20260909002942-e5261e9e2937`, which resolves to
  the Antfly v0.2.1 source commit. `go.sum` locks both module checksums and
  `go mod verify` passes.
- Added connector, Antfly Lite, and target-platform build identity to all
  operator binaries and to the MCP initialize response.
- Built an 11 MB macOS/arm64 release archive containing the portable plugin,
  operator binaries, `libantfly.dylib`, exact license terms, source
  provenance, build metadata, per-file checksums, and an archive checksum.
- Packaged binaries use a loader-relative native-library path. An extracted
  archive created and backed up a database, checked integrity, launched the MCP
  sidecar, and retrieved expected evidence without the Antfly source checkout.
- Hermes' official validator accepted the extracted archive, with only the
  expected native-binary caution.
- A fresh Hermes home enabled the packaged plugin and Hermes' own MCP client
  retrieved the approved fixture from profile-scoped plugin data.
- Disabling and removing the temporary plugin installation preserved both
  `knowledge.aflite` and its `knowledge.afb` backup.
- Linux x86-64 and Linux arm64 packages remain unqualified, so the overall
  platform packaging gate remains open.

### 2026-10-07 — setup and doctor baseline

- Added a user-facing `antfly-hermes-setup` entry point for creating a database
  from governed JSONL input or restoring a portable `.afb` into a new artifact.
- Setup, backup, and restore now enforce owner-only `0600` permissions for both
  live and portable knowledge artifacts.
- Added a machine-readable `doctor` command that verifies connector build
  identity, C ABI compatibility, database integrity, file permissions, backup
  freshness, Lite status, retrieval mode, and a caller-selected health query.
- `doctor` fails readiness for broad local permissions, stale backups, invalid
  integrity, ABI errors, or a health query returning no evidence.
- The setup/doctor smoke covers successful creation, permission rejection,
  stale-backup rejection, and secure restore. The extracted release-package
  smoke now requires `doctor` readiness before testing MCP retrieval.

### 2026-10-07 — governed Support path and Linux matrix

- Ingestion now fails closed on missing or invalid lifecycle, audience,
  visibility, timestamps, source URLs, duplicate IDs, and size limits.
- The governed fixture includes public, internal, restricted, expired,
  superseded, future-effective, cross-audience, contradictory, and
  prompt-injected records. Default Support ingestion writes six authorized
  records and reports six deterministic skip reasons.
- Public-only, internal, and explicitly restricted artifacts were exercised
  independently; restricted and cross-audience source IDs do not leak into the
  default corpus.
- MCP search and source lookup now return normalized structured evidence with
  stable citations and risk labels, without exposing Antfly backend fields such
  as `stored_json`.
- The eight-case Support retrieval suite passes with 100% expected-source
  recall, zero forbidden-source leakage, and 100% citation validity. It also
  records database-open and query latency.
- An incompatible `.afb` version fails restore without publishing a target
  database and emits format/version compatibility guidance.
- Linux amd64 and Linux arm64 C ABI libraries and connector binaries were built,
  packaged, checksum-verified, and exercised in matching Ubuntu containers.
  Setup, doctor, evaluation, and MCP retrieval passed on both targets.
- CI and tag-release workflows use immutable action SHAs and build macOS arm64,
  Linux amd64, and Linux arm64 natively from the pinned Antfly source commit.
- A final clean Hermes profile consumed normalized `structuredContent`, verified
  the citation URL, and confirmed the backend serialization was absent.
