#!/usr/bin/env bash
# Установка JDK 21 и Android SDK в домашний каталог (без sudo).
# Запуск: ./scripts/setup-android-dev.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OPT_DIR="${HOME}/.local/opt"
SDK_DIR="${HOME}/Android/Sdk"
JDK_DIR="${OPT_DIR}/jdk-21.0.11+10"
CMDLINE_ZIP_URL="https://dl.google.com/android/repository/commandlinetools-linux-13114758_latest.zip"
JDK_URL="https://api.adoptium.net/v3/binary/latest/21/ga/linux/x64/jdk/hotspot/normal/eclipse?project=jdk"

echo "==> JDK 21 (Temurin)"
if [[ -x "${JDK_DIR}/bin/java" ]]; then
	echo "    уже установлен: ${JDK_DIR}"
else
	mkdir -p "${OPT_DIR}"
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp}"' EXIT
	echo "    скачивание…"
	curl -fsSL "${JDK_URL}" -o "${tmp}/jdk21.tar.gz"
	tar -xzf "${tmp}/jdk21.tar.gz" -C "${OPT_DIR}"
	echo "    установлен: ${JDK_DIR}"
fi

export JAVA_HOME="${JDK_DIR}"
export PATH="${JAVA_HOME}/bin:${PATH}"
java -version

echo "==> Android SDK"
mkdir -p "${SDK_DIR}/cmdline-tools"
if [[ ! -x "${SDK_DIR}/cmdline-tools/latest/bin/sdkmanager" ]]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp}"' EXIT
	echo "    скачивание command-line tools…"
	curl -fsSL "${CMDLINE_ZIP_URL}" -o "${tmp}/cmdline-tools.zip"
	unzip -q "${tmp}/cmdline-tools.zip" -d "${tmp}/extract"
	rm -rf "${SDK_DIR}/cmdline-tools/latest"
	mkdir -p "${SDK_DIR}/cmdline-tools/latest"
	mv "${tmp}/extract/cmdline-tools/"* "${SDK_DIR}/cmdline-tools/latest/"
	echo "    command-line tools установлены"
else
	echo "    command-line tools уже есть"
fi

export ANDROID_HOME="${SDK_DIR}"
export ANDROID_SDK_ROOT="${SDK_DIR}"
export PATH="${SDK_DIR}/cmdline-tools/latest/bin:${SDK_DIR}/platform-tools:${PATH}"

echo "    лицензии SDK…"
yes | sdkmanager --licenses >/dev/null

echo "    platform-tools, android-36, build-tools 36.0.0…"
sdkmanager "platform-tools" "platforms;android-36" "build-tools;36.0.0"

echo "==> android/local.properties"
mkdir -p "${ROOT}/android"
printf 'sdk.dir=%s\n' "${SDK_DIR}" >"${ROOT}/android/local.properties"

ENV_FILE="${ROOT}/scripts/android-env.sh"
cat >"${ENV_FILE}" <<EOF
# Сгенерировано scripts/setup-android-dev.sh — source в shell или подключается из Makefile
export JAVA_HOME="${JDK_DIR}"
export ANDROID_HOME="${SDK_DIR}"
export ANDROID_SDK_ROOT="${SDK_DIR}"
export PATH="\${JAVA_HOME}/bin:\${ANDROID_HOME}/cmdline-tools/latest/bin:\${ANDROID_HOME}/platform-tools:\${PATH}"
EOF
chmod +x "${ENV_FILE}"

echo ""
echo "Готово."
echo "  source ${ENV_FILE}"
echo "  make android-apk"
echo ""
echo "Для постоянного окружения добавьте в ~/.zshrc:"
echo "  source ${ENV_FILE}"
