# Contributing

Changes must preserve the narrow read-only MCP contract and keep administrative
operations outside model-visible tools.

Before opening a pull request:

```bash
scripts/build-local.sh
scripts/validate.sh
scripts/smoke.sh
scripts/concurrency-smoke.sh
scripts/failure-smoke.sh
scripts/doctor-smoke.sh
scripts/support-eval.sh
scripts/package-release.sh
scripts/package-smoke.sh
```

Use governed synthetic fixtures; never commit customer content, credentials,
live `.aflite` files, `.afb` backups, or generated release archives. Any change
to lifecycle, visibility, citation, or retrieval behavior must add an evaluation
case. Native dependency changes must update `SOURCE_PROVENANCE.md`, Go checksums,
the CI source revision, and release notes together.
