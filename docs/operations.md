# Operations

The live corpus consists of `knowledge.aflite`, its portable backup
`knowledge.afb`, and the validated `knowledge.manifest.json` sidecar. The
manifest records corpus identity, provenance, health query, and qualification
suites. These artifacts should not be edited in place. One setup/maintenance
writer and multiple proven read-only sidecars are supported. A second writer
must receive a busy error.

Before enabling a profile:

1. Build or restore the database with `antfly-hermes-setup`.
2. Run `antfly-hermes-maintain doctor` with a broad expected query.
3. Run the role evaluation suite.
4. Store the `.afb` backup separately from the live artifact.
5. Validate and enable the plugin in Hermes.

Upgrades replace plugin binaries but preserve `${PLUGIN_DATA}`. Disable and
uninstall operations must not delete `.aflite`, `.afb`, or the manifest. Restore
always targets a new filename, which prevents partial replacement of the live
artifact. After validation, promote the database, backup, and matching manifest
as one stopped-reader operation. Never infer or synthesize a manifest for an
existing database whose corpus identity is unknown.

Supported package targets under qualification are macOS arm64, Linux amd64, and
Linux arm64. Every package contains its native library, build metadata, source
provenance, file checksums, and archive checksum. The CI release matrix builds
and exercises each target natively.
