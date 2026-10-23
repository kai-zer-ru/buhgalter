#!/usr/bin/env bash
# Fallback release signing: ~/.android/debug.keystore (same as CI when secrets are unset).
set -euo pipefail

ROOT="${1:-.}"
if [ -f "${ROOT}/android/keystore.properties" ]; then
	exit 0
fi
if [ -n "${ANDROID_KEYSTORE_FILE:-}" ]; then
	exit 0
fi

KEYSTORE="${HOME}/.android/debug.keystore"
if [ -f "$KEYSTORE" ]; then
	exit 0
fi

if [ -z "${JAVA_HOME:-}" ] || [ ! -x "${JAVA_HOME}/bin/keytool" ]; then
	echo "ERROR: JAVA_HOME must point to a JDK with keytool to create ${KEYSTORE}" >&2
	echo "  source scripts/android-env.sh" >&2
	exit 1
fi

mkdir -p "${HOME}/.android"
"${JAVA_HOME}/bin/keytool" -genkey -v -keystore "$KEYSTORE" -storepass android \
	-alias androiddebugkey -keypass android -keyalg RSA -keysize 2048 -validity 10000 \
	-dname "CN=Android Debug,O=Android,C=US"
echo "Created ${KEYSTORE} for local release APK signing (debug key; not for Play Store)."
echo "  Production: android/keystore.properties — see keystore.properties.example"
