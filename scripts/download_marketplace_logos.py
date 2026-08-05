#!/usr/bin/env python3
"""Download official brand logos (marketplaces, subscriptions, …) into data/category_icons/."""

from __future__ import annotations

import base64
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "data" / "category_icons"

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

# Official brand assets (site CDN / apple-touch-icon / favicon service).
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
    # Subscriptions / streaming / SaaS (RU + common global).
    "yandex-plus": [
        "https://favicon.yandex.net/favicon/v2/https://plus.yandex.ru?size=120",
    ],
    "yandex-music": [
        "https://favicon.yandex.net/favicon/v2/https://music.yandex.ru?size=120",
    ],
    "kinopoisk": [
        "https://favicon.yandex.net/favicon/v2/https://www.kinopoisk.ru?size=120",
    ],
    "ivi": [
        "https://favicon.yandex.net/favicon/v2/https://www.ivi.ru?size=120",
    ],
    "okko": [
        "https://favicon.yandex.net/favicon/v2/https://okko.tv?size=120",
    ],
    "wink": [
        "https://favicon.yandex.net/favicon/v2/https://wink.ru?size=120",
    ],
    "premier": [
        "https://favicon.yandex.net/favicon/v2/https://premier.one?size=120",
    ],
    "litres": [
        "https://favicon.yandex.net/favicon/v2/https://www.litres.ru?size=120",
    ],
    "vk": [
        "https://favicon.yandex.net/favicon/v2/https://vk.com?size=120",
    ],
    "spotify": [
        "https://favicon.yandex.net/favicon/v2/https://www.spotify.com?size=120",
    ],
    "netflix": [
        "https://favicon.yandex.net/favicon/v2/https://www.netflix.com?size=120",
    ],
    "youtube": [
        "https://favicon.yandex.net/favicon/v2/https://www.youtube.com?size=120",
    ],
    "disney-plus": [
        "https://favicon.yandex.net/favicon/v2/https://www.disneyplus.com?size=120",
    ],
    "apple-music": [
        "https://favicon.yandex.net/favicon/v2/https://music.apple.com?size=120",
    ],
    "twitch": [
        "https://favicon.yandex.net/favicon/v2/https://www.twitch.tv?size=120",
    ],
    "steam": [
        "https://favicon.yandex.net/favicon/v2/https://store.steampowered.com?size=120",
    ],
    "telegram": [
        "https://favicon.yandex.net/favicon/v2/https://telegram.org?size=120",
    ],
    "discord": [
        "https://favicon.yandex.net/favicon/v2/https://discord.com?size=120",
    ],
    "chatgpt": [
        "https://favicon.yandex.net/favicon/v2/https://chatgpt.com?size=120",
    ],
    "github": [
        "https://favicon.yandex.net/favicon/v2/https://github.com?size=120",
    ],
    "notion": [
        "https://favicon.yandex.net/favicon/v2/https://www.notion.so?size=120",
    ],
    "adobe": [
        "https://favicon.yandex.net/favicon/v2/https://www.adobe.com?size=120",
    ],
    "microsoft-365": [
        "https://favicon.yandex.net/favicon/v2/https://www.microsoft365.com?size=120",
    ],
    "google-one": [
        "https://favicon.yandex.net/favicon/v2/https://one.google.com?size=120",
    ],
    # Магазины / доставка / лояльность (RU).
    "gazprom-bonus": [
        "https://favicon.yandex.net/favicon/v2/https://gazprombonus.ru?size=120",
        "https://favicon.yandex.net/favicon/v2/https://www.gazprombonus.ru?size=120",
    ],
    "pyaterochka": [
        "https://favicon.yandex.net/favicon/v2/https://5ka.ru?size=120",
    ],
    "magnit": [
        "https://favicon.yandex.net/favicon/v2/https://magnit.ru?size=120",
    ],
    "perekrestok": [
        "https://favicon.yandex.net/favicon/v2/https://www.perekrestok.ru?size=120",
    ],
    "lenta": [
        "https://favicon.yandex.net/favicon/v2/https://lenta.com?size=120",
    ],
    "auchan": [
        "https://favicon.yandex.net/favicon/v2/https://www.auchan.ru?size=120",
    ],
    "vkusvill": [
        "https://favicon.yandex.net/favicon/v2/https://vkusvill.ru?size=120",
    ],
    "dixy": [
        "https://favicon.yandex.net/favicon/v2/https://dixy.ru?size=120",
    ],
    "fix-price": [
        "https://favicon.yandex.net/favicon/v2/https://fix-price.ru?size=120",
    ],
    "chizhik": [
        "https://favicon.yandex.net/favicon/v2/https://chizhik.club?size=120",
    ],
    "verny": [
        "https://favicon.yandex.net/favicon/v2/https://www.verno-info.ru?size=120",
    ],
    "spar": [
        "https://favicon.yandex.net/favicon/v2/https://spar-online.ru?size=120",
    ],
    "globus": [
        "https://favicon.yandex.net/favicon/v2/https://www.globus.ru?size=120",
    ],
    "azbuka-vkusa": [
        "https://favicon.yandex.net/favicon/v2/https://av.ru?size=120",
    ],
    "metro-cc": [
        "https://favicon.yandex.net/favicon/v2/https://online.metro-cc.ru?size=120",
    ],
    "samokat": [
        "https://favicon.yandex.net/favicon/v2/https://samokat.ru?size=120",
    ],
    "yandex-lavka": [
        "https://favicon.yandex.net/favicon/v2/https://lavka.yandex.ru?size=120",
    ],
    "yandex-eda": [
        "https://favicon.yandex.net/favicon/v2/https://eda.yandex.ru?size=120",
    ],
    "kuper": [
        "https://favicon.yandex.net/favicon/v2/https://kuper.ru?size=120",
    ],
    "bristol": [
        "https://favicon.yandex.net/favicon/v2/https://bristol.ru?size=120",
    ],
    "krasnoe-beloe": [
        "https://favicon.yandex.net/favicon/v2/https://krasnoeibeloe.ru?size=120",
    ],
    "mvideo": [
        "https://favicon.yandex.net/favicon/v2/https://www.mvideo.ru?size=120",
    ],
    "dns": [
        "https://favicon.yandex.net/favicon/v2/https://www.dns-shop.ru?size=120",
    ],
    "eldorado": [
        "https://favicon.yandex.net/favicon/v2/https://www.eldorado.ru?size=120",
    ],
    "citilink": [
        "https://favicon.yandex.net/favicon/v2/https://www.citilink.ru?size=120",
    ],
    "detmir": [
        "https://favicon.yandex.net/favicon/v2/https://www.detmir.ru?size=120",
    ],
    "sportmaster": [
        "https://favicon.yandex.net/favicon/v2/https://www.sportmaster.ru?size=120",
    ],
    "lamoda": [
        "https://favicon.yandex.net/favicon/v2/https://www.lamoda.ru?size=120",
    ],
    "hoff": [
        "https://favicon.yandex.net/favicon/v2/https://www.hoff.ru?size=120",
    ],
    "leroy-merlin": [
        "https://favicon.yandex.net/favicon/v2/https://leroymerlin.ru?size=120",
    ],
    "petrovich": [
        "https://favicon.yandex.net/favicon/v2/https://petrovich.ru?size=120",
    ],
    "vseinstrumenti": [
        "https://favicon.yandex.net/favicon/v2/https://www.vseinstrumenti.ru?size=120",
    ],
    "goldapple": [
        "https://favicon.yandex.net/favicon/v2/https://goldapple.ru?size=120",
    ],
    "letu": [
        "https://favicon.yandex.net/favicon/v2/https://www.letu.ru?size=120",
    ],
    "rive-gauche": [
        "https://favicon.yandex.net/favicon/v2/https://rivegauche.ru?size=120",
    ],
    "apteka-ru": [
        "https://favicon.yandex.net/favicon/v2/https://apteka.ru?size=120",
    ],
    "zdravcity": [
        "https://favicon.yandex.net/favicon/v2/https://zdravcity.ru?size=120",
    ],
    "sberprime": [
        "https://favicon.yandex.net/favicon/v2/https://www.sberbank.com/sberprime?size=120",
        "https://favicon.yandex.net/favicon/v2/https://www.sberbank.ru/ru/person/sberprime?size=120",
    ],
    # Банки / банковские сервисы (подписки, комиссии).
    "tinkoff": [
        "https://cdn.tbank.ru/params/common_front/resourses/icons/apple-touch-icon-180x180.png",
        "https://favicon.yandex.net/favicon/v2/https://www.tbank.ru?size=120",
        "https://favicon.yandex.net/favicon/v2/https://www.tinkoff.ru?size=120",
    ],
    "sberbank": [
        "https://esa-res.online.sberbank.ru/ESA/common/r-2.15/img/apple-touch-icon.png",
        "https://favicon.yandex.net/favicon/v2/https://www.sberbank.ru?size=120",
    ],
    "vtb": [
        "https://favicon.yandex.net/favicon/v2/https://www.vtb.ru?size=120",
        "https://www.vtb.ru/favicon.ico",
    ],
    "alfabank": [
        "https://favicon.yandex.net/favicon/v2/https://alfabank.ru?size=120",
        "https://click.alfabank.ru/static/logo.svg",
        "https://alfabank.ru/favicon.ico",
    ],
    "gazprombank": [
        "https://favicon.yandex.net/favicon/v2/https://www.gazprombank.ru?size=120",
        "https://cdn.gpb.ru/upload/files/bve/802/tltfdbfk6msczspsuxm277gjq3aghnsl/Logo_01_GPB.png",
    ],
    "raiffeisen": [
        "https://favicon.yandex.net/favicon/v2/https://www.raiffeisen.ru?size=120",
    ],
    "psb": [
        "https://favicon.yandex.net/favicon/v2/https://www.psbank.ru?size=120",
        "https://www.psbank.ru/apple-touch-icon.png",
    ],
    "sovcombank": [
        "https://favicon.yandex.net/favicon/v2/https://sovcombank.ru?size=120",
    ],
    "rosbank": [
        "https://favicon.yandex.net/favicon/v2/https://www.rosbank.ru?size=120",
    ],
    "mkb": [
        "https://favicon.yandex.net/favicon/v2/https://mkb.ru?size=120",
        "https://mkb.ru/apple-touch-icon.png",
    ],
    "rshb": [
        "https://favicon.yandex.net/favicon/v2/https://www.rshb.ru?size=120",
    ],
    "open-bank": [
        "https://favicon.yandex.net/favicon/v2/https://www.open.ru?size=120",
        "https://static.rustore.ru/apk/1398463/content/ICON/ffd45b51-77b9-48df-b44c-76fcdbe2de20.png",
    ],
    "uralsib": [
        "https://favicon.yandex.net/favicon/v2/https://www.uralsib.ru?size=120",
    ],
    "homecredit": [
        "https://favicon.yandex.net/favicon/v2/https://www.home.bank?size=120",
        "https://favicon.yandex.net/favicon/v2/https://www.homecredit.ru?size=120",
    ],
    "ozon-bank": [
        "https://favicon.yandex.net/favicon/v2/https://finance.ozon.ru?size=120",
        "https://finance.ozon.ru/favicon.ico",
    ],
    "yandex-bank": [
        "https://favicon.yandex.net/favicon/v2/https://bank.yandex.ru?size=120",
        "https://yandex.ru/apple-touch-icon.png",
    ],
    "wbbank": [
        "https://wb-bank.ru/apple-touch-icon.png",
        "https://favicon.yandex.net/favicon/v2/https://wb-bank.ru?size=120",
    ],
    "otpbank": [
        "https://favicon.yandex.net/favicon/v2/https://www.otpbank.ru?size=120",
    ],
    "atb": [
        "https://favicon.yandex.net/favicon/v2/https://www.atb.su?size=120",
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
    # Reject empty/HTML and 1×1 favicon placeholders from CDN.
    if len(data) < 200:
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
