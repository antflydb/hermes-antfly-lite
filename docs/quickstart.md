# Quickstart

Use the release archive matching the host:

- `darwin-arm64` for Apple Silicon macOS;
- `linux-amd64` for x86-64 Linux;
- `linux-arm64` for arm64 Linux.

Verify the adjacent `.sha256` file before extraction. Extract the archive into
the active Hermes plugin directory as `hermes-antfly`, validate it, and
enable it:

```bash
cd ~/.hermes/plugins
tar -xzf ~/Downloads/hermes-antfly-0.3.0-PLATFORM.tar.gz
mv hermes-antfly-0.3.0-PLATFORM hermes-antfly
hermes plugins validate ~/.hermes/plugins/hermes-antfly
hermes plugins enable hermes-antfly
hermes config set memory.provider hermes-antfly
hermes memory status
```

The connector stores data under Hermes' stable `${PLUGIN_DATA}` namespace, not
inside the replaceable plugin directory. The included pilot scripts discover
that namespace from the selected Hermes profile; no installation-specific hash
is hardcoded. For a manual setup, obtain the resolved path from Hermes'
plugin/MCP diagnostics, then create the first Support corpus and its manifest:

```bash
~/.hermes/plugins/hermes-antfly/bin/antfly-hermes-setup \
  --db PATH_TO_PLUGIN_DATA/knowledge.aflite \
  --input ./support-documents.jsonl \
  --backup PATH_TO_PLUGIN_DATA/knowledge.afb \
  --manifest ./support-corpus-manifest.json \
  --audience support \
  --max-visibility internal

~/.hermes/plugins/hermes-antfly/bin/antfly-hermes-maintain doctor \
  --db PATH_TO_PLUGIN_DATA/knowledge.aflite \
  --backup PATH_TO_PLUGIN_DATA/knowledge.afb \
  --query "a broad phrase expected in the corpus"
```

Restart the Hermes session after enabling the plugin. Ask a known-answer Support
question and confirm that the response cites an approved source URL.

Antfly long-term memory is now active for the profile using its default embedded
Lite backend. It automatically
recalls relevant memories before a turn, records completed primary-agent turns,
and mirrors curated `MEMORY.md` and `USER.md` writes. See
[Memory provider](memory-provider.md) for scope, configuration, and limitations.

The manifest is installed as `knowledge.manifest.json` beside the database.
`knowledge_status` reports its corpus identity and provenance, while the pilot
launcher uses its health query and plugin-relative evaluation suites.

The Git repository is the source distribution. Native runtime binaries are
delivered through platform-specific release archives; cloning the repository
directly into the plugin directory is not an installation method.
