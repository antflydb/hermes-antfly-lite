# Source and binary provenance

This connector embeds Antfly Lite through its public Go/C ABI binding.

- Antfly source repository: `https://github.com/antflydb/antfly`
- Antfly release source commit: `e5261e9e2937d03943b518974bf8063351ad1449`
- Antfly release tag: `v0.2.1`
- Go module: `github.com/antflydb/antfly/go/pkg/antflylite`
- Go module version: `v0.0.0-20260909002942-e5261e9e2937`
- Native library build: `zig build capi -Doptimize=ReleaseFast`

Each release archive contains `build-metadata.json`, an internal `SHA256SUMS`,
and a separate archive checksum. The metadata records the target platform and
the exact native-library digest shipped in that archive.
