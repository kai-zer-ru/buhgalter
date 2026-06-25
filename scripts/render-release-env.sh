#!/usr/bin/env bash
# .env для GitHub Release: на основе docker/.env.example рядом с docker-compose.yaml.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"
VERSION="${VERSION#v}"
if [[ -z "$VERSION" ]]; then
	echo "render-release-env: нужен аргумент VERSION" >&2
	exit 1
fi

EXAMPLE="$ROOT/docker/.env.example"
OUT_DIR="$ROOT/build/release"
mkdir -p "$OUT_DIR"

if [[ ! -f "$EXAMPLE" ]]; then
	echo "render-release-env: не найден $EXAMPLE" >&2
	exit 1
fi

sed "s/^BUHGALTER_IMAGE_TAG=.*/BUHGALTER_IMAGE_TAG=${VERSION}/" "$EXAMPLE" >"$OUT_DIR/.env"
cp "$EXAMPLE" "$OUT_DIR/.env.example"
echo "render-release-env: build/release/.env и .env.example (BUHGALTER_IMAGE_TAG=${VERSION})"
