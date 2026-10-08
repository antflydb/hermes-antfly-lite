# Hermes catalog distribution

`antfly-hermes-lite/` is the immutable, universal plugin directory intended
for the Hermes plugin catalog. It contains checksummed runtime payloads for
macOS arm64, Linux amd64, and Linux arm64. The local launcher selects and
extracts only the current platform into the profile's plugin-data directory;
it performs no network requests and never replaces its bundled payloads.

The catalog entry must pin the repository commit containing this directory and
use `subdir: catalog/antfly-hermes-lite`. A catalog release is assembled only
from artifacts that passed the connector's cross-platform CI at the
`payload_source_commit` recorded in `payloads/manifest.json`.
