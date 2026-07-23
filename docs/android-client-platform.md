# Android-клиент — платформа

Архитектура, авторизация, сеть, офлайн и сборка. UI — в [android-client-ui.md](android-client-ui.md).

## Структура репозитория

```
android/
├── ui/                    # SvelteKit → ui/build
├── app/                   # Gradle / Capacitor WebView
├── capacitor.config.json    # webDir: ui/build
└── package.json             # @capacitor/*
```

Корневой `web/` и бинарник сервера (`make build`) **не зависят** от Android.

## Capacitor

- Конфиг: `android/capacitor.config.json`
- `webDir`: `ui/build`
- Схема WebView: `https` (`localhost` — origin приложения, не API-сервер)

Синхронизация: `make android-sync` (сборка UI + иконки + `npx cap sync android`).

## Авторизация

После выбора сервера экран `/login` предлагает два способа:

| Способ | Маршрут | API | Что сохраняется |
|--------|---------|-----|-----------------|
| **Логин / пароль** | `/login/password` | `POST /api/v1/auth/login` | session Bearer (`auth_kind=session`) |
| **API-токен** | `/login/token` | вставка токена → `GET /api/v1/auth/me` | API-токен (`auth_kind=api_token`) |

Можно вернуться на `/login` (другой способ) или на `/server-setup` (смена сервера).

1. Токен (session или API) хранится в **native secure storage** (`@aparajita/capacitor-secure-storage`) в APK; в **browser** (`vite dev` / обычный Chromium) — memory + `localStorage` (`buhgalter.secure.*`), без вызова Capacitor-плагина (нереализованный bridge зависает на «Загрузка…»). При первом запуске после обновления токен мигрируется из legacy `localStorage` ключа `buhgalter.auth_token`. Вид токена — `localStorage` ключ `buhgalter.auth_kind`.
2. Все запросы: `Authorization: Bearer <token>`, `credentials: omit`.
3. **Выход:** для `session` — `POST /api/v1/auth/logout` (если сеть есть), затем локальная очистка; для `api_token` — только локально. **Блокировка приложения** (PIN и биометрия) сбрасывается.

Код: `android/ui/src/lib/platform/auth-token.ts`, экраны `android/ui/src/routes/login/`.

См. также [api/authentication.md](api/authentication.md).

## URL сервера

Профиль сервера в `localStorage` (`buhgalter.server_profile.v1`):

| Поле | Назначение |
|------|------------|
| **LAN URL** | Адрес в домашней сети, напр. `http://192.168.1.176:8765` |
| **Внешний URL** | Опционально: reverse proxy / `external_url` — задаётся в **Настройки → Сервер** |
| **Домашние Wi‑Fi** | До **5** SSID — только в настройках сервера |
| **LAN fallback** | Опция в настройках: на домашнем Wi‑Fi при недоступном LAN пробовать внешний URL |
| **Доверенные HTTPS** | TOFU для self-signed — при сохранении HTTPS в настройках |

**Первый запуск** (`/server-setup`): только LAN discovery + ручной **LAN URL**. Внешний URL и SSID — позже в `/settings/server`.

- `getApiBase()` / `getServerUrl()` возвращают **активный** URL после `refreshActiveServerUrl()` (SSID + профиль).
- Миграция: старый `buhgalter.server_url` → `lanUrl`.
- Проверка перед сохранением: `pingServer(origin)` → `GET /api/v1/health` **без** Bearer.
- **LAN discovery:** на `/server-setup` и в `/settings/server` — mDNS (`_buhgalter._tcp`, ~4 с) параллельно со сканированием Wi‑Fi-подсети (порт **8765**) + ручной ввод. Сервер публикует mDNS при старте (`BUHGALTER_MDNS_ENABLED`, по умолчанию `true`). Если в админке задан **внешний URL**, он возвращается в `GET /api/v1/health` как `external_url` и показывается в списке найденных серверов (домен reverse proxy, не результат reverse DNS по IP).
- **SSID:** native `WifiSubnet.getSsid()`; Android 10+ требует `ACCESS_FINE_LOCATION` (кнопка «Добавить текущую сеть» запрашивает разрешение).
- Оба origin должны указывать на **один** инстанс сервера; ref-cache хранится отдельно по URL.
- Ошибки на `/server-setup` — **одна строка под полем**, без стека toast-уведомлений.

