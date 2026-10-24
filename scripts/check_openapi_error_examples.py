#!/usr/bin/env python3
"""Fail if an ApiError response has no explicit example, or a local $ref does not resolve.

Redoc 2.4.0 aborts the whole /docs page on an unresolved schema token.
"""

from __future__ import annotations

import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("check_openapi_error_examples: install PyYAML", file=sys.stderr)
    sys.exit(2)


def _unescape(token: str) -> str:
    return token.replace("~1", "/").replace("~0", "~")


def _collect_refs(node: object, found: list[str]) -> None:
    if isinstance(node, dict):
        ref = node.get("$ref")
        if isinstance(ref, str) and ref.startswith("#/"):
            found.append(ref)
        for value in node.values():
            _collect_refs(value, found)
    elif isinstance(node, list):
        for value in node:
            _collect_refs(value, found)


def _resolves(doc: object, ref: str) -> bool:
    cur: object = doc
    for raw in ref[2:].split("/"):
        token = _unescape(raw)
        if isinstance(cur, dict) and token in cur:
            cur = cur[token]
        else:
            return False
    return True


def main() -> int:
    path = Path(sys.argv[1] if len(sys.argv) > 1 else "docs/api/openapi.yaml")
    doc = yaml.safe_load(path.read_text(encoding="utf-8"))
    paths = doc.get("paths") or {}
    missing: list[str] = []

    for route, methods in paths.items():
        if not isinstance(methods, dict):
            continue
        for method, op in methods.items():
            if method.startswith("x-") or not isinstance(op, dict):
                continue
            for status, resp in (op.get("responses") or {}).items():
                if not isinstance(resp, dict):
                    continue
                content = resp.get("content") or {}
                app_json = content.get("application/json") or {}
                schema = app_json.get("schema") or {}
                ref = schema.get("$ref", "")
                if not ref.endswith("/ApiError"):
                    continue
                if "example" not in app_json and "examples" not in app_json:
                    missing.append(f"{method.upper()} {route} -> {status}")

    failed = False
    if missing:
        failed = True
        print(f"openapi: ApiError responses without example ({path}):", file=sys.stderr)
        for line in missing:
            print(f"  - {line}", file=sys.stderr)

    found_refs: list[str] = []
    _collect_refs(doc, found_refs)
    dangling = sorted({ref for ref in found_refs if not _resolves(doc, ref)})
    if dangling:
        failed = True
        print(f"openapi: unresolved $ref ({path}):", file=sys.stderr)
        for ref in dangling:
            print(f"  - {ref}", file=sys.stderr)

    if failed:
        return 1

    checked = sum(
        1
        for methods in (doc.get("paths") or {}).values()
        if isinstance(methods, dict)
        for op in methods.values()
        if isinstance(op, dict)
        for resp in (op.get("responses") or {}).values()
        if isinstance(resp, dict)
        and ((resp.get("content") or {}).get("application/json") or {})
        .get("schema", {})
        .get("$ref", "")
        .endswith("/ApiError")
    )
    print(
        f"openapi: {checked} ApiError response(s) with examples OK; "
        f"{len(found_refs)} local $ref(s) resolve ({path})"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
