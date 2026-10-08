#!/usr/bin/env python3
"""Launch a pinned Antfly Lite binary from the universal Hermes catalog bundle."""

from __future__ import annotations

import fcntl
import hashlib
import json
import os
import platform
import shutil
import sys
import tarfile
import tempfile
from pathlib import Path


TARGETS = {
    ("Darwin", "arm64"): "darwin-arm64",
    ("Linux", "x86_64"): "linux-amd64",
    ("Linux", "aarch64"): "linux-arm64",
    ("Linux", "arm64"): "linux-arm64",
}


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _safe_extract(archive: Path, destination: Path) -> Path:
    with tarfile.open(archive, "r:gz") as package:
        members = package.getmembers()
        for member in members:
            relative = Path(member.name)
            if relative.is_absolute() or ".." in relative.parts:
                raise RuntimeError(f"unsafe path in runtime payload: {member.name}")
            if not (member.isdir() or member.isfile()):
                raise RuntimeError(f"unsupported entry in runtime payload: {member.name}")
            resolved = (destination / relative).resolve()
            if destination.resolve() not in (resolved, *resolved.parents):
                raise RuntimeError(f"runtime payload escapes extraction root: {member.name}")
        package.extractall(destination, members=members)
    roots = [item for item in destination.iterdir() if item.is_dir()]
    if len(roots) != 1:
        raise RuntimeError("runtime payload must contain exactly one package directory")
    return roots[0]


def _runtime_root(bundle: Path, target: str, version: str, expected: str) -> Path:
    configured = os.environ.get("ANTFLY_RUNTIME_DIR", "").strip()
    if not configured:
        raise RuntimeError("ANTFLY_RUNTIME_DIR is required for the catalog runtime")
    cache = Path(configured).expanduser().resolve()
    cache.mkdir(mode=0o700, parents=True, exist_ok=True)
    destination = cache / f"{version}-{target}"
    marker = destination / ".payload-sha256"
    if marker.is_file() and marker.read_text(encoding="utf-8").strip() == expected:
        return destination

    lock_path = cache / f".{version}-{target}.lock"
    with lock_path.open("a+b") as lock:
        os.chmod(lock_path, 0o600)
        fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
        if marker.is_file() and marker.read_text(encoding="utf-8").strip() == expected:
            return destination
        archive = bundle / "payloads" / f"antfly-hermes-lite-{version}-{target}.tar.gz"
        if not archive.is_file() or _sha256(archive) != expected:
            raise RuntimeError(f"Antfly catalog runtime checksum failed for {target}")
        staging = Path(tempfile.mkdtemp(prefix=f".{version}-{target}.", dir=cache))
        try:
            extracted = _safe_extract(archive, staging)
            if destination.exists():
                raise RuntimeError(f"incomplete Antfly runtime already exists: {destination}")
            extracted_marker = extracted / ".payload-sha256"
            extracted_marker.write_text(expected + "\n", encoding="utf-8")
            os.chmod(extracted_marker, 0o600)
            os.replace(extracted, destination)
        finally:
            shutil.rmtree(staging, ignore_errors=True)
    return destination


def main() -> int:
    bundle = Path(__file__).resolve().parent
    metadata = json.loads((bundle / "payloads" / "manifest.json").read_text(encoding="utf-8"))
    target = TARGETS.get((platform.system(), platform.machine()))
    if target is None or target not in metadata["payloads"]:
        supported = ", ".join(sorted(metadata["payloads"]))
        raise RuntimeError(
            f"unsupported Antfly Lite platform {platform.system()}/{platform.machine()}; "
            f"bundle supports {supported}"
        )
    command = Path(sys.argv[0]).name
    if command == "runtime.py":
        raise RuntimeError("invoke the catalog runtime through a command in bin/")
    payload = metadata["payloads"][target]
    runtime = _runtime_root(bundle, target, metadata["version"], payload["sha256"])
    executable = runtime / "bin" / command
    if not executable.is_file() or not os.access(executable, os.X_OK):
        raise RuntimeError(f"runtime payload does not provide {command}")
    env = dict(os.environ)
    library_path = str(runtime / "bin")
    variable = "DYLD_LIBRARY_PATH" if platform.system() == "Darwin" else "LD_LIBRARY_PATH"
    env[variable] = library_path + (os.pathsep + env[variable] if env.get(variable) else "")
    os.execve(executable, [str(executable), *sys.argv[1:]], env)
    return 127


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"antfly-hermes-lite runtime error: {exc}", file=sys.stderr)
        raise SystemExit(1)
