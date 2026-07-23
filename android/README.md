# Android (Capacitor)

Мобильный клиент Бухгалтера. Самодостаточный каталог: UI, Capacitor и Gradle — всё здесь, без зависимости от `web/`.

## Структура

| Путь | Назначение |
|------|------------|
| `ui/` | SvelteKit UI (сборка → `ui/build`) |
| `app/` | Gradle / WebView (Capacitor) |
| `capacitor.config.json` | Конфиг Capacitor |

## Требования

- Node.js 20+
- JDK 21+ (для Gradle)
- Android SDK: platform 36, build-tools 36.0.0
- **Минимальная версия Android: 8.0 (API 26)**

## Установка JDK и SDK (Arch, без sudo)

Один раз из корня репозитория:

```bash
chmod +x scripts/setup-android-dev.sh
./scripts/setup-android-dev.sh
```

Скрипт ставит в домашний каталог:

| Компонент | Путь |
|-----------|------|
| JDK 21 (Temurin) | `~/.local/opt/jdk-21.0.11+10` |
| Android SDK | `~/Android/Sdk` |
| Переменные окружения | `scripts/android-env.sh` |

Подключить окружение в текущем shell:

```bash
source scripts/android-env.sh
```

## Сборка

Из корня репозитория:

```bash
source scripts/android-env.sh   # если ещё не в .zshrc
make android-sync             # UI build + icons + cap sync
make android-apk              # debug APK
make android-apk-release      # release APKs (universal + per-ABI)
make android-install          # собрать debug и установить на устройство (adb)
make android-install-release  # собрать release и установить universal APK (adb)

# Из корня репо: prepare/test покрывают и android/ui
make prepare                  # + prepare-android (lint:fix)
make test-unit                # + android/ui check + vitest
```

Иконки лаунчера — из `ui/static/icon-512.png` + mono `icon-monochrome-512.png` (`make android-icons`).

UI — боковое меню слева (~70% ширины экрана): `ui/src/lib/android/AndroidShell.svelte`.

Release APKs: `app/build/outputs/apk/release/` — `app-release.apk` (universal), `app-arm64-v8a-release.apk`, `app-armeabi-v7a-release.apk`, `app-x86_64-release.apk`.

`package-lock.json` в `android/` и `android/ui/` **в git** (нужны для `npm ci` / кеша в CI).

```bash
adb install -r android/app/build/outputs/apk/debug/app-debug.apk
# или: make android-install
```

## Разработка UI

```bash
cd android/ui && npm install && npm run dev
```

Документация: [docs/android-client.md](../docs/android-client.md) ([UI](../docs/android-client-ui.md), [платформа](../docs/android-client-platform.md)).

### Отладочные логи

**Настройки → Сервер → «Включить логирование»** — журнал API, sync, кеша и ошибок (токены маскируются). При выключении — сохранение `buhgalter-debug-*.log` в **Загрузки**. Подробнее: [android-client-platform.md](../docs/android-client-platform.md#отладочное-логирование).

После изменений: `make android-sync`, затем Run в Android Studio (каталог `android/`).

**Application ID:** `ru.kai_zer.buhgalter`

`local.properties` создаётся скриптом установки и в git не попадает.

## Подпись release APK

Локально: `android/keystore.properties` по образцу `keystore.properties.example`. Файл `.jks` **не коммитить** (в корневом `.gitignore`: `*.jks`); хранить отдельно и бэкапить пароль.

CI (GitHub Secrets): `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD`. Без secrets CI подпишет debug-keystore (предупреждение в логе).

Пароль keystore — только **ASCII** (PKCS12 в современных JDK не принимает кириллицу).

## HTTP (не HTTPS)

Для self-hosted без TLS в `AndroidManifest.xml` включён `usesCleartextTraffic` — только для локальных и доверенных сетей.
