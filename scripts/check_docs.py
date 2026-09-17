#!/usr/bin/env python3
"""Validate the repository's AI-facing documentation contract."""

from __future__ import annotations

import os
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
USE_CASE_DIR_RE = re.compile(r"UC-\d{2}")
REQUIRED_USE_CASE_FILES = (
    "README.md",
    "specification.md",
    "realization.md",
)
LEGACY_DIRECTORIES = (
    "01_vopc_design_class_diagram",
    "02_domain_model",
    "03_database_erd",
    "04_operation_contracts",
    "05_state_machines",
    "06_traceability",
    "sequence_digrams",
    "uc03/docs",
)
LEGACY_ROOT_FILES = (
    "implementation_plan.md",
    "usecase_realization_step_1_3.md",
)
RETIRED_CODE_DIRECTORIES = (
    "uc03",
)
RETIRED_DOCUMENTATION_DIRECTORIES = (
    "docs/archive",
)
LEGACY_REFERENCE_EXEMPT_PATHS = {
    Path("AGENTS.md"),
    Path("docs/MIGRATION_PLAN.md"),
    Path("scripts/check_docs.py"),
}


def tracked_markdown() -> set[Path]:
    result = subprocess.run(
        ["git", "ls-files", "--", "*.md"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return {ROOT / line for line in result.stdout.splitlines() if line}


def tracked_files() -> list[Path]:
    result = subprocess.run(
        ["git", "ls-files", "-z"],
        cwd=ROOT,
        check=True,
        capture_output=True,
    )
    return [ROOT / raw.decode("utf-8", errors="surrogateescape") for raw in result.stdout.split(b"\0") if raw]


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


def validate_use_case_packages(errors: list[str]) -> None:
    use_case_root = ROOT / "docs/use-cases"
    use_case_index = use_case_root / "README.md"
    documentation_index = ROOT / "docs/INDEX.md"

    if not use_case_index.is_file() or not documentation_index.is_file():
        return

    use_case_index_text = use_case_index.read_text(encoding="utf-8")
    documentation_index_text = documentation_index.read_text(encoding="utf-8")

    for directory in sorted(path for path in use_case_root.glob("UC-*") if path.is_dir()):
        if not USE_CASE_DIR_RE.fullmatch(directory.name):
            errors.append(f"invalid use-case directory name: {directory.relative_to(ROOT)}")
            continue

        for filename in REQUIRED_USE_CASE_FILES:
            required = directory / filename
            if not required.is_file():
                errors.append(f"incomplete use-case package; missing: {required.relative_to(ROOT)}")

        model_target = f"{directory.name}/README.md"
        if f"({model_target})" not in use_case_index_text:
            errors.append(
                f"docs/use-cases/README.md does not register {directory.name}: {model_target}"
            )

        documentation_target = f"use-cases/{directory.name}/README.md"
        if f"({documentation_target})" not in documentation_index_text:
            errors.append(
                f"docs/INDEX.md does not route {directory.name}: {documentation_target}"
            )


def main() -> int:
    errors: list[str] = []
    ids: dict[str, Path] = {}
    files = markdown_candidates()

    for relative in LEGACY_DIRECTORIES:
        if (ROOT / relative).exists():
            errors.append(f"legacy documentation directory must not exist: {relative}")
    for relative in LEGACY_ROOT_FILES:
        if (ROOT / relative).exists():
            errors.append(f"legacy root document must not exist: {relative}")
    for relative in RETIRED_CODE_DIRECTORIES:
        if (ROOT / relative).exists():
            errors.append(f"retired code directory must not exist: {relative}")
    for relative in RETIRED_DOCUMENTATION_DIRECTORIES:
        if (ROOT / relative).exists():
            errors.append(f"retired documentation directory must not exist: {relative}")

    validate_use_case_packages(errors)

    legacy_references = (*LEGACY_DIRECTORIES, *LEGACY_ROOT_FILES)
    for path in tracked_files():
        if not path.is_file():
            continue
        rel = path.relative_to(ROOT)
        if rel in LEGACY_REFERENCE_EXEMPT_PATHS or rel.parts[:2] == ("docs", "archive"):
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        for line_number, line in enumerate(text.splitlines(), start=1):
            if "git show " in line:
                continue
            for legacy in legacy_references:
                if legacy in line:
                    errors.append(f"{rel}:{line_number}: legacy path reference outside provenance: {legacy}")

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

    require_plantuml = os.environ.get("REQUIRE_PLANTUML") == "1"
    if shutil.which("plantuml"):
        diagrams = sorted(ROOT.rglob("*.puml"))
        result = subprocess.run(["plantuml", "-checkonly", *map(str, diagrams)], cwd=ROOT)
        if result.returncode:
            errors.append("PlantUML syntax validation failed")
    elif require_plantuml:
        errors.append("PlantUML is required but the plantuml executable is not installed")
    else:
        print("NOTICE: plantuml is not installed; local diagram syntax check skipped (CI requires it)")

    if errors:
        print("Documentation validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Documentation validation passed: {len(files)} Markdown files, {len(ids)} documented artifact IDs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
