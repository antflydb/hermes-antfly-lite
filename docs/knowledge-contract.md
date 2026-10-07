# Knowledge contract

The connector builds a role-scoped Antfly Lite artifact. Authorization happens
during ingestion; model prompts are not used as access control.

Every JSONL record must contain:

| Field | Contract |
| --- | --- |
| `id` | Stable source/chunk identifier, at most 512 bytes |
| `title` | Evidence title, at most 300 bytes |
| `text` | Explanatory evidence chunk, at most 16 KiB |
| `source_url` | Absolute HTTPS URL approved for the target audience |
| `audience` | `shared`, `support`, `research`, `sales`, `marketing`, or `hr` |
| `visibility` | `public`, `internal`, or `restricted` |
| `state` | `approved`, `draft`, `expired`, or `superseded` |
| `updated_at` | RFC3339 source update time |

Optional `effective_at` and `expires_at` values are RFC3339 timestamps.
`risk_labels` is an array of labels such as `prompt_injection` and
`untrusted_content`. Additional source metadata is preserved in storage but is
not automatically exposed by the normalized evidence contract.

Setup fails on malformed or duplicate records. It writes only records that are:

- `approved`;
- for the selected audience or `shared`;
- at or below the selected maximum visibility;
- already effective and not expired.

Search returns normalized objects containing source identity, lifecycle and
scope metadata, score, risk labels, and a citation object. Backend fields such
as encoded IDs and serialized internal documents are not part of the public MCP
contract. Source lookup fails closed if a stored record is not approved.

The current release mode is explicitly `full_text`. Antfly Lite advertises
caller-supplied dense and hybrid capabilities, but this connector does not claim
hybrid operation until a query-embedding producer with a pinned model identity
is configured and evaluated. Different vector spaces must never share an index.
