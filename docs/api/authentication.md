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
| `GET` | `/api/v1/user/tokens` | Список API-токенов |
| `POST` | `/api/v1/user/tokens` | Создать API-токен |
| `DELETE` | `/api/v1/user/tokens/{id}` | Отозвать API-токен |

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
