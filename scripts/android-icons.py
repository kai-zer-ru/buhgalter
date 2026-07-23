#!/usr/bin/env python3
"""Generate Android launcher mipmaps from android/ui/static/icon-512.png.

Also requires icon-monochrome-512.png (Material You themed-icon source) and
wires <monochrome> into adaptive-icon XML (drawable/ic_launcher_monochrome).
"""

from __future__ import annotations

import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "android" / "ui" / "static" / "icon-512.png"
SRC_MONO = ROOT / "android" / "ui" / "static" / "icon-monochrome-512.png"
RES = ROOT / "android" / "app" / "src" / "main" / "res"
ANYDPI = RES / "mipmap-anydpi-v26"
MONOCHROME_DRAWABLE = RES / "drawable" / "ic_launcher_monochrome.xml"

LAUNCHER_SIZES = {
	"mipmap-mdpi": 48,
	"mipmap-hdpi": 72,
	"mipmap-xhdpi": 96,
	"mipmap-xxhdpi": 144,
	"mipmap-xxxhdpi": 192,
}

FOREGROUND_SIZES = {
	"mipmap-mdpi": 108,
	"mipmap-hdpi": 162,
	"mipmap-xhdpi": 216,
	"mipmap-xxhdpi": 324,
	"mipmap-xxxhdpi": 432,
}

ADAPTIVE_ICON_XML = """\
<?xml version="1.0" encoding="utf-8"?>
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@color/ic_launcher_background"/>
    <foreground android:drawable="@mipmap/ic_launcher_foreground"/>
    <monochrome android:drawable="@drawable/ic_launcher_monochrome"/>
</adaptive-icon>
"""


def resize_icon(img: Image.Image, size: int) -> Image.Image:
	out = img.resize((size, size), Image.Resampling.LANCZOS)
	if out.mode != "RGBA":
		out = out.convert("RGBA")
	return out


def write_adaptive_xml() -> None:
	ANYDPI.mkdir(parents=True, exist_ok=True)
	for name in ("ic_launcher.xml", "ic_launcher_round.xml"):
		(ANYDPI / name).write_text(ADAPTIVE_ICON_XML, encoding="utf-8")


def main() -> int:
	if not SRC.is_file():
		print(f"Source icon not found: {SRC}", file=sys.stderr)
		return 1
	if not SRC_MONO.is_file():
		print(
			f"Monochrome icon source not found: {SRC_MONO}\n"
			"Add a white-on-transparent silhouette (brand «1») for Material You themed icons.",
			file=sys.stderr,
		)
		return 1
	if not MONOCHROME_DRAWABLE.is_file():
		print(
			f"Monochrome drawable missing: {MONOCHROME_DRAWABLE}\n"
			"Keep vector drawable ic_launcher_monochrome.xml in res/drawable/.",
			file=sys.stderr,
		)
		return 1

	img = Image.open(SRC).convert("RGBA")

	for folder, size in LAUNCHER_SIZES.items():
		target_dir = RES / folder
		target_dir.mkdir(parents=True, exist_ok=True)
		icon = resize_icon(img, size)
		icon.save(target_dir / "ic_launcher.png")
		icon.save(target_dir / "ic_launcher_round.png")

	for folder, size in FOREGROUND_SIZES.items():
		target_dir = RES / folder
		target_dir.mkdir(parents=True, exist_ok=True)
		# Adaptive foreground: icon centered with ~12% inset safe zone.
		inset = int(size * 0.12)
		inner = size - inset * 2
		fg = Image.new("RGBA", (size, size), (0, 0, 0, 0))
		inner_img = resize_icon(img, inner)
		fg.paste(inner_img, (inset, inset), inner_img)
		fg.save(target_dir / "ic_launcher_foreground.png")

	write_adaptive_xml()

	night_bg = RES / "values-night" / "ic_launcher_background.xml"
	night_bg.parent.mkdir(parents=True, exist_ok=True)
	night_bg.write_text(
		'<?xml version="1.0" encoding="utf-8"?>\n'
		"<resources>\n"
		'    <color name="ic_launcher_background">#0f172a</color>\n'
		"</resources>\n",
		encoding="utf-8",
	)

	print(f"Generated launcher icons from {SRC}")
	print(f"Adaptive monochrome → {MONOCHROME_DRAWABLE} (source {SRC_MONO})")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())
