#!/usr/bin/env python3
"""Fail if an API error code has no human string (would show as ERR_… in the UI).

Collects codes from server Go (WriteR / quoted ERR_* / CONFLICT_* / apperror constants)
and from server/locales (except notifications.*). Each code must exist in:

- server/locales/{ru,en}.json with a value other than the code
- web and android UI catalogs as errors.{code}, value other than the code / errors.{code}

Dynamic UI lookup: formatApiError uses errors.${err.code}; missing keys fall back to
the server message, which is the code when the locale is missing.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
SERVER = REPO / "server"
LOCALES = SERVER / "locales"
UI_CATALOGS = (
    REPO / "web" / "src" / "lib" / "i18n",
    REPO / "android" / "ui" / "src" / "lib" / "i18n",
)

QUOTED_CODE = re.compile(r'"(ERR_[A-Z0-9_]+|CONFLICT_[A-Z0-9_]+)"')
CONST_DEF = re.compile(r'^\s+[A-Z][A-Za-z0-9]+\s+=\s+"([A-Z][A-Z0-9_]+)"', re.M)


def load_json(path: Path) -> dict[str, str]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise SystemExit(f"{path}: expected JSON object")
    return {str(k): str(v) for k, v in data.items()}


def go_error_codes() -> set[str]:
    codes: set[str] = set()
    apperror = SERVER / "internal" / "apperror" / "errors.go"
    codes.update(CONST_DEF.findall(apperror.read_text(encoding="utf-8")))
    for path in SERVER.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        codes.update(QUOTED_CODE.findall(text))
    return codes


def server_error_keys(catalog: dict[str, str]) -> set[str]:
    return {k for k in catalog if not k.startswith("notifications.")}


def is_raw(value: str, code: str) -> bool:
    v = value.strip()
    return not v or v == code or v == f"errors.{code}"


def main() -> int:
    ru = load_json(LOCALES / "ru.json")
    en = load_json(LOCALES / "en.json")
    universe = go_error_codes() | server_error_keys(ru) | server_error_keys(en)

    missing: list[str] = []
    for code in sorted(universe):
        for lang, cat in (("ru", ru), ("en", en)):
            if code not in cat:
                missing.append(f"server/locales/{lang}.json: {code}")
            elif is_raw(cat[code], code):
                missing.append(f"server/locales/{lang}.json: {code} is the raw code")
        for i18n_dir in UI_CATALOGS:
            key = f"errors.{code}"
            rel = i18n_dir.relative_to(REPO)
            for lang in ("ru", "en"):
                cat = load_json(i18n_dir / f"{lang}.json")
                if key not in cat:
                    missing.append(f"{rel}/{lang}.json: {key}")
                elif is_raw(cat[key], code):
                    missing.append(f"{rel}/{lang}.json: {key} is the raw code")

    if missing:
        print("api-error-i18n: codes without a human message:\n", file=sys.stderr)
        for item in missing:
            print(f"  {item}", file=sys.stderr)
        print(
            "\nAdd the string to server/locales/{ru,en}.json and errors.{CODE} "
            "in web and android/ui i18n catalogs.",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
