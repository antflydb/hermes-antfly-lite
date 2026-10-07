# Operations

The live artifact is `knowledge.aflite`; the portable backup is
`knowledge.afb`. They are different formats and neither should be edited in
place. One setup/maintenance writer and multiple proven read-only sidecars are
supported. A second writer must receive a busy error.

Before enabling a profile:

1. Build or restore the database with `antfly-hermes-setup`.
2. Run `antfly-hermes-maintain doctor` with a broad expected query.
3. Run the role evaluation suite.
4. Store the `.afb` backup separately from the live artifact.
5. Validate and enable the plugin in Hermes.

Upgrades replace plugin binaries but preserve `${PLUGIN_DATA}`. Disable and
uninstall operations must not delete `.aflite` or `.afb`. Restore always targets
a new filename, which prevents partial replacement of the live artifact. After
validation, switch the configured artifact or replace it only while writers and
readers are stopped.

Supported package targets under qualification are macOS arm64, Linux amd64, and
Linux arm64. Every package contains its native library, build metadata, source
provenance, file checksums, and archive checksum. The CI release matrix builds
and exercises each target natively.