Код: `server-profile.ts`, `server-url.ts`, `server-connect.ts`, `server-setup/+page.svelte`, `settings/server/+page.svelte`.

### Адрес и порт

- Нужен **API-сервер** (`make dev-server` / бинарник), порт по умолчанию **8765**.
- `make dev-server` читает **корневой** `.env` (`BUHGALTER_ENV_FILE`); после правки `BUHGALTER_ALLOWED_HOSTS` перезапустите сервер.
- **Не** подходит Vite dev-web (`:5173`) — ответ будет HTML, не JSON.
- Пример: `http://192.168.1.176:8765` (IP ПК в LAN, не `127.0.0.1` с телефона).

### Доступ с телефона (middleware ExternalAccess)

Сервер проверяет `Host` запроса. Для LAN без `external_url` в настройках добавьте IP или hostname сервера в `.env`:

```env
BUHGALTER_ALLOWED_HOSTS=192.168.1.176
```

Либо укажите **external URL** в веб-админке (тот же origin, что вводите в приложении). Иначе ответ **403** — в приложении текст «Сервер отклонил запрос…».

### HTTPS и self-signed

Для **внешнего HTTPS** с самоподписанным или иным недоверенным сертификатом при проверке `/api/v1/health` показывается попап: переключатель **«Доверять этому серверу»** и предупреждение «на свой страх и риск». После включения origin попадает в `trustedOrigins` профиля; запросы к API для этого origin идут через native `SslTrust` (OkHttp с отключённой проверкой только для явно доверенных host). HTTP (LAN) — без изменений (`usesCleartextTraffic`).

Код: `SslTrustPlugin.java`, `ssl-trust.ts`, `ServerTrustDialog.svelte`, `server-verify.ts`.

### Bootstrap (`+layout.svelte`)

1. Нет URL → `/server-setup`, сброс локального токена
2. Есть URL, нет пользователя на защищённом маршруте → `/login` (выбор способа входа)
3. Есть токен → **сразу** экран PIN / биометрии (`unlockWithExistingSession`), даже без кэша `/auth/me`. Профиль берётся из `buhgalter.last_user.v1` (не привязан к URL), затем ref-cache (любой origin LAN/remote), иначе минимальный stub. Офлайн до ответа `/health`; probe + `loadUser` + версии / remote i18n — **только в фоне**. Health **никогда** не блокирует ввод PIN при наличии сессии
4. Нет токена → `prepareBootstrapConnectivity` (может ждать probe) и дальше login / «Сервер недоступен»
5. После мутаций ref-cache сбрасывается с **сохранением** `/api/v1/auth/me` (`clearRefCache({ preserveAuthMe: true })`); профиль также пишется в `last_user` при логине и успешном `loadUser`

## Офлайн (outbox)

Каталог: `android/ui/src/lib/offline/`.

