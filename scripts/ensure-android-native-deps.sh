#!/usr/bin/env bash
# Gradle needs @capacitor/android/capacitor/build.gradle; incomplete npm installs break release builds.
set -euo pipefail

ROOT="${1:-.}"
ANDROID="${ROOT}/android"
MARKER="${ANDROID}/node_modules/@capacitor/android/capacitor/build.gradle"

if [[ -f "$MARKER" ]]; then
	exit 0
fi

echo "android native deps: @capacitor/android is incomplete (missing build.gradle)" >&2
echo "  Reinstalling android/node_modules (npm ci)…" >&2
( cd "$ANDROID" && npm ci )

if [[ ! -f "$MARKER" ]]; then
	echo "ERROR: ${MARKER} still missing after npm ci" >&2
	exit 1
fi

echo "android native deps: OK"
