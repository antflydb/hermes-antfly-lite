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
completed primary-agent turns asynchronously, extracts conservative standalone
facts from short user declarations, mirrors successful built-in `MEMORY.md` and
`USER.md` add/replace/remove operations, and commits a durable checkpoint before
Hermes compresses context. Instruction-like and URL-bearing statements are not
automatically promoted to facts. Child and delegated agents can read scoped
context, but only the primary agent writes automatically.

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
- `auto_extract_facts`: separately retain safe declarative user facts (default
  `true`);
- `semantic_recall`: enable Antfly Lite hybrid lexical/vector retrieval
  (default `false`);
- `embedding_url`: an OpenAI-compatible Antfly Inference embeddings endpoint;
- `embedding_model`: the embedding model identifier;
- `embedding_space`: stable vector-space identity, defaulting to the model;
- `embedding_timeout_seconds`: bounded inference timeout before lexical
  fallback (default `3`).

Configuration is stored at
`$HERMES_HOME/antfly-hermes-lite/config.json` with owner-only permissions.

## Optional semantic recall

Lexical recall remains the zero-configuration default. To enable semantic
recall, run `hermes memory setup` or set these fields in the dashboard:

```json
{
  "semantic_recall": true,
  "embedding_url": "http://127.0.0.1:8082/ai/v1/embeddings",
  "embedding_model": "bge-base-en-v1.5",
  "embedding_space": "support-memory-v1",
  "embedding_timeout_seconds": 3
}
```

The provider supplies embeddings to Antfly Lite, which owns the dense-vector
index and native reciprocal-rank-fusion query. A configured endpoint failure is
fail-open: memory writes and recall continue lexically. Remote endpoints can use
the optional `ANTFLY_INFERENCE_API_KEY` profile secret. The response may use the
OpenAI `data[0].embedding` shape or an `embeddings[0]` array.

Use one embedding model/vector space for the lifetime of a memory database.
The provider pins its identity in `antfly-memory/embedding-space.json` and
refuses a mismatch. Changing spaces requires starting a new `memory.aflite` or
performing an explicit re-embedding migration; equal dimensions do not imply
compatible vectors. The endpoint URL may change without changing the space.

## Current boundary

Version 0.3 provides stable idempotent IDs, lexical and optional hybrid vector
retrieval, temporal and provenance metadata, scope enforcement, conservative
fact capture, serialized background writes, and a one-file embedded database.
Fact capture intentionally handles high-confidence declaration patterns rather
than asking a language model to rewrite every conversation. Graph/entity
resolution and model-assisted consolidation remain future retrieval-quality
layers.
