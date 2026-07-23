# Android-клиент — релиз

Полноценный мобильный клиент (Capacitor) к self-hosted инстансу Бухгалтера. Отдельный UI в `android/ui/`; веб `web/` в APK не входит.

Документация: [docs/android-client.md](../docs/android-client.md) ([UI](../docs/android-client-ui.md), [платформа](../docs/android-client-platform.md)).

**Распространение:** только **APK из [GitHub Releases](https://github.com/kai-zer-ru/buhgalter/releases)** — universal (`buhgalter-android-{version}.apk`) и per-ABI (`-arm64-v8a`, `-armeabi-v7a`, `-x86_64`). Google Play и iOS **не планируются**.

## Архитектура

- **Capacitor** — `android/ui/` → `ui/build`, нативная оболочка `android/app/`
- **API** — REST `/api/v1/...`, Bearer (session после логина/пароля или API-токен)
- **Офлайн** — outbox в `android/ui/src/lib/offline/` (coalescing)

## Первый запуск

Минимальный онбординг — без внешнего URL и Wi‑Fi:

1. **`/server-setup`** — mDNS (`_buhgalter._tcp`) + сканирование подсети (:8765) + ручной ввод **LAN URL**; проверка `GET /api/v1/health` (поле `external_url` — подпись домена из админки)
2. **`/login`** — выбор: логин/пароль (`/login/password`) или API-токен (`/login/token`)
3. Работа с приложением

Внешний URL, домашние Wi‑Fi (SSID), LAN fallback и HTTPS TOFU — только в **Настройки → Сервер** (`/settings/server`).

## Настройки сервера (после онбординга)

| Возможность | Где |
|-------------|-----|
| LAN / внешний URL, до 5 SSID, автовыбор по Wi‑Fi | `/settings/server` |
| LAN discovery «найти снова» | `/settings/server` |
| HTTPS self-signed — попап «Доверять» (TOFU) | при сохранении HTTPS origin |
| Смена сервера / отключение | `/settings/server` |

## Готово к релизу

### Ядро

- [x] Capacitor-проект, drawer-оболочка, вход: логин/пароль или API-токен
- [x] Главная, счета, операции, переводы, офлайн outbox
- [x] LAN discovery (mDNS `_buhgalter._tcp` + subnet scan), кнопка «Назад», полноэкранные формы
- [x] Два URL + SSID (в настройках сервера)
- [x] Блокировка PIN + биометрия (`/settings/security`)
- [x] MVP-навигация: долги, кредиты, бюджет, статистика, настройки, админка (без веб-only разделов)
- [x] Офлайн polish, версия в drawer, хабы настроек/админки

### Безопасность и дистрибуция

- [x] API-токен в secure storage
- [x] HTTPS self-signed — TOFU-попап «Доверять этому серверу»
- [x] Минимальная версия Android **8.0 (API 26)**
- [x] Release APK в CI → GitHub Releases (universal + per-ABI: `arm64-v8a`, `armeabi-v7a`, `x86_64`)
- [x] Release keystore в CI (`ANDROID_KEYSTORE_*` secrets; локально — `keystore.properties`)
- [x] Блокировка приложения при отставании мажор/минор (`app < server`, патч не блокирует) — полноэкранный экран, только «Скачать APK»
- [x] `make android-install` (adb)

### Только веб (не в APK)

- Управление пользователями (`/admin/users`)

### Добавлено после релиза

- [x] Создание кредита — пошаговый мастер `/credits/new/{basics,options,schedule}`
- [x] Главная: сводка долгов + виджет бюджета; «сделать периодической» на home / operations / account
- [x] PageLoadGate на профиле и токенах; EmptyState на recurring / categories / tokens
- [x] Вход: `/login` → логин/пароль или API-токен (с возвратом к выбору)

## Офлайн (outbox)

Очередь **намерений**; при появлении сети — FIFO replay. Kinds: `transaction` | `transfer` | `category` | `debt` | `account` | `budget`. Подробнее — [android-client-platform.md](../docs/android-client-platform.md#офлайн-outbox).

## Локализация

Строки в `android/ui/src/lib/i18n/`; в APK — bundled. При `app < server` — sync с `GET /api/v1/ui/i18n/{lang}` (см. [android-client-platform.md](../docs/android-client-platform.md#локализация-android)).

- [x] Синхронизация переводов с сервера при рассинхроне версий

## Nice-to-have

- [x] Ярлык «+ Расход» (static shortcut → `/transactions/new?type=expense`)
- [x] Синхронизация outbox при возврате в приложение (`appStateChange`)
- [x] Экспорт очереди outbox в буфер (Настройки → Сервер)
- [x] Stale-while-revalidate ref-cache + прогрев при старте и sync
- [x] Отладочное логирование (Настройки → Сервер) + экспорт в Загрузки
- [x] Глобальное состояние ошибки загрузки (`PageLoadGate`, `page-load.ts`) — карточка с «Повторить» вместо пустого экрана
- [x] Домен внешнего URL в списке LAN discovery (`external_url` в `/api/v1/health`)
- [x] Сброс PIN/биометрии при выходе, отключении сервера и истечении сессии
- [x] «Сохранить» в настройках/админке без перехода; формы «Добавить» — возврат через `from`
- [x] Тёмная adaptive icon (`values-night`) и splash
- [x] Themed icon: adaptive `<monochrome>` (Material You) + источник `icon-monochrome-512.png`
- [x] i18n: биометрия cancel, строки экспорта, `values-en` для shortcut
- [x] Home-screen виджеты: быстрые действия, баланс, бюджет, «Скоро», один счёт (`WidgetBridge` + WorkManager)
- [x] Static shortcuts: расход / доход / перевод
- [x] **Share-intent:** `ACTION_SEND` text/image → форма расхода с префиллом описания (`ShareTargetPlugin`)
- [x] **Шире outbox:** `account` (create/update/archive/unarchive) + `budget` (create/update/delete)
- [x] Фикс: Select / Combobox не «отрываются» от поля (`.relative` только вокруг контрола; [ui-dialogs.md](../docs/ui-dialogs.md))

## Опционально (не начато)

Не блокер; **не** веб-паритет.

- Outbox: кредиты/платежи, recurring, удаление счёта, settle долга
- Share: вложения файла в операцию / OCR ([сканер чеков](receipt-scanner.md)); `SEND_MULTIPLE` / PDF
- Не путать share со **static shortcuts** (launcher)
## Сборка

```bash
make android-sync
make android-apk              # debug
make android-apk-release      # release APKs (universal + per-ABI)
make android-install          # debug + adb
make android-install-release  # release universal + adb
```

## Решения (не открытые вопросы)

- **Google Play** — нет, только APK из Releases
- **iOS** — не планируется
- **mDNS** — сервер публикует `_buhgalter._tcp` (отключение: `BUHGALTER_MDNS_ENABLED=false`)
- **Certificate pinning** — нет; для self-hosted HTTPS — TOFU в настройках сервера
- **Минимальный Android** — API 26+
