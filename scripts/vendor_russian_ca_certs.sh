#!/usr/bin/env bash
# Обновить vendored-сертификаты НУЦ Минцифры для docker/Dockerfile (MAX official API).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/docker/certs"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSLO --output-dir "$TMP" https://gu-st.ru/content/lending/linux_russian_trusted_root_ca_pem.zip
curl -fsSLO --output-dir "$TMP" https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.zip
(
	cd "$TMP"
	unzip -o -q linux_russian_trusted_root_ca_pem.zip
	unzip -o -q russian_trusted_sub_ca_pem.zip
)
mkdir -p "$DEST"
cp "$TMP/russian_trusted_root_ca_pem.crt" "$TMP/russian_trusted_sub_ca_2024_pem.crt" "$DEST/"
echo "OK: $DEST"
