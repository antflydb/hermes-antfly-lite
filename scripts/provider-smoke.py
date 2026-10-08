#!/usr/bin/env python3
"""Exercise the packaged provider lifecycle without model or network access."""

from __future__ import annotations

import argparse
import importlib.util
import sys
import tempfile
import threading
import types
from dataclasses import dataclass
from pathlib import Path


@dataclass
class RecallStatus:
    provider_label: str
    count: int
    glyph: str


class MemoryProvider:
    pass


def spawn_context_thread(target, *, args=(), name=None):
    return threading.Thread(target=target, args=args, name=name, daemon=True)


def load_provider(plugin_root: Path):
    agent = types.ModuleType("agent")
    memory_provider = types.ModuleType("agent.memory_provider")
    memory_provider.PRE_COMPRESS_CHECKPOINT_API_VERSION = 2
    memory_provider.MemoryProvider = MemoryProvider
    memory_provider.RecallStatus = RecallStatus
    memory_provider.spawn_context_thread = spawn_context_thread
    sys.modules["agent"] = agent
    sys.modules["agent.memory_provider"] = memory_provider
    spec = importlib.util.spec_from_file_location("antfly_provider_smoke", plugin_root / "__init__.py")
    if spec is None or spec.loader is None:
        raise RuntimeError("could not load provider")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.AntflyLiteMemoryProvider


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--plugin-root", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()
    provider_class = load_provider(args.plugin_root.resolve())
    with tempfile.TemporaryDirectory(prefix="antfly-provider-smoke.") as raw_home:
        home = Path(raw_home)
        first = provider_class()
        first.initialize(
            "session-a", hermes_home=str(home), agent_identity="support",
            user_id="alice", cwd="/support", agent_context="primary",
        )
        remembered = first.handle_tool_call("antfly_memory", {
            "action": "remember", "text": "The escalation codename is cobalt kestrel",
            "scope": "user", "importance": 0.9,
        })
        assert "cobalt kestrel" in remembered
        first.sync_turn(
            "My maintenance window is Sunday at 03:00 UTC.",
            "I will remember the maintenance window.", session_id="session-a",
        )
        first.shutdown()

        second = provider_class()
        second.initialize(
            "session-b", hermes_home=str(home), agent_identity="support",
            user_id="alice", cwd="/support", agent_context="primary",
        )
        assert "cobalt kestrel" in second.prefetch("What is the escalation codename?")
        assert "Sunday at 03:00 UTC" in second.prefetch("maintenance window")
        second.on_memory_write("add", "user", "Prefers concise pilot reports", {})
        second._drain(timeout=5)
        second.on_memory_write(
            "replace", "user", "Prefers concise release reports",
            {"previous_content": "Prefers concise pilot reports"},
        )
        second._drain(timeout=5)
        assert not second.prefetch("pilot reports")
        assert "release reports" in second.prefetch("release reports")
        second.on_memory_write(
            "remove", "user", "", {"previous_content": "Prefers concise release reports"},
        )
        second.shutdown()

        third = provider_class()
        third.initialize(
            "session-c", hermes_home=str(home), agent_identity="support",
            user_id="alice", cwd="/support", agent_context="primary",
        )
        assert not third.prefetch("release reports")
        assert not third.prefetch("escalation codename", session_id="session-c") == ""
        third.shutdown()
    print("Antfly Hermes memory provider smoke passed")


if __name__ == "__main__":
    main()
