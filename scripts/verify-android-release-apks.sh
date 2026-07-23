#!/usr/bin/env bash
# Verify release APK set for GitHub Releases / goreleaser extra_files.
# Usage: scripts/verify-android-release-apks.sh [apk_dir]
set -euo pipefail

DIR="${1:-android/app/build/outputs/apk/release}"
REQUIRED=(
  app-release.apk
  app-arm64-v8a-release.apk
  app-armeabi-v7a-release.apk
  app-x86_64-release.apk
)

if [[ ! -d "$DIR" ]]; then
  echo "ERROR: APK directory not found: $DIR" >&2
  exit 1
fi

missing=0
for name in "${REQUIRED[@]}"; do
  path="$DIR/$name"
  if [[ ! -f "$path" ]]; then
    echo "ERROR: missing $path" >&2
    missing=1
    continue
  fi
  size=$(wc -c <"$path" | tr -d ' ')
  if [[ "$size" -lt 1000 ]]; then
    echo "ERROR: $path is too small (${size} bytes)" >&2
    missing=1
    continue
  fi
  echo "OK  $name ($(numfmt --to=iec --suffix=B "$size" 2>/dev/null || echo "${size}B"))"
done

if [[ "$missing" -ne 0 ]]; then
  echo "--- directory listing ---" >&2
  ls -la "$DIR" >&2 || true
  echo "ERROR: expected universal app-release.apk + per-ABI (arm64-v8a, armeabi-v7a, x86_64)" >&2
  exit 1
fi

echo "All release APKs present in $DIR"
