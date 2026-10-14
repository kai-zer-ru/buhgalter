#!/usr/bin/env python3
"""Fail if UI code uses a literal i18n key missing from ru.json or en.json.

Surfaces: web/src and android/ui/src. Only static $_('…') / tr('…') literals;
dynamic/computed keys are out of scope.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]

# $_('key') / $_("key") / tr('key') — not obj.tr( or similar.
KEY_LIT = re.compile(
    r"""(?:\$_\(|(?<![.\w])tr\()\s*['"]([a-zA-Z0-9_.]+)['"]"""
)

SURFACES: list[tuple[str, Path, Path]] = [
    ("web", REPO / "web" / "src", REPO / "web" / "src" / "lib" / "i18n"),
    (
        "android",
        REPO / "android" / "ui" / "src",
        REPO / "android" / "ui" / "src" / "lib" / "i18n",
    ),
]

SCAN_SUFFIXES = {".svelte", ".ts"}


def load_catalog(path: Path) -> set[str]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise SystemExit(f"{path}: expected flat JSON object")
    return set(data)


def extract_keys(src_root: Path) -> set[str]:
    keys: set[str] = set()
    for path in sorted(src_root.rglob("*")):
        if path.suffix not in SCAN_SUFFIXES:
            continue
        if "node_modules" in path.parts:
            continue
        text = path.read_text(encoding="utf-8")
        keys.update(KEY_LIT.findall(text))
    return keys


def main() -> int:
    missing: list[str] = []
    for name, src_root, i18n_dir in SURFACES:
        if not src_root.is_dir():
            print(f"ui-i18n-keys: missing source tree {src_root}", file=sys.stderr)
            return 1
        if not i18n_dir.is_dir():
            print(f"ui-i18n-keys: missing i18n dir {i18n_dir}", file=sys.stderr)
            return 1
        used = extract_keys(src_root)
        ru = load_catalog(i18n_dir / "ru.json")
        en = load_catalog(i18n_dir / "en.json")
        for key in sorted(used):
            if key not in ru:
                missing.append(f"{name} ru: {key}")
            if key not in en:
                missing.append(f"{name} en: {key}")

    if missing:
        print(
            "ui-i18n-keys: literal UI keys missing from catalogs:\n",
            file=sys.stderr,
        )
        for item in missing:
            print(f"  {item}", file=sys.stderr)
        print(
            "\nAdd each listed key to the named locale file for that surface.",
            file=sys.stderr,
        )
        return 1

    print("ui-i18n-keys check OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
