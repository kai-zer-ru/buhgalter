#!/usr/bin/env python3
"""Download official bank logos from bank CDN/websites into data/banks/."""

from __future__ import annotations

import base64
import json
import re
import subprocess
import sys
from pathlib import Path
from urllib.parse import urljoin

ROOT = Path(__file__).resolve().parents[1]
OUT_DIRS = [
    ROOT / "data" / "banks",
    ROOT / "web" / "static" / "banks",
    ROOT / "server" / "internal" / "bank" / "data" / "banks",
]
JSON_PATH = ROOT / "data" / "banks_ru.json"
SERVER_JSON = ROOT / "server" / "internal" / "bank" / "data" / "banks_ru.json"

# Official assets from bank sites / CDNs.
# Notes:
# - open.ru is often unreachable; icon is taken from the bank's RuStore app package.
# - homecredit.ru returns 401 here; favicon is loaded from a Wayback snapshot of the official site.
# Values: one URL or an ordered list of fallbacks (first success wins).
OFFICIAL_URLS: dict[str, str | list[str]] = {
    "sberbank": "https://esa-res.online.sberbank.ru/ESA/common/r-2.15/img/apple-touch-icon.png",
    "tinkoff": "https://cdn.tbank.ru/params/common_front/resourses/icons/apple-touch-icon-180x180.png",
    "vtb": [
        "https://www.vtb.ru/favicon.ico",
        "https://www.vtb.ru/media-files/vtb.ru/shared/images/icon/favicon_512x512px.webp",
    ],
    "alfabank": [
        "https://click.alfabank.ru/static/logo.svg",
        "https://alfabank.ru/favicon.ico",
    ],
    "gazprombank": "https://cdn.gpb.ru/upload/files/bve/802/tltfdbfk6msczspsuxm277gjq3aghnsl/Logo_01_GPB.png",
    "raiffeisen": [
        "https://www.raiffeisen.ru/favicon.ico",
        "https://www.raiffeisen.ru/static/common/initial/LightHeader/Logo_retail.svg",
    ],
    "rosbank": "https://web.archive.org/web/2023id_/https://www.rosbank.ru/favicon.ico",
    "mkb": [
        "https://mkb.ru/apple-touch-icon.png",
        "https://mkb.ru/favicon.ico",
    ],
    "rshb": "https://www.rshb.ru/wcms-resources/LOGO_RSHB.svg",
    "open": [
        "https://static.rustore.ru/apk/1398463/content/ICON/ffd45b51-77b9-48df-b44c-76fcdbe2de20.png",
        "https://upload.wikimedia.org/wikipedia/commons/2/2a/Otkritie_Bank_logo.svg",
    ],
    "sovcombank": "https://sovcombank.ru/favicon.ico",
    "psb": [
        "https://www.psbank.ru/apple-touch-icon.png",
        "https://www.psbank.ru/favicon.ico",
        "https://www.psbank.ru/qpstorage/psb/images/logo.svg",
    ],
    "uralsib": "https://y-cdn.uralsib.ru/front/static/img/retail/main/retail-icon.png",
    "homecredit": "https://web.archive.org/web/2023id_/https://www.homecredit.ru/favicon.ico",
    "ozon": "https://finance.ozon.ru/favicon.ico",
    "yandex": [
        "https://yandex.ru/apple-touch-icon.png",
        "https://yastatic.net/s3/home/logos/share/share-logo-ru.png",
        "https://favicon.yandex.net/favicon/v2/https://bank.yandex.ru?size=120",
        "https://bank.yandex.ru/favicon.ico",
    ],
    "wbbank": "https://wb-bank.ru/apple-touch-icon.png",
    "otpbank": "https://www.otpbank.ru/favicon.ico",
    "atb": "https://www.atb.su/local/templates/dt_private/img/svgs/icon_logo.svg",
}

FALLBACK_PAGES: dict[str, str] = {
    "vtb": "https://www.vtb.ru/",
}

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

# Logos that need a light tile so they stay visible on dark UI.
ICON_LIGHT_BG: set[str] = {"raiffeisen", "psb", "atb", "rosbank", "alfabank", "rshb"}


