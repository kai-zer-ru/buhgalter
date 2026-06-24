#!/usr/bin/env python3
"""Download official brand logos (marketplaces, Avito, …) into data/category_icons/."""

from __future__ import annotations

import base64
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "data" / "category_icons"

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

# Official marketplace assets (site CDN / apple-touch-icon).
OFFICIAL_URLS: dict[str, list[str]] = {
    "wildberries": [
        "https://www.wildberries.ru/apple-touch-icon.png",
        "https://favicon.yandex.net/favicon/v2/https://www.wildberries.ru?size=120",
    ],
    "ozon": [
        "https://favicon.yandex.net/favicon/v2/https://www.ozon.ru?size=120",
        "https://www.ozon.ru/favicon.ico",
    ],
    "yandex-market": [
        "https://favicon.yandex.net/favicon/v2/https://market.yandex.ru?size=120",
        "https://market.yandex.ru/favicon.ico",
    ],
    "avito": [
        "https://www.avito.ru/apple-touch-icon.png",
        "https://favicon.yandex.net/favicon/v2/https://www.avito.ru?size=120",
        "https://www.avito.ru/favicon.ico",
    ],
}


def curl(url: str) -> bytes:
    proc = subprocess.run(
        ["curl", "-fsSL", "--retry", "3", "--retry-delay", "1", "-m", "45", "-A", UA, url],
        capture_output=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.decode("utf-8", "replace") or f"curl failed: {url}")
    return proc.stdout


def is_image_payload(data: bytes) -> bool:
    if len(data) < 32:
        return False
    head = data[:512].lstrip().lower()
    if head.startswith(b"<!doctype") or head.startswith(b"<html") or b"<head>" in head[:256]:
        return False
    return True


def mime_for(data: bytes, url: str) -> str:
    if data.startswith(b"<svg") or b"<svg" in data[:256]:
        return "image/svg+xml"
    if data.startswith(b"\x89PNG"):
        return "image/png"
    if data[:4] == b"\x00\x00\x01\x00" or url.lower().endswith(".ico"):
        return "image/x-icon"
    if data.startswith(b"RIFF") and b"WEBP" in data[:16]:
        return "image/webp"
    return "application/octet-stream"


def to_category_svg(data: bytes, url: str) -> str:
    mime = mime_for(data, url)
    if mime == "image/svg+xml":
        inner = data.decode("utf-8", "replace").strip()
        if inner.startswith("<?xml"):
            inner = inner.split("?>", 1)[-1].strip()
        body = inner[inner.find(">") + 1 : inner.rfind("</svg>")] if inner.startswith("<svg") else inner
        return (
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">'
            '<defs><clipPath id="c"><rect width="32" height="32" rx="8"/></clipPath></defs>'
            '<g clip-path="url(#c)">'
            '<svg x="0" y="0" width="32" height="32" viewBox="0 0 32 32" preserveAspectRatio="xMidYMid meet">'
            f"{body}</svg></g></svg>"
        )

    encoded = base64.b64encode(data).decode("ascii")
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 32 32">'
        '<defs><clipPath id="c"><rect width="32" height="32" rx="8"/></clipPath></defs>'
        f'<image width="32" height="32" preserveAspectRatio="xMidYMid meet" clip-path="url(#c)" '
        f'xlink:href="data:{mime};base64,{encoded}"/>'
        "</svg>"
    )


def download_logo(icon_id: str) -> tuple[bytes, str]:
    errors: list[str] = []
    for url in OFFICIAL_URLS[icon_id]:
        try:
            data = curl(url)
            if not is_image_payload(data):
                raise RuntimeError("response is HTML, not an image")
            return data, url
        except Exception as exc:
            errors.append(f"{url}: {exc}")
    raise RuntimeError("; ".join(errors))


def main() -> int:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    failures: list[str] = []

    for icon_id in OFFICIAL_URLS:
        try:
            data, source = download_logo(icon_id)
            svg = to_category_svg(data, source)
            filename = f"{icon_id}.svg"
            (OUT_DIR / filename).write_text(svg, encoding="utf-8")
            print(f"OK {icon_id}: {source} ({len(data)} bytes)")
        except Exception as exc:
            failures.append(f"{icon_id}: {exc}")
            print(f"FAIL {icon_id}: {exc}", file=sys.stderr)

    if failures:
        print(f"\n{len(failures)} failed", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
