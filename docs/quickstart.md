# Quickstart

Use the release archive matching the host:

- `darwin-arm64` for Apple Silicon macOS;
- `linux-amd64` for x86-64 Linux;
- `linux-arm64` for arm64 Linux.

Verify the adjacent `.sha256` file before extraction. Extract the archive into
the active Hermes plugin directory as `antfly-hermes-lite`, validate it, and
enable it:

```bash
cd ~/.hermes/plugins
tar -xzf ~/Downloads/antfly-hermes-lite-0.0.1-PLATFORM.tar.gz
mv antfly-hermes-lite-0.0.1-PLATFORM antfly-hermes-lite
hermes plugins validate ~/.hermes/plugins/antfly-hermes-lite
hermes plugins enable antfly-hermes-lite
```

The connector stores data under Hermes' stable `${PLUGIN_DATA}` namespace, not
inside the replaceable plugin directory. Obtain the resolved path from Hermes'
plugin/MCP diagnostics, then create the first Support corpus:

```bash
~/.hermes/plugins/antfly-hermes-lite/bin/antfly-hermes-setup \
  --db PATH_TO_PLUGIN_DATA/knowledge.aflite \
  --input ./support-documents.jsonl \
  --backup PATH_TO_PLUGIN_DATA/knowledge.afb \
  --audience support \
  --max-visibility internal

~/.hermes/plugins/antfly-hermes-lite/bin/antfly-hermes-maintain doctor \
  --db PATH_TO_PLUGIN_DATA/knowledge.aflite \
  --backup PATH_TO_PLUGIN_DATA/knowledge.afb \
  --query "a broad phrase expected in the corpus"
```

Restart the Hermes session after enabling the plugin. Ask a known-answer Support
question and confirm that the response cites an approved source URL.

The Git repository is the source distribution. Native runtime binaries are
delivered through platform-specific release archives; cloning the repository
directly into the plugin directory is not an installation method.
