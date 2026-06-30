#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${BUHGALTER_E2E_DATA_DIR:-$(mktemp -d)}"
export BUHGALTER_DATA_DIR="$DATA_DIR"
export BUHGALTER_DB_PATH="${DATA_DIR}/buhgalter.db"
export BUHGALTER_ADDR="${BUHGALTER_ADDR:-:9876}"
export BUHGALTER_STATIC_EMBED=true
export BUHGALTER_E2E=1
export BUHGALTER_LOG_DIR="${DATA_DIR}/logs"
export BUHGALTER_LOCALES_DIR="${ROOT}/server/locales"

BIN="${ROOT}/bin/buhgalter"
if [[ ! -x "$BIN" ]]; then
	echo "e2e-server: run 'make build' first (missing $BIN)" >&2
	exit 1
fi

echo "e2e-server: data=$DATA_DIR addr=$BUHGALTER_ADDR" >&2
cd "$ROOT"
exec "$BIN"
