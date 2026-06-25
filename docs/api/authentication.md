# API — аутентификация

Справка по сессиям, API-токенам и правилам доступа.

---

## Механизмы авторизации

- **Session cookie** (`session`) — основной способ для web UI.
- **Bearer token** — для API-клиентов (`Authorization: Bearer ...`).
- **API tokens** — отдельные пользовательские токены, создаются в настройках.

## Основные endpoints

| Метод | Путь | Описание |
|------|------|----------|
| `POST` | `/api/v1/auth/login` | Вход, выдаёт session + token в ответе |
| `POST` | `/api/v1/auth/logout` | Выход и инвалидирование сессии |
| `POST` | `/api/v1/auth/register` | Регистрация (если включена) |
| `GET` | `/api/v1/auth/verify` | Проверка валидности токена |
| `GET` | `/api/v1/auth/me` | Текущий пользователь |
| `POST` | `/api/v1/auth/request-password-reset` | Запрос сброса пароля (v1.1; body: `{ "login" }`) |
| `GET` | `/api/v1/user/tokens` | Список API-токенов |
| `POST` | `/api/v1/user/tokens` | Создать API-токен |
| `DELETE` | `/api/v1/user/tokens/{id}` | Отозвать API-токен |

## Сброс пароля (v1.1)

Self-service смены пароля по e-mail **нет**. Сценарий для self-hosted:

1. Пользователь: `POST /api/v1/auth/request-password-reset` с логином. Ответ всегда `200` (не раскрывает, существует ли учётка). Rate limit: 5 запросов/мин с IP.
2. Администратор: `GET /api/v1/admin/password-reset-requests` — список ожидающих; `POST .../ack` — скрыть запрос из очереди.
3. Администратор: `PUT /api/v1/admin/users/{id}/password` — новый пароль; сессии пользователя инвалидируются.

Таблица БД: `password_reset_requests` (миграция `023`).

Подробнее: [../release-notes-v1.1.md](../release-notes-v1.1.md).

## Ошибки

Стандартный формат:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "..."
  }
}
```

Коды ошибок и схемы — в [openapi.yaml](openapi.yaml).

## Смежные документы

- [OpenAPI](openapi.yaml)
- [UI диалоги и confirm](../ui-dialogs.md)
