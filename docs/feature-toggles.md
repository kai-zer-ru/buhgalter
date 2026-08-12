# Включение / отключение функционала

Администратор инстанса включает и выключает **опциональные модули**. Уровень — весь инстанс (не per-user). Данные при выключении **не удаляются**.

Планирование и история решений: [roadmap/feature-toggles.md](../roadmap/feature-toggles.md).

## Модель

- Реестр ключей в коде: `server/internal/features`.
- Overrides в БД: таблица `feature_flags` (миграция `050_feature_flags.sql`).
- Нет строки в БД → `default_enabled` из реестра.
- Неизвестный ключ в `PUT` → `400`.

Самостоятельная регистрация — флаг `registration` (раньше `system_settings.registration_enabled`). Публично читается как `registration_enabled` в `GET /setup/status`.

## API

| Метод | Путь | Кто |
|-------|------|-----|
| `GET` | `/api/v1/features` | авторизованный |
| `GET` | `/api/v1/admin/features` | admin (каталог + метаданные; UI — только web) |
| `PUT` | `/api/v1/admin/features` | admin (overrides; UI — только web) |

При выключенном модуле API отвечает **404** с кодом `FEATURE_DISABLED`.

## Каталог ключей

| Ключ | Default |
|------|---------|
| `registration` | `false` |
| `debts` | `true` |
| `credits` | `true` |
| `budget` | `true` |
| `subscriptions` | `true` |
| `recurring` | `true` |
| `balance_maintenance` | `true` |
| `import_export` | `true` |
| `stats` | `true` |
| `notifications` | `true` |
| `merchants_tags` | `true` |
| `transaction_templates` | `true` |

## Клиенты

Web и Android загружают снимок после логина (`isFeatureEnabled` / nav filter). Прямой URL выключенного модуля → `/feature-disabled`.

**Настройка флагов** — только в **web**-админке на `/admin/features`. В Android UI переключателей модулей нет (как и `/admin/users`).

## Правило для новых модулей

Опциональный модуль **не мержится** без:

1. Ключа в реестре + i18n title/description.
2. Миграции `INSERT OR IGNORE` (обычно enabled).
3. Строки в каталоге выше.
4. Гейтов API (`RequireFeature`) и scheduler/notify при необходимости.
5. Фильтра nav / роутов в web и Android.
6. Тестов (флаг off → `FEATURE_DISABLED`).

Подфичи внутри модуля отдельным флагом не плодить.
