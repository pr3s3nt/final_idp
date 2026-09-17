#!/usr/bin/env python3
"""Validate the repository's AI-facing documentation contract."""

from __future__ import annotations

import re
import shutil
import subprocess
import sys
from pathlib import Path
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parents[1]
ALLOWED_STATUS = {
    "draft",
    "current",
    "superseded",
    "deferred",
    "historical",
    "evidence",
}
LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)]+)\)")


def tracked_markdown() -> set[Path]:
    result = subprocess.run(
        ["git", "ls-files", "--", "*.md"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return {ROOT / line for line in result.stdout.splitlines() if line}


def markdown_candidates() -> list[Path]:
    files = {path for path in tracked_markdown() if path.exists()}
    files.update((ROOT / "docs").rglob("*.md"))
    files.update(path for path in (ROOT / "README.md", ROOT / "AGENTS.md") if path.exists())
    return sorted(files)


def parse_front_matter(path: Path, text: str) -> tuple[dict[str, str], str | None]:
    if not text.startswith("---\n"):
        return {}, "missing YAML front matter"
    end = text.find("\n---\n", 4)
    if end < 0:
        return {}, "front matter has no closing delimiter"
    values: dict[str, str] = {}
    for line in text[4:end].splitlines():
        if not line or line[0].isspace() or ":" not in line:
            continue
        key, value = line.split(":", 1)
        values[key.strip()] = value.strip().strip('"\'')
    missing = [key for key in ("id", "artifact", "status") if not values.get(key)]
    if missing:
        return values, f"front matter is missing: {', '.join(missing)}"
    if values["status"] not in ALLOWED_STATUS:
        return values, f"invalid status: {values['status']}"
    return values, None


def normalized_link(raw: str) -> str:
    raw = raw.strip()
    if raw.startswith("<") and raw.endswith(">"):
        return raw[1:-1]
    return raw


def main() -> int:
    errors: list[str] = []
    ids: dict[str, Path] = {}
    files = markdown_candidates()

    for path in files:
        rel = path.relative_to(ROOT)
        text = path.read_text(encoding="utf-8")

        if text.startswith("---\n") or (rel.parts and rel.parts[0] == "docs"):
            metadata, error = parse_front_matter(path, text)
            if error:
                errors.append(f"{rel}: {error}")
            elif metadata["id"] in ids:
                errors.append(f"{rel}: duplicate id {metadata['id']} (also in {ids[metadata['id']].relative_to(ROOT)})")
            else:
                ids[metadata["id"]] = path

        for match in LINK_RE.finditer(text):
            raw = normalized_link(match.group(1))
            if raw.startswith(("http://", "https://", "mailto:", "#")):
                continue
            target_text = unquote(raw.split("#", 1)[0])
            if not target_text:
                continue
            line = text[: match.start()].count("\n") + 1
            if target_text.startswith("/"):
                errors.append(f"{rel}:{line}: absolute local link is not portable: {raw}")
                continue
            if not (path.parent / target_text).resolve().exists():
                errors.append(f"{rel}:{line}: missing link target: {raw}")

    if shutil.which("plantuml"):
        diagrams = sorted(ROOT.rglob("*.puml"))
        result = subprocess.run(["plantuml", "-checkonly", *map(str, diagrams)], cwd=ROOT)
        if result.returncode:
            errors.append("PlantUML syntax validation failed")
    else:
        print("NOTICE: plantuml is not installed; diagram syntax check skipped")

    if errors:
        print("Documentation validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Documentation validation passed: {len(files)} Markdown files, {len(ids)} documented artifact IDs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
