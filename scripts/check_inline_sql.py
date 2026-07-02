#!/usr/bin/env python3
"""Fail if production Go code contains inline SQL (see docs/sql-access.md)."""

from __future__ import annotations

import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
SERVER = REPO / "server"

SKIP_DIR_PARTS = {"internal/db/sqlc"}

# Paths relative to server/ — every inline SQL literal in the file must match one pattern.
FILE_EXCEPTIONS: dict[str, list[re.Pattern[str]]] = {
    "internal/db/manager.go": [
        re.compile(r"VACUUM", re.I),
        re.compile(r"PRAGMA", re.I),
    ],
    "internal/accountbalance/hook.go": [
        re.compile(r"pragma_table_info", re.I),
    ],
    "internal/admin/handler.go": [
        re.compile(r"goose_db_version", re.I),
    ],
}

DB_CALL = re.compile(r"\.(Exec|Query|QueryRow)(Context)?\s*\(", re.MULTILINE)
RAW_STRING = re.compile(r"`([\s\S]*?)`")
SQL_KEYWORDS = re.compile(r"\b(SELECT|INSERT|UPDATE|DELETE|FROM|INTO|JOIN|WHERE)\b", re.I)


def rel(path: Path) -> str:
    return path.relative_to(SERVER).as_posix()


def is_skipped(path: Path) -> bool:
    if path.name.endswith("_test.go"):
        return True
    parts = path.relative_to(SERVER).parts
    return any(part in SKIP_DIR_PARTS for part in parts)


def allowed(rel_path: str, sql: str) -> bool:
    patterns = FILE_EXCEPTIONS.get(rel_path)
    if not patterns:
        return False
    return any(pattern.search(sql) for pattern in patterns)


def sql_after_db_call(content: str, call_start: int) -> list[tuple[int, str]]:
    """Return (line, sql) for raw string literals in a db call argument list."""
    depth = 0
    i = call_start
    n = len(content)
    while i < n and content[i] != "(":
        i += 1
    if i >= n:
        return []
    i += 1
    depth = 1
    segment_start = i
    hits: list[tuple[int, str]] = []
    while i < n and depth > 0:
        ch = content[i]
        if ch == "(":
            depth += 1
            i += 1
            continue
        if ch == ")":
            depth -= 1
            if depth == 0:
                break
            i += 1
            continue
        if ch == "`":
            end = content.find("`", i + 1)
            if end == -1:
                break
            sql = content[i + 1 : end]
            line = content[:i].count("\n") + 1
            hits.append((line, sql))
            i = end + 1
            continue
        i += 1
    return hits


def main() -> int:
    violations: list[str] = []
    for path in sorted(SERVER.rglob("*.go")):
        if is_skipped(path):
            continue
        rel_path = rel(path)
        content = path.read_text(encoding="utf-8")
        for match in DB_CALL.finditer(content):
            for line_no, sql in sql_after_db_call(content, match.start()):
                if not SQL_KEYWORDS.search(sql):
                    continue
                if allowed(rel_path, sql):
                    continue
                snippet = " ".join(sql.split())[:120]
                violations.append(f"{rel_path}:{line_no}: {snippet}")

    if violations:
        print("inline SQL check failed — use server/queries/ and make sqlc:\n", file=sys.stderr)
        for item in violations:
            print(f"  {item}", file=sys.stderr)
        print("\nAllowed exceptions: docs/sql-access.md", file=sys.stderr)
        return 1

    print("inline SQL check OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
