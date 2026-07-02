# API — статус пользователя

Техническая справка по модерации и блокировке учётных записей (v1.3.0).

Продуктовая спецификация: [roadmap/user-status.md](../../roadmap/user-status.md).

---

## Схема БД

Колонка `users.status` (`TEXT`, миграция `029_user_status.sql`):

| Значение | Описание |
|----------|----------|
| `active` | Полный доступ |
| `pending` | Ожидает активации администратором |
| `banned` | Доступ запрещён |

`CHECK (status IN ('active', 'pending', 'banned'))`. Существующие пользователи при миграции получают `active`.

## Начальные значения

| Сценарий | Статус |
|----------|--------|
| Миграция / setup / админ создаёт пользователя | `active` |
| `POST /auth/register` | `pending` |

## Доступ

### Регистрация

`POST /api/v1/auth/register` — пользователь создаётся со статусом `pending`. Сессия **не** выдаётся.

Ответ `201`:

```json
{
  "user": {
    "id": "...",
    "login": "...",
    "status": "pending",
    ...
  }
}
```

### Вход

`POST /api/v1/auth/login` — после проверки пароля:

| Статус | HTTP | Код ошибки |
|--------|------|------------|
| `pending` | 403 | `USER_PENDING_MODERATION` |
| `banned` | 403 | `USER_BANNED` |
| `active` | 200 | — |

### Middleware

`RequireAuth` и `RequireAPIToken` отклоняют запросы пользователей со статусом не `active` (коды те же). При бане сессии инвалидируются админским endpoint.

## Смена статуса (админ)

`PUT /api/v1/admin/users/{id}/status`

```json
{ "status": "active" }
```

или

```json
{ "status": "banned" }
```

Допустимые переходы:

- `pending` → `active` | `banned`
- `active` → `banned`
- `banned` → `active`

Ограничения:

- нельзя менять свой статус (`400`, `ERR_CANNOT_CHANGE_OWN_STATUS`);
- нельзя установить `pending` через API (`400`, `USER_STATUS_INVALID`);
- недопустимый переход — `400`, `USER_STATUS_TRANSITION`.

При переходе в `banned` все сессии пользователя удаляются. Audit: `admin.user.status`.

Поле `status` возвращается в `GET /admin/users`, `POST /admin/users`, `GET /auth/me`.

## Уведомления администраторам

При `POST /auth/register` — отправка в Telegram/MAX всем админам с настроенным каналом и включённым триггером `user_registration`. Регистрация в системе должна быть разрешена (`registration_enabled`).

По образцу [`password_reset`](../../server/internal/notify/service.go) (`NotifyAdminsOnPasswordReset`).

Триггер `user_registration` и шаблон в настройках уведомлений доступны **только администраторам** и **скрываются в UI**, если самостоятельная регистрация отключена. API не возвращает шаблон и не принимает изменения триггера/шаблона при `registration_enabled = 0`.

### Триггер и шаблон

| Поле | Значение |
|------|----------|
| `trigger_type` | `user_registration` |
| Доступ | только администраторы (настройка и шаблон в UI) |
| Видимость в UI | только при `system_settings.registration_enabled = 1` |
| `notification_settings` | `trigger_user_registration` (default `1`) |

Плейсхолдеры:

| Плейсхолдер | Описание |
|-------------|----------|
| `login` | Логин |
| `display_name` | Имя |
| `registered_at` | RFC3339 → локальное время админа |
| `moderation_url` | `{external_url}/admin/users?moderate={user_id}` |

Если `system_settings.external_url` пуст — `moderation_url` не заполняется (в дефолтном шаблоне — пояснение про настройку URL).

Миграция: `030_notification_user_registration.sql` (расширение `CHECK` в `notification_templates`, колонка в `notification_settings`).

Дефолты локалей: `notifications.templates.user_registration` в [`server/locales/`](../../server/locales/).

### Дедупликация

`entity_id` = `user_id` нового пользователя, `date_key` = календарный день — одно уведомление на админа/канал/день (как у `password_reset`).

## Плашка и попап модерации

### Плашка

Компонент `AdminPendingUsersBanner` (аналог [`AdminPasswordResetBanner`](../../web/src/lib/components/AdminPasswordResetBanner.svelte)):

- видна только `$user?.is_admin`;
- данные: `GET /admin/users` → фильтр `status === 'pending'`;
- текст: i18n `admin.userModeration.notice`;
- кнопка → `/admin/users?moderate={user_id}`.

Монтирование: [`web/src/routes/+layout.svelte`](../../web/src/routes/+layout.svelte).

### Попап

На [`/admin/users`](../../web/src/routes/admin/users/+page.svelte) query `moderate={id}`:

- `ModalShell` с логином/именем;
- **Активировать** → `PUT .../status` `{ "status": "active" }`;
- **Заблокировать** → confirm + `{ "status": "banned" }`;
- после смены статуса плашка обновляется автоматически (пользователь больше не `pending`).

Отдельная очередь в БД не требуется.

## Ошибки на фронтенде

Для **регистрации, входа и админки пользователей** — `formatAuthUserApiError` (`web/src/lib/auth/api-errors.ts`):

- при кодах `VALIDATION_ERROR` / `CONFLICT` показывается конкретный `error.message` с сервера (например «Логин уже занят»), а не общая «Ошибка валидации»;
- при `error.field` — подпись поля (`Логин: …`, `Пароль: …`) или вывод у соответствующего поля на `/register`;
- коды вроде `USER_BANNED`, `USER_PENDING_MODERATION` — через i18n `errors.*`.

Остальные экраны по-прежнему используют общий `formatApiError`.

## Смежные документы

- [Аутентификация](authentication.md)
- [OpenAPI](openapi.yaml)
- [Модель данных](../data-model.md)
- [UI — меню «⋯»](../ui-row-actions.md)
- [Сброс пароля](authentication.md#сброс-пароля-v11) — аналогичный паттерн уведомлений и плашки
- [Telegram / MAX — кнопки и ссылки](../../roadmap/telegram-max-buttons.md) — `external_url` для `moderation_url`
