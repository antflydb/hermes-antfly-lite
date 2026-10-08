# Antfly Lite + Hermes roadmap

## Product objective

Make Antfly the local-first system of record for Hermes agent context. A user
can begin with embedded Antfly Lite and later move the same logical memory and
knowledge model to an Antfly instance without rewriting the agent.

Memory and governed knowledge remain separate lifecycle domains even when they
share Antfly's record identity, schema, retrieval, provenance, and backup
contracts. This does not initially replace Hermes' SQLite session ledger.

## Phase 1: Apache-2.0 release gate

- Release Antfly Lite under Apache License 2.0.
- Pin this connector to the Apache-licensed Antfly Lite source commit.
- Update the connector license, manifest, and source provenance.
- Rebuild macOS arm64, Linux amd64, and Linux arm64 payloads from that source.
- Regenerate checksums and the universal Hermes catalog bundle.
- Run the complete release, provider, failure, concurrency, and package suites.
- Publish immutable `v0.3.0` artifacts.

No current ELv2 runtime will be submitted as the final Hermes catalog build.

## Phase 2: Hermes catalog and adoption

- Update the draft Hermes catalog PR to the Apache-2.0 connector commit.
- Pass Hermes' anonymous-clone, exact-SHA, security, and plugin-validation
  gates.
- Mark the PR ready and address maintainer review.
- Verify a clean `hermes plugins install antfly-hermes-lite` installation.
- Run the Support-agent pilot through the catalog-installed package.
- Establish comparative recall, latency, reliability, storage, and setup
  baselines before proposing a default-provider change.

## Phase 3: Antfly instance URL for hosted Hermes

Add a backend abstraction to the same Hermes provider:

```yaml
memory:
  provider: antfly-hermes-lite
  antfly:
    backend: cloud
    instance_url: https://customer.antfly.example
    memory_namespace: hermes-memory
    knowledge_namespace: support-knowledge
```

Local profiles continue to use `backend: lite`. Hosted Hermes deployments use
`backend: cloud` with profile-scoped credentials from Hermes' secret store.

Required behavior:

- use the same logical record, scope, provenance, and deletion contracts in
  Lite and Cloud;
- keep memory and governed knowledge in separate logical namespaces;
- enforce user, workspace, agent, and visibility boundaries server-side;
- require HTTPS for non-local instance URLs;
- bound requests with explicit connect and response timeouts;
- expose the selected backend and a non-secret instance identity in status;
- never silently redirect failed Cloud writes into a local database, which
  would create split-brain memory;
- allow turns to continue without recalled context when Cloud is unavailable,
  while surfacing degraded health and failed writes;
- run one provider conformance suite against both backends.

Acceptance criteria for `v0.4.0`:

- a hosted Hermes profile can configure an Antfly instance URL and secret;
- automatic recall, turn synchronization, explicit memory tools, checkpoints,
  scoped deletion, and governed knowledge retrieval work through Cloud;
- retries remain idempotent and do not duplicate memories;
- tenant-isolation and outage tests pass;
- logs and status never expose credentials or memory content;
- local Lite behavior and zero-key installation remain unchanged.

## Phase 4: Lite-to-Cloud promotion

- Export a profile's Lite memory and knowledge through portable Antfly backup
  bundles.
- Restore into the configured Antfly instance while preserving stable record
  IDs, timestamps, scope, provenance, and supersession history.
- Verify record counts, schemas, index definitions, and representative queries.
- Switch the Hermes profile only after verification succeeds.
- Preserve the local backup and a documented rollback path.

Promotion must be explicit and transactional. The connector will not maintain
implicit dual writes between Lite and Cloud.

## Phase 5: differentiated context system

- Establish a temporal, provenance-aware context ledger for observations,
  inferred claims, confirmed facts, and superseded facts.
- Build a token-budgeted context compiler spanning scoped memory and governed
  knowledge with citations, conflicts, selection reasons, and deterministic
  inference-free fallback.
- Add entity and relationship retrieval only where evaluation shows measurable
  gains.
- Evaluate one bounded Hermes operational-state subsystem after memory and
  knowledge behavior is established.
- Use external adoption and evaluation evidence before proposing Antfly as the
  default context backend for new Hermes profiles.
