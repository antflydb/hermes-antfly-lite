# Hermes Support-agent pilot

This pilot proves the complete user path: an operator builds a governed local
Support knowledge base in Antfly Lite, Hermes loads it from a portable plugin,
and the agent answers with retrieved evidence and citations. The test profile is
isolated from the operator's normal Hermes configuration.

## What the pilot tests

- approved Support and shared sources are searchable;
- internal Support material is available, while restricted, HR, expired,
  superseded, and future-effective records are excluded at ingestion;
- contradictory refund policies are both retrieved and scoped correctly;
- retrieved prompt injection remains untrusted evidence;
- supported answers cite their source;
- unsupported questions are escalated instead of invented; and
- the database remains usable after a new Hermes process starts.

The deterministic Antfly suite contains eight retrieval and policy cases. The
Hermes conversation suite contains six agent-level cases and verifies both the
`search_knowledge` tool call and the final cited answer.

## One-time setup

Install a platform release archive of this plugin in Hermes, rather than the
source checkout. Create an empty profile so the pilot does not inherit messaging
tokens, history, or unrelated skills. Until this private repository publishes
releases, download the artifact from a successful CI run, verify its outer and
inner checksums, extract it, and install that extracted directory:

```bash
hermes profile create supportpilot --no-alias --no-skills \
  --description "Isolated Support agent pilot using Hermes and Antfly Lite."
hermes -p supportpilot plugins validate /path/to/extracted-release --json
mkdir -p "$HOME/.hermes/profiles/supportpilot/plugins"
cp -R /path/to/extracted-release \
  "$HOME/.hermes/profiles/supportpilot/plugins/antfly-hermes-lite"
hermes -p supportpilot plugins enable antfly-hermes-lite \
  --no-allow-tool-override
hermes -p supportpilot pm install --extra mcp
hermes -p supportpilot plugins doctor antfly-hermes-lite --ci
```

The destination must be absent before copying. Replacing an enabled installation
in place is not a supported update procedure. The Git source checkout excludes
generated native binaries, so installing directly from the repository will not
produce a runnable connector.

Authenticate the model provider inside the isolated profile. Credentials are
not copied from another profile:

```bash
hermes -p supportpilot auth add openai-codex --type oauth
```

## Build and verify the knowledge base

From this repository checkout:

```bash
HERMES_BIN="$HOME/.local/bin/hermes" \
  scripts/support-agent-pilot.sh prepare
HERMES_BIN="$HOME/.local/bin/hermes" \
  scripts/support-agent-pilot.sh verify
```

`prepare` writes `knowledge.aflite` and its portable `knowledge.afb` backup to
the plugin's namespaced data directory with private permissions. It indexes six
of the twelve fixture records; the remaining six are intentionally rejected by
the lifecycle, audience, visibility, or time policy. `verify` requires an Antfly
health check plus 8/8 deterministic retrieval cases, full expected recall, zero
forbidden-source leakage, and valid citations.

## Try the agent

Run one Support turn:

```bash
HERMES_BIN="$HOME/.local/bin/hermes" \
  scripts/support-agent-pilot.sh ask \
  "A customer forgot their password. What should I tell them?"
```

Run all conversational acceptance cases:

```bash
HERMES_BIN="$HOME/.local/bin/hermes" \
  scripts/support-agent-pilot.sh conversations
```

The JSON report is written to
`pilot-results/support-agent-conversations.json` by default and is ignored by
Git. Set `SUPPORT_AGENT_OUTPUT` to retain it elsewhere. Each case fails unless
Hermes calls Antfly's `search_knowledge`, exits successfully, includes the
required evidence in its answer, cites the expected source URLs, and avoids the
case's forbidden claim.

The defaults target `supportpilot`, `openai-codex`, and `gpt-5.6-sol`. Override
them with `HERMES_PROFILE`, `HERMES_PROVIDER`, and `HERMES_MODEL`. The data and
plugin paths can likewise be overridden. The portable Support skill uses its
Hermes-qualified name by default; override `HERMES_SUPPORT_SKILL` only if the
plugin identity changes. Run the script without arguments for the complete
list.

## Production exit criteria

This pilot is ready to graduate when the deterministic and conversational
suites pass from a clean profile using a published, checksum-verified release
artifact; a process restart still passes an agent turn; logs contain no secrets
or restricted-source text; and the same artifact passes the repository's macOS
and Linux CI matrix. Real customer content should only be introduced after its
source owners, retention policy, access tier, redaction process, and escalation
owner are documented.
