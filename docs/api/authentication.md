# API — аутентификация

Справка по сессиям, API-токенам и правилам доступа.

---

## Механизмы авторизации

- **Session cookie** (`session`) — основной способ для web UI.
- **Bearer token** — для API-клиентов и Android (`Authorization: Bearer ...`): session token после `POST /auth/login` или долгоживущий API-токен.
- **API tokens** — отдельные пользовательские токены, создаются в настройках (в т.ч. вход в Android через `/login/token`).

## Основные endpoints

| Метод | Путь | Описание |
|------|------|----------|
| `POST` | `/api/v1/auth/login` | Вход, выдаёт session + token в ответе |
| `POST` | `/api/v1/auth/logout` | Выход и инвалидирование сессии |
| `POST` | `/api/v1/auth/register` | Регистрация (если включена); статус `pending`, без сессии |
| `GET` | `/api/v1/auth/verify` | Проверка валидности токена |
| `GET` | `/api/v1/auth/me` | Текущий пользователь |
| `POST` | `/api/v1/auth/request-password-reset` | Запрос сброса пароля (body: `{ "login" }`; ответ `204`) |
| `GET` | `/api/v1/admin/password-reset-requests` | Ожидающие запросы (админ) |
| `POST` | `/api/v1/admin/password-reset-requests/{id}/ack` | Скрыть запрос из очереди (админ) |
| `PUT` | `/api/v1/admin/users/{id}/password` | Задать новый пароль пользователю (админ) |
| `GET` | `/api/v1/ui/meta` | Агрегированные справочники для старта UI |
| `GET` | `/api/v1/user/tokens` | Список API-токенов |
| `POST` | `/api/v1/user/tokens` | Создать API-токен |
| `DELETE` | `/api/v1/user/tokens/{id}` | Отозвать API-токен (только свой) |

### API-токены

- По умолчанию срок действия — **30 дней** с момента создания.
- `never_expires: true` — бессрочный токен (не рекомендуется; UI показывает предупреждение).
- `expires_at` (RFC3339) — явная дата истечения; приоритетнее срока по умолчанию, но игнорируется при `never_expires`.
- Отозвать можно только **свой** токен; попытка удалить чужой — `404 NOT_FOUND`.
- Полное значение токена возвращается **один раз** в ответе `POST /user/tokens` (поле `token`).
- В UI (`/settings/tokens`) при выборе явной даты истечения календарь блокирует **сегодня и прошлые дни** — минимум завтра в часовом поясе пользователя (`futureDateOnlyPicker`, см. [date-time-display.md](../date-time-display.md)).

Пример создания бессрочного токена:

```json
{ "name": "Home Assistant", "never_expires": true }
```

Пример с явной датой:

```json
{ "name": "Script", "expires_at": "2026-12-31T23:59:59Z" }
```

## Сброс пароля (v1.1)

Self-service смены пароля по e-mail **нет**. Сценарий для self-hosted:

1. Пользователь: `POST /api/v1/auth/request-password-reset` с логином. Ответ всегда `204` (не раскрывает, существует ли учётка). Rate limit: 5 запросов/мин с IP.
2. Администратор: `GET /api/v1/admin/password-reset-requests` — список ожидающих; `POST .../ack` — скрыть запрос из очереди.
3. Администратор: `PUT /api/v1/admin/users/{id}/password` — новый пароль; сессии пользователя инвалидируются.

Схемы в [openapi.yaml](openapi.yaml): `PasswordResetRequest`, пути выше + `GET /ui/meta` (`UIMetaResponse`).

Таблица БД: `password_reset_requests` (миграция `023_password_reset_requests.sql`).

## Статус пользователя (v1.3.0)

Модерация регистрации и блокировка учётных записей — см. [user-status.md](user-status.md).

## Ошибки

Стандартный формат:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Некорректные параметры запроса",
    "field": "monthly_payment"
  }
}
```

Поле **`field`** опционально: заполняется для ошибок валидации (`ERR_CREDIT_*`, `ERR_INVALID_*`, `ERR_ACCOUNT_*` и аналогичных), чтобы UI мог подсветить конкретное поле формы.

Коды ошибок и схемы — в [openapi.yaml](openapi.yaml) (в т.ч. `/stats/*`, `POST /credits/schedule/preview`).

## Смежные документы

- [OpenAPI](openapi.yaml)
- [UI диалоги и confirm](../ui-dialogs.md)
