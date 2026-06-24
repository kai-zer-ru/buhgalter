#!/usr/bin/env python3
"""Generate category icon SVGs from data/category_icons.json."""

from __future__ import annotations

import json
import shutil
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
JSON_PATH = ROOT / "data" / "category_icons.json"
OFFICIAL_DIR = ROOT / "data" / "category_icons"
OUT_DIR = ROOT / "web" / "static" / "icons" / "categories"

EMOJI_TEMPLATE = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="8" fill="#3B82F6" opacity="0.15"/><text x="16" y="22" text-anchor="middle" font-size="16">{emoji}</text></svg>
"""

BRAND_TEMPLATE = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="8" fill="{bg}"/><text x="16" y="21" text-anchor="middle" font-size="{size}" font-weight="bold" fill="{fg}" font-family="system-ui,sans-serif">{label}</text></svg>
"""


def main() -> int:
    data = json.loads(JSON_PATH.read_text(encoding="utf-8"))
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    seen: set[str] = set()
    for item in data["icons"]:
        icon_id = item["id"]
        if icon_id in seen:
            raise SystemExit(f"duplicate icon id: {icon_id}")
        seen.add(icon_id)

        if item.get("official_logo"):
            src = OFFICIAL_DIR / f"{icon_id}.svg"
            if not src.is_file():
                print(
                    f"missing official logo for {icon_id}: run "
                    f"'python3 scripts/download_marketplace_logos.py'",
                    file=sys.stderr,
                )
                return 1
            shutil.copyfile(src, OUT_DIR / f"{icon_id}.svg")
        elif "brand" in item:
            b = item["brand"]
            svg = BRAND_TEMPLATE.format(
                bg=b["bg"],
                fg=b["fg"],
                label=b["label"],
                size=b.get("size", 11),
            )
            (OUT_DIR / f"{icon_id}.svg").write_text(svg, encoding="utf-8")
        else:
            svg = EMOJI_TEMPLATE.format(emoji=item["emoji"])
            (OUT_DIR / f"{icon_id}.svg").write_text(svg, encoding="utf-8")

    print(f"OK {len(seen)} icons -> {OUT_DIR}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
