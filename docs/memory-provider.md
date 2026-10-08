# Native Hermes memory provider

Antfly Lite can be selected as Hermes' profile-local long-term memory provider:

```bash
hermes plugins enable antfly-hermes-lite
hermes config set memory.provider antfly-hermes-lite
hermes memory status
```

Restart active Hermes sessions after changing the provider. No daemon, API key,
Docker service, or embedding model is required.

## What it owns

The provider stores long-term memory in
`$HERMES_HOME/antfly-memory/memory.aflite`. Hermes' SQLite session history
remains the authoritative conversation ledger, and the connector's governed
knowledge corpus remains in `${PLUGIN_DATA}/knowledge.aflite`. These are
deliberately separate lifecycle domains:

- session history records the complete interaction;
- Antfly memory recalls durable context across sessions;
- the knowledge corpus supplies approved, cited source material.

The provider automatically recalls relevant memories before each turn, records
completed primary-agent turns asynchronously, mirrors successful built-in
`MEMORY.md` and `USER.md` add/replace/remove operations, and commits a durable
checkpoint before Hermes compresses context. Child and delegated agents can
read scoped context, but only the primary agent writes automatically.

## Scope and isolation

Every record has one scope: `user`, `workspace`, `agent`, `session`, or
`profile`. The default is `user` when Hermes supplies a user identity, with a
safe fallback to workspace, agent, session, and finally profile. Search and
deletion enforce the same identity boundary; a record outside the active scope
is neither returned nor removable.

Use `hermes memory setup` or the Hermes dashboard to configure:

- `auto_capture`: store completed turns (default `true`);
- `auto_recall`: prefetch memories before turns (default `true`);
- `max_recall_results`: inject at most 1–20 matches (default `6`);
- `default_scope`: choose the preferred isolation boundary (default `user`).

Configuration is stored at
`$HERMES_HOME/antfly-hermes-lite/config.json` with owner-only permissions.

## Current boundary

Version 0.2 provides a production-safe lexical baseline: stable idempotent IDs,
full-text retrieval, temporal and provenance metadata, scope enforcement,
serialized background writes, and a one-file embedded database. It does not yet
extract atomic facts from every conversation or enable vector/reranked recall.
Those are the next retrieval-quality layer; the persisted schema and provider
contract are designed so they can be added without changing the Hermes-facing
integration.
