# Troubleshooting

`Library not loaded` or `libantfly.so: cannot open shared object file` means the
plugin archive does not match the host or was extracted incompletely. Re-verify
the archive checksum and ensure the native library is next to the connector
binaries in `bin/`.

`ABI mismatch` means the Go binding and native library came from different
Antfly revisions. Reinstall one complete release archive; do not mix files from
different versions.

`database busy` for a writer is expected when another writer owns the artifact.
Stop setup/maintenance writers and retry. Read-only MCP sidecars may remain open
in the combinations covered by the concurrency test.

`doctor` returns nonzero when any production gate fails. Inspect its JSON
fields and warnings. Common remediations are tightening file permissions to
`0600`, creating a fresh `.afb`, or selecting a health query that should match
approved evidence.

A zero-hit answer is not a reason to bypass the connector with filesystem or
terminal search. Confirm audience, visibility, lifecycle dates, and the setup
skip reasons. If the source is intentionally excluded, use a separately
authorized artifact rather than broadening the existing agent profile.