- Очередь create / update / delete для **`transaction` | `transfer` | `category` | `debt` | `account` | `budget`** (`EntityKind` в `types.ts`). Wrappers: `transactions-api`, `categories-api`, `debts-api`, `accounts-api`, `budgets-api`. Ещё онлайн-only: кредиты, recurring, удаление счёта, settle долга.
- **Coalescing** (create→edit→delete на одной сущности)
- При появлении сети: `scheduleSyncOutbox()` → FIFO replay через существующие API
- Синхронизация не стартует без настроенного URL сервера
- **Справочники (ref-cache):** GET через `request()` кэшируются в `localStorage` (ключ включает URL сервера), **кроме** `GET /credits/{id}` (полный график платежей слишком тяжёлый для синхронного SWR), путей с `/preview` (в т.ч. `GET /budgets/spent-preview`) и **`GET /setup/status`** (флаг регистрации / bootstrap). **Stale-while-revalidate:** при наличии кэша экран рисуется сразу, сеть — в фоне (`ref-cache.ts`, store `refCacheUpdate` по path, legacy `refCacheTick`). При недоступности сервера — только кэш (некэшируемые GET сразу дают miss). `GET /ui/meta` прогревает кэш `GET /categories?type=…`. Кэш очищается при logout и отключении сервера; после успешного write — `clearRefCache({ preserveAuthMe: true })` (профиль `/auth/me` сохраняется для офлайн PIN). **`assignIfChanged`** в `load({ background: true })` — меньше лишних reactive-прогонов при неизменившемся ответе. Native HTTPS (`SslTrust`) выполняется в пуле потоков + JS-таймаут, чтобы не блокировать очередь CapacitorPlugins.
- **Списки операций:** после merge outbox + server (`mergeTransactionLists`) клиент сортирует `transaction_date DESC`, `created_at DESC` — как API `sort=date_desc`. Иначе pending из outbox (FIFO) оказывались сверху в порядке создания (от старых к новым).
- **Прогрев кэша:** при старте приложения и ручной синхронизации (`warmRefCache`) — dashboard, ui/meta, счета, кредиты, долги, бюджет, типовые списки операций.
- **Ошибка загрузки экрана:** при сбое первой загрузки без кеша — `PageLoadGate` с текстом и «Повторить» (не пустой экран); при фоновом SWR с уже показанными данными — только toast. См. [android-client-ui.md](android-client-ui.md#ошибка-загрузки-страницы).
- **Локальное состояние:** `balance-overlay.ts` пересчитывает баланс и прогноз по outbox относительно последнего снимка с сервера; `local-state.ts` применяет дельты к dashboard, счетам и формам. Индекс транзакций (`transaction-index.ts`) нужен для корректного update/delete офлайн.
- **Синхронизация:** ручной pull (`AndroidDrawerSync`) — кнопка всегда активна (кроме момента sync); в офлайне сначала принудительный `/health`, при успехе — прогрев кэша + replay outbox, при неудаче остаётся офлайн. Фоновый `syncOutbox` — только outbox. После успешной отправки страницы обновляются через `dataRefreshTick`; фоновое обновление кэша — через `refCacheTick`. После локального save в outbox — мгновенный re-merge через `localDataTick` / `outboxTick`. Кнопка sync в drawer **не закрывает** меню.
- **Версии:** `fetchAppVersionInfo()` сравнивает `APP_VERSION` с `current_version` сервера. **Блокировка** (`VersionBlockScreen`) — только при отставании **мажор** или **минор** (`1.4.x` vs `1.5.x`, `1.x` vs `2.x`). Патч (`1.4.0` vs `1.4.1`) — предупреждение в drawer и попап по клику, без блокировки.
- **Failed sync:** операции с ошибкой отправки помечаются ⚠️ в списке (tooltip с текстом ошибки); баннер «не отправилось» — только при доступном сервере; в офлайне счётчик в полоске внизу и в drawer.
- **Outbox при resume / смене сети:** внеочередной probe `/health`; при online — warm cache и `scheduleSyncOutbox`, если очередь не пуста.
- **Экспорт очереди:** Настройки → Сервер — JSON в буфер обмена (для support).
- **Ярлыки и share:** static shortcuts «+ Расход/Доход», «Перевод» → deep links; **«Поделиться»** (`ACTION_SEND` text/plain, image/*) → `ShareTargetPlugin` → `/transactions/new?type=expense` с префиллом описания (`share-target.ts`). Вложение файла / OCR — не в MVP.
- **Тёмная иконка / splash:** `values-night/ic_launcher_background` (#0f172a), splash в night mode — тот же фон до загрузки WebView. Это системный night mode лаунчера, не настройка темы SPA (`system` / light / dark в профиле). **Night bg лаунчера не гарантирован на HyperOS/Xiaomi** (кеш иконки); эталон — Pixel Launcher / AVD. Splash при этом может темнеть независимо.
- **Themed icon (Material You):** слой `<monochrome>` в adaptive icon (`@drawable/ic_launcher_monochrome`), исходник `ui/static/icon-monochrome-512.png` → `make android-icons`. Работает при включённых тематических иконках ОС (API 33+); на OEM — основной способ «иконка меняет вид».
- **SystemBars:** Capacitor 8 `SystemBars.setStyle` из `applyTheme` (`system-bars.ts`) — contrast иконок status/nav bar следует **resolved** теме SPA, не только OS night. Cold start до JS может кратко совпадать с device night.
- **Доступность сервера:** `server-connectivity.ts` — отдельно от «есть Wi‑Fi». При недоступности API (в т.ч. Capacitor `Failed to connect`) включается офлайн-режим: запросы не уходят на сервер, GET берутся из ref-cache, мутации — в outbox. Полоска внизу показывает «Нет соединения» и при наличии очереди — число операций; фоновая проверка `/health` раз в **60 с**; при смене сети / resume приложения — внеочередной probe; кнопка «Синхронизировать» тоже форсирует probe. Опциональный fallback LAN→remote на домашнем SSID.

Плагины: `@capacitor/network`, `@capacitor/preferences`, `@capacitor/app` (кнопка «Назад», блокировка в фоне), встроенный `SystemBars` (стиль status/nav), native `WifiSubnet`, `SslTrust`, `ShareTarget`, `WidgetBridge`, `DebugExport`, `LanDiscovery`; `@aparajita/capacitor-biometric-auth`, `@aparajita/capacitor-secure-storage` (PIN и API-токен).
## Блокировка приложения

Настройки → **Безопасность** (`/settings/security`).

- **PIN** — 4 цифры; обязателен при включении блокировки; отклоняются простые комбинации (повторы, последовательности, 1234, 1111 и т.п.)
- **Биометрия** — опционально (отпечаток / Face ID), если устройство поддерживает
- Блокировка при **холодном старте** и после выбранного **таймаута в фоне** (`appStateChange` → `@capacitor/app`): 30 с / 1 мин / 5 / 10 / 15 / 30 мин / 1 час (по умолчанию **1 мин**)
- PIN-hash хранится в **native secure storage** (`@aparajita/capacitor-secure-storage`), не в открытом `localStorage` (в browser — тот же `secure-store` fallback на `buhgalter.secure.*`)
- **Выход** (drawer), **отключение сервера** (Настройки → Сервер) и **истечение сессии** — `clearAppLock()`: PIN, биометрия и таймаут удаляются без ввода старого PIN; следующий вход — настройка с нуля
- Экран разблокировки: `AppLockScreen.svelte` — между auth bootstrap и `AndroidShell`

**Ограничения:** PIN скрывает UI на устройстве; ref-cache и outbox остаются в `localStorage`. API-токен — в secure storage, но не шифрует остальные локальные данные.

Код: `app-lock.ts`, `secure-store.ts`, `AppLockScreen.svelte`, `settings/security/+page.svelte`.

Плагины: `@aparajita/capacitor-biometric-auth`, `@aparajita/capacitor-secure-storage`; Android: `USE_BIOMETRIC`.

## Отладочное логирование

Экран: **Настройки → Сервер** — переключатель «Включить логирование».

### Включение и экспорт

1. Включить переключатель — начинается новая сессия лога (предыдущие записи очищаются), toast «Логирование включено».
2. Повторить проблемный сценарий в приложении.
3. Выключить переключатель — диалог «Сохранить лог?»: **Скачать** → файл `buhgalter-debug-YYYY-MM-DDTHH-MM-SS.log` в **Загрузки** (Android 10+ через MediaStore; plugin `DebugExportPlugin.java`); **Пропустить** — журнал остаётся в памяти до следующего включения.

В браузере при `npm run dev` (без native) — скачивание через `<a download>`.

### Что пишется

| Категория | Источник | Примеры |
|-----------|----------|---------|
| `api` | `client.ts` | `→ GET /accounts`, `← GET /accounts 200 (42ms)`, ошибки с кодом API |
| `sync` | `offline/sync.ts` | `syncOutbox started/finished`, `warmRefCache`, skip при offline |
| `cache` | `offline/ref-cache.ts` | SWR hit, revalidate, offline miss |
| `connectivity` | `serverReachability` store | смена `online` / `offline` / `checking` |
| `nav` | `+layout.svelte` | смена маршрута |
| `bootstrap` | `+layout.svelte` | старт приложения |
| `session` | `setDebugLogEnabled(true)` | снимок окружения при включении |
| `uncaught` / `unhandledrejection` | глобальные listeners | JS-ошибки |

Лимит — **3000** записей в `localStorage` (`buhgalter.debug_log.entries`); старые отбрасываются. Флаг — `buhgalter.debug_log.enabled`.

### Редактирование секретов

`redactHeaders` / `redactData`: ключи `authorization`, `token`, `password`, `pin`, `secret`, `api_token` → маска; `Authorization: Bearer ***`; длинные строки обрезаются до 2000 символов.

### Формат экспорта

Текстовый файл UTF-8:

1. Заголовок и время экспорта
2. Блок **Environment** — JSON: версия APK, native/web, userAgent, активный URL, профиль сервера, reachability, счётчики outbox
3. Блок **Events** — строки `ISO8601 [level] [category] message {json}`

Код: `debug-log.ts`, `debug-export.ts`, UI — `settings/server/+page.svelte`. Тесты: `debug-log.test.ts`.

Серверные HTTP-логи (не путать с клиентским журналом) — `BUHGALTER_LOG_MODE` в `.env`, см. [README.md](../README.md#отладочные-логи).

## Home-screen виджеты

Launcher-виджеты (не карточки на главной в WebView). Реализация: `AppWidgetProvider` + `RemoteViews` в `android/app/.../widgets/`.

| Виджет | Содержание | Тап |
|--------|------------|-----|
| Быстрые действия | Расход / Доход / Перевод | формы операций |
| Баланс | «Мои средства», прогноз, долг по картам | `/` |
| Бюджет месяца | план / факт / прогресс | `/budget` |
| Скоро | до 5 ближайших: кредит, долг, future | карточка / список |
| Один счёт | выбранный счёт (основной по умолчанию) | `/accounts/{id}` |

Данные: JSON-снимок в EncryptedSharedPreferences через Capacitor plugin `WidgetBridge` (`publish` / `setLockEnabled` / `clear`). UI публикует снимок после `warmRefCache` и загрузки главной; выход и смена сервера очищают. Фон: WorkManager раз в 60 мин (`WidgetRefreshWorker`, OkHttp + Bearer).

Приватность: при включённом PIN (`app_lock.enabled`) суммы в виджетах — «Заблокировано» / `••••`; quick actions остаются кликабельны (вход всё равно через lock). Без токена — «Откройте приложение и войдите».

Код UI: `android/ui/src/lib/widgets/`.

## Локализация (Android)

Строки UI лежат в `android/ui/src/lib/i18n/{ru,en}.json` и вшиты в APK.

При **рассинхроне версий** (`APP_VERSION` < `current_version` с `GET /api/v1/version/check`) клиент дополнительно запрашивает `GET /api/v1/ui/i18n/{lang}` и мержит каталог поверх bundled (`svelte-i18n` `addMessages`). Кеш в `localStorage` по паре «версия сервера + язык». Офлайн — мягкий сбой (остаются строки APK).

На сервере каталоги копируются в `server/ui_locales/` (`make copy-ui-i18n`, проверка `ui-i18n-check` в CI) и отдаются с версией бинарника. Это отдельно от `server/locales/` (сообщения API/ошибок).

Код: `android/ui/src/lib/i18n/remote-sync.ts`, `server/internal/ui/i18n.go`.

## Сборка и установка

| Команда | Действие |
|---------|----------|
| `make android-icons` | Mipmaps из `ui/static/icon-512.png` + monochrome из `icon-monochrome-512.png` |
| `make android-ui-build` | `category-icons-json` + `npm run build` в `android/ui` |
| `make prepare-android` | `npm run lint:fix` в `android/ui` (входит в `make prepare`) |
| `make test-unit` | Go + web/android `svelte-check` + android `vitest` |
| `make test-e2e-web` | Playwright e2e web UI |
| `make test-e2e` | = `test-e2e-web` (Android e2e нет — ручная приёмка на устройстве) |
| `make android-sync` | UI build + icons + cap sync |
| `make android-apk` | Debug APK |
| `make android-apk-release` | Release APK: universal + per-ABI (`arm64-v8a`, `armeabi-v7a`, `x86_64`) |
| `make android-install` | Debug-сборка + `adb install -r` |
| `make android-install-release` | Release-сборка + `adb install -r` для `app-universal-release.apk` |

Требования: **Android 8.0+** (API 26), JDK 21, Android SDK, Node 20+, **Pillow** (`pip3 install --user Pillow` — для `make android-icons`). Окружение: `scripts/setup-android-dev.sh`, `scripts/android-env.sh`.

Debug APK: `android/app/build/outputs/apk/debug/app-debug.apk`.  
Release APK (`make android-apk-release` → `android/app/build/outputs/apk/release/`):

| Файл | Назначение |
|------|------------|
| `app-release.apk` | Universal (все ABI) |
| `app-arm64-v8a-release.apk` | Телефоны arm64 |
| `app-armeabi-v7a-release.apk` | Старые 32-bit ARM |
| `app-x86_64-release.apk` | Эмулятор x86_64 |

На GitHub Release: `buhgalter-android-{version}.apk` (universal) и `buhgalter-android-{version}-{abi}.apk`.

CI (`.github/workflows/release.yml`): job `android-apk` ставит SDK (`platforms;android-36`, `build-tools;36.0.0`), пишет `android/local.properties`, Pillow, собирает `make android-apk-release`, проверяет все четыре APK (`scripts/verify-android-release-apks.sh`), заливает каталог `release/` артефактом; job `goreleaser` скачивает артефакт в тот же путь и прикрепляет файлы через `release.extra_files` в `.goreleaser.yaml`. Без всех APK релиз падает.

Подпись release APK: `keystore.properties` / GitHub Secrets; `*.jks` в `.gitignore` (см. [android/README.md](../android/README.md)).

## HTTP без TLS

Self-hosted API обычно на `http://` в LAN. Нужно три уровня:

1. `AndroidManifest.xml`: `usesCleartextTraffic="true"` — разрешает HTTP на уровне ОС.
2. `capacitor.config.json`: `plugins.CapacitorHttp.enabled: true` — `fetch` к API идёт через нативный HTTP (обходит mixed content: WebView на `https://localhost` иначе блокирует запросы к `http://192.168.x.x`).
3. `android.allowMixedContent: true` — запасной вариант для прочих запросов в WebView.

После смены конфига: `make android-sync` и переустановка APK (`make android-install`).

## Возможности платформы (сводка)

- Отдельный UI в `android/ui/`, drawer-оболочка (~70% ширины; «Главная» первым)
- Вход: логин/пароль (session) или API-токен; токен в secure storage; экран URL при первом запуске
- LAN discovery (mDNS + subnet scan); `external_url` в health для подписи домена в списке
- Два URL + SSID, HTTPS TOFU, офлайн outbox (в т.ч. счета и бюджет) и ref-cache SWR
- Создание кредита (пошаговый мастер); home-screen виджеты; share-intent; static shortcuts
- Тема `light` | `dark` | `system` (default); SystemBars по resolved теме; themed icon `<monochrome>`
- Remote i18n при `app < server` (`GET /ui/i18n/{lang}`)
- Блокировка PIN/биометрия; настраиваемый таймаут в фоне; сброс при выходе и отключении сервера
- `PageLoadGate` — ошибки начальной загрузки с retry; `gotoReplace` после создания сущностей
- Отладочное логирование с экспортом в Загрузки
- Release APK в GitHub Releases (universal + per-ABI)

## См. также

- [android-client-ui.md](android-client-ui.md) — экраны и навигация
- [roadmap/android-client.md](../roadmap/android-client.md)
