# Security model

The connector is a retrieval component, not a host sandbox. Hermes profiles
separate configuration and plugin data, but processes running as the same OS
user may still be able to read each other's files.

Controls implemented by the connector:

- the MCP sidecar opens the live database read-only;
- setup is the only baseline ingestion writer;
- lifecycle, audience, visibility, date, URL, duplicate-ID, and size policy is
  enforced before a document is written;
- `.aflite` and `.afb` files are forced to owner-only `0600` permissions;
- `doctor` fails readiness for broader permissions or a stale backup;
- model-visible tools are read-only and exclude setup, backup, restore, and
  maintenance operations;
- evidence is normalized and retrieved text is explicitly treated as inert;
- prompt-injection fixtures must remain labeled evidence and never become tool
  instructions;
- logs and diagnostics contain metadata, not document bodies or credentials.

Use separate artifacts and, for sensitive HR or regulated data, separate OS
users or containers. Do not place action credentials in the plugin data
directory. A restricted corpus must be built explicitly with
`--max-visibility restricted` and independently reviewed.

Threats still requiring deployment-specific controls include a compromised host
user, malicious local binary replacement, unsafe upstream document extraction,
and leakage through a channel whose participants cannot open the cited source.
Release archives include SHA-256 checksums; production distribution should also
add signed release provenance.
