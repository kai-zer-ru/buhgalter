#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

usage() {
	echo "Usage: make version vX.Y.Z" >&2
	echo "       scripts/set_version.sh vX.Y.Z" >&2
	exit 1
}

[[ $# -eq 1 ]] || usage

raw="$1"
ver="${raw#v}"

if ! [[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
	echo "set_version: invalid semver: $raw (expected v1.2.3 or 1.2.3)" >&2
	exit 1
fi

printf '%s\n' "$ver" > VERSION

sed_inplace() {
	local file="$1"
	local pattern="$2"
	if [[ ! -f "$file" ]]; then
		echo "set_version: missing file: $file" >&2
		exit 1
	fi
	sed -i "$pattern" "$file"
}

sed_inplace Makefile "s/^VERSION ?= .*/VERSION ?= ${ver}/"

for openapi in \
	docs/api/openapi.yaml \
	server/internal/docs/openapi.yaml; do
	sed_inplace "$openapi" "s/^  version: .*/  version: ${ver}/"
done

sed_inplace docker/Dockerfile "s/^ARG VERSION=.*/ARG VERSION=${ver}/"

python3 - "$ver" <<'PY'
import re
import sys
from pathlib import Path

ver = sys.argv[1]

main_go = Path("server/cmd/buhgalter/main.go")
text = main_go.read_text(encoding="utf-8")
new_text, n = re.subn(
    r"(var \(\n\t)version\s*=.*",
    rf'\g<1>version       = "{ver}"',
    text,
    count=1,
)
if n != 1:
    raise SystemExit(f"set_version: could not update {main_go}")
main_go.write_text(new_text, encoding="utf-8")

pkg = Path("web/package.json")
data = __import__("json").loads(pkg.read_text(encoding="utf-8"))
data["version"] = ver
pkg.write_text(__import__("json").dumps(data, ensure_ascii=False, indent="\t") + "\n", encoding="utf-8")

lock = Path("web/package-lock.json")
if lock.exists():
    lock_data = __import__("json").loads(lock.read_text(encoding="utf-8"))
    lock_data["version"] = ver
    root = (lock_data.get("packages") or {}).get("")
    if isinstance(root, dict):
        root["version"] = ver
    lock.write_text(__import__("json").dumps(lock_data, ensure_ascii=False, indent="\t") + "\n", encoding="utf-8")
PY

echo "set_version: ${ver}"
echo "  VERSION"
echo "  Makefile"
echo "  docs/api/openapi.yaml"
echo "  server/internal/docs/openapi.yaml"
echo "  server/cmd/buhgalter/main.go"
echo "  docker/Dockerfile"
echo "  web/package.json"
echo "  web/package-lock.json"