def curl(url: str) -> bytes:
    proc = subprocess.run(
        ["curl", "-fsSL", "--retry", "3", "--retry-delay", "1", "-m", "45", "-A", UA, url],
        capture_output=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.decode("utf-8", "replace") or f"curl failed: {url}")
    return proc.stdout


def sniff_logo_from_page(page_url: str) -> str | None:
    html = curl(page_url).decode("utf-8", "replace")
    candidates: list[str] = []
    for pat in (r'https?://[^"\'\s>]+\.(?:svg|png|webp|ico)', r'(/[^"\'\s>]+\.(?:svg|png|webp|ico))'):
        for match in re.findall(pat, html, flags=re.I):
            u = match if match.startswith("http") else urljoin(page_url, match)
            if re.search(r"logo|brand|apple-touch|favicon", u, re.I):
                candidates.append(u)
    return candidates[0] if candidates else None


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


def with_light_bg(svg: str) -> str:
    if "#F3F4F6" in svg and 'rx="10"' in svg:
        return svg
    inner = svg.strip()
    if inner.startswith("<?xml"):
        inner = inner.split("?>", 1)[-1].strip()
    viewbox = "0 0 48 48"
    match = re.search(r'viewBox="([^"]+)"', inner)
    if match:
        viewbox = match.group(1)
    body = inner[inner.find(">") + 1 : inner.rfind("</svg>")] if inner.startswith("<svg") else inner
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 48 48">'
        '<rect width="48" height="48" rx="10" fill="#F3F4F6"/>'
        f'<svg x="4" y="4" width="40" height="40" viewBox="{viewbox}" preserveAspectRatio="xMidYMid meet">'
        f"{body}</svg></svg>"
    )


def to_svg_asset(data: bytes, url: str, bank_id: str) -> str:
    mime = mime_for(data, url)
    if mime == "image/svg+xml":
        svg = data.decode("utf-8", "replace").strip()
    else:
        encoded = base64.b64encode(data).decode("ascii")
        svg = (
            '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 48 48">'
            f'<image width="48" height="48" preserveAspectRatio="xMidYMid meet" '
            f'xlink:href="data:{mime};base64,{encoded}"/>'
            "</svg>"
        )
    if bank_id in ICON_LIGHT_BG:
        return with_light_bg(svg)
    return svg


def download_logo(bank_id: str) -> tuple[bytes, str]:
    raw = OFFICIAL_URLS[bank_id]
    urls = raw if isinstance(raw, list) else [raw]
    errors: list[str] = []
    for url in urls:
        try:
            data = curl(url)
            if not is_image_payload(data):
                raise RuntimeError("response is HTML, not an image")
            return data, url
        except Exception as exc:
            errors.append(f"{url}: {exc}")
    page = FALLBACK_PAGES.get(bank_id)
    if page:
        scraped = sniff_logo_from_page(page)
        if scraped:
            return curl(scraped), scraped
    raise RuntimeError("; ".join(errors))


def main() -> int:
    banks = json.loads(JSON_PATH.read_text(encoding="utf-8"))
    failures: list[str] = []

    for bank in banks:
        bank_id = bank["id"]
        try:
            data, source = download_logo(bank_id)
            svg = to_svg_asset(data, source, bank_id)
            filename = f"{bank_id}.svg"

            for out_dir in OUT_DIRS:
                out_dir.mkdir(parents=True, exist_ok=True)
                (out_dir / filename).write_text(svg, encoding="utf-8")

            bank["icon_path"] = f"banks/{filename}"
            print(f"OK {bank_id}: {source} ({len(data)} bytes)")
        except Exception as exc:
            failures.append(f"{bank_id}: {exc}")
            print(f"FAIL {bank_id}: {exc}", file=sys.stderr)

    JSON_PATH.write_text(json.dumps(banks, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    SERVER_JSON.write_text(JSON_PATH.read_text(encoding="utf-8"), encoding="utf-8")

    if failures:
        print(f"\n{len(failures)} failed", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
