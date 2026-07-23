# Android-клиент — обзор

Мобильное приложение (Capacitor) к self-hosted инстансу Бухгалтера.

**Исходники целиком в `android/`** — веб `web/` в сборку APK не входит.

| Документ | Содержание |
|----------|------------|
| [android-client-ui.md](android-client-ui.md) | Интерфейс: оболочка, навигация, экраны |
| [android-client-platform.md](android-client-platform.md) | Платформа: авторизация, сервер, офлайн, сборка |

Сборка APK: [android/README.md](../android/README.md). Дорожная карта: [roadmap/android-client.md](../roadmap/android-client.md).

**Application ID:** `ru.kai_zer.buhgalter` (дефис в `kai-zer` → `_` по требованию Android).

## Первый запуск

1. **Адрес сервера** (`/server-setup`) — поиск в LAN (mDNS + скан подсети) + ручной **LAN URL**; проверка `GET /api/v1/health`. Если на сервере задан внешний URL в админке, в списке найденных серверов показывается его домен (поле `external_url` в ответе health).
2. **Вход** (`/login`) — выбор способа: **логин/пароль** (`/login/password`) или **API-токен** (`/login/token`); можно вернуться и сменить способ или сервер
3. Работа с API выбранного сервера

Внешний URL, домашние Wi‑Fi и прочие параметры сервера — **Настройки → Сервер** (`/settings/server`).

Смена сервера: боковое меню → Настройки → Сервер, или ссылка на экране входа.

Блокировка: Настройки → Безопасность — PIN, опциональная биометрия и таймаут в фоне. **Выход**, отключение сервера и истечение сессии сбрасывают блокировку — при следующем входе настраивается заново.

При сбое загрузки данных экран показывает сообщение и кнопку «Повторить» (не пустой экран) — см. [android-client-ui.md](android-client-ui.md#ошибка-загрузки-страницы).

**Отладка:** Настройки → Сервер → «Включить логирование»; при выключении — экспорт `buhgalter-debug-*.log` в Загрузки. См. [android-client-platform.md](android-client-platform.md#отладочное-логирование).

**Сохранить** в настройках и админке — toast, остаётесь на экране; **Добавить / изменить** (операция, счёт, …) — `leaveForm` (снятие слота формы через `history.back`), мастер кредита — `gotoReplace`. См. [android-client-ui.md](android-client-ui.md#сохранение-и-возврат).

Холодный старт с сохранённой сессией: PIN сразу (даже без кэша `/auth/me`), проверка сервера в фоне — [android-client-platform.md](android-client-platform.md#bootstrap-layoutsvelte).

**Только веб:** управление пользователями (`/admin/users`). Офлайн outbox — только Android. Создание кредита в APK — пошаговый мастер `/credits/new/*` (см. [android-client-ui.md](android-client-ui.md)).

Опциональный бэклог (кредиты/recurring в outbox, OCR share): [roadmap/android-client.md](../roadmap/android-client.md#опционально-не-начато).

**Тема:** светлая / тёмная / «Как на устройстве» (по умолчанию). **SystemBars** и themed icon — [android-client-platform.md](android-client-platform.md).

**Share:** из других приложений «Поделиться» (текст или картинка) открывает форму нового расхода. **Outbox:** операции, переводы, категории, долги, счета, бюджет.

**Home-screen виджеты:** быстрые действия, баланс, бюджет, «Скоро», один счёт — см. [android-client-platform.md](android-client-platform.md#home-screen-виджеты).

## MVP-навигация

Все пункты drawer (`nav-items.ts`) открывают существующие маршруты: главная, счета, операции, долги, кредиты, бюджет, статистика, настройки (хаб), админка (хаб). Подробнее — [android-client-ui.md](android-client-ui.md).
