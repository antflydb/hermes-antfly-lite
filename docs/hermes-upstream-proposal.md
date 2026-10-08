# Hermes upstream proposal

## Objective

Make Antfly Lite a first-class local context and memory backend for Hermes while
preserving Hermes' SQLite session ledger and built-in Markdown compatibility.

## Proposed contribution sequence

1. Publish `antfly-hermes-lite` in the Hermes plugin catalog as an installable
   `MemoryProvider` with prebuilt macOS and Linux artifacts.
2. Add Antfly Lite to `hermes memory setup` discovery and documentation as the
   local-first hybrid-memory option.
3. Qualify automatic recall, completed-turn synchronization, curated-memory
   mirroring, pre-compression checkpoints, scoped deletion, backups, and plugin
   host isolation in Hermes' provider conformance suite.
4. Measure install conversion, recall quality, latency, and provider failures
   before discussing a bundled or default-provider change.
5. Only after adoption, propose Antfly Lite as the default new-profile context
   backend. Existing profiles remain opt-in and migration remains reversible.

## Maintainer-facing boundary

The initial contribution does not replace Hermes' SQLite sessions, modify the
conversation schema, or require a service. The plugin owns one profile-local
`.aflite` memory file and uses the existing `MemoryProvider` lifecycle. Its MCP
server independently exposes governed, cited knowledge bases, so maintainers can
review memory and knowledge integration as separate capabilities.

## Evidence to include

- zero-key lexical setup and optional Antfly Inference hybrid retrieval;
- macOS arm64, Linux amd64, and Linux arm64 packages with checksums;
- out-of-process plugin-host verification;
- cross-session, cross-user, replacement, deletion, concurrency, backup, and
  inference-fallback tests;
- a Support-agent pilot using commit-pinned Antfly documentation;
- explicit limitations and a rollback path to built-in memory.

## Suggested maintainer pitch

Hermes currently separates session history, curated Markdown memory, and
external memory providers. Antfly Lite adds an embedded local database that can
serve scoped long-term memory and governed knowledge without replacing the
session ledger. The working provider installs as a normal Hermes plugin, starts
in zero-configuration lexical mode, and can opt into local hybrid retrieval via
Antfly Inference. We would like to upstream the provider catalog entry and its
conformance tests first, gather adoption evidence, and use that data to evaluate
whether Antfly Lite should become the default context backend for new profiles.
