# Hermes upstream proposal

## Objective

Make Antfly a first-class context and memory backend for Hermes, with embedded
Lite as the default local mode and shared Antfly Cloud instances as the hosted
mode, while preserving Hermes' SQLite session ledger and built-in Markdown
compatibility.

## Proposed contribution sequence

1. Publish `hermes-antfly` in the Hermes plugin catalog as an installable
   `MemoryProvider` using the universal bundle in
   `catalog/hermes-antfly`. It carries immutable, checksummed macOS arm64,
   Linux amd64, and Linux arm64 payloads and extracts only the current host's
   runtime into profile-local plugin data.
2. Add Antfly to `hermes memory setup` discovery and documentation as the
   local-first hybrid-memory option, using Lite by default.
3. Qualify automatic recall, completed-turn synchronization, curated-memory
   mirroring, pre-compression checkpoints, scoped deletion, backups, and plugin
   host isolation in Hermes' provider conformance suite.
4. Add an Antfly instance URL backend for hosted Hermes while preserving the
   same provider contract and keeping local Lite as the zero-key default.
5. Demonstrate explicit, verified promotion from a local Lite profile to the
   hosted Antfly backend without changing logical record identities or scopes.
6. Measure install conversion, recall quality, latency, and provider failures
   before discussing a bundled or default-provider change.
7. Only after adoption, propose Antfly as the default new-profile context
   backend. New local profiles begin in Lite mode; existing profiles remain
   opt-in and migration remains reversible.

## Maintainer-facing boundary

The initial contribution does not replace Hermes' SQLite sessions, modify the
conversation schema, or require a service. The plugin owns one profile-local
`.aflite` memory file and uses the existing `MemoryProvider` lifecycle. Its MCP
server independently exposes governed, cited knowledge bases, so maintainers can
review memory and knowledge integration as separate capabilities.

Hosted Hermes support is a follow-up milestone rather than an expansion of the
initial catalog PR. It will accept an Antfly instance URL and profile-scoped
secret, use separate logical namespaces for memory and knowledge, and run the
same behavior-conformance suite as Lite. Cloud outages must never create an
implicit local fallback database or dual-write split brain. See
[the roadmap](roadmap.md) for the configuration and acceptance criteria.

## Evidence to include

- zero-key lexical setup and optional Antfly Inference hybrid retrieval;
- macOS arm64, Linux amd64, and Linux arm64 packages with checksums;
- out-of-process plugin-host verification;
- cross-session, cross-user, replacement, deletion, concurrency, backup, and
  inference-fallback tests;
- a Support-agent pilot using commit-pinned Antfly documentation;
- explicit limitations and a rollback path to built-in memory.

## Submission gate

Hermes catalog CI performs an unauthenticated clone of the pinned repository.
The repository must therefore be public before submitting the entry. Generate
the catalog YAML only after the universal-bundle commit is final so its `sha`
can pin that exact commit. Until then, validate locally with
`scripts/catalog-smoke.sh` and Hermes' `plugins validate` command; do not open a
catalog PR whose clone gate is guaranteed to fail.

## Suggested maintainer pitch

Hermes currently separates session history, curated Markdown memory, and
external memory providers. Antfly adds a context database that begins as an
embedded Lite file and can later connect to a shared Cloud instance. It serves
scoped long-term memory and governed knowledge without replacing the session
ledger. The working provider installs as a normal Hermes plugin, starts in
zero-configuration lexical Lite mode, and can opt into local hybrid retrieval
via Antfly Inference. We would like to upstream the provider catalog entry and
its conformance tests first, gather adoption evidence, and use that data to
evaluate whether Antfly should become the default context backend for new
profiles.
