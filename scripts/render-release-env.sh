#!/usr/bin/env bash
# .env для GitHub Release: тег образа под версию релиза (compose — docker/docker-compose.yml).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"
VERSION="${VERSION#v}"
if [[ -z "$VERSION" ]]; then
	echo "render-release-env: нужен аргумент VERSION" >&2
	exit 1
fi

mkdir -p "$ROOT/build/release"
printf 'BUHGALTER_IMAGE_TAG=%s\n' "$VERSION" >"$ROOT/build/release/.env"
echo "render-release-env: build/release/.env (BUHGALTER_IMAGE_TAG=${VERSION})"
