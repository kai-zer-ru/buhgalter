# Webhook (исходящие события)

Не входит в v1. Черновик для обсуждения.

## Зачем

Сейчас интеграции **входящие**: API по токену, Telegram/MAX **исходящие уведомления** с сервера. Webhook — чтобы внешние системы (n8n, Zapier, свой скрипт, [Home Assistant](home-assistant.md)) реагировали на события в бухгалтерии без polling `GET /transactions`.

## События (черновик)

| Событие | Когда |
|---------|--------|
| `transaction.created` | новая операция |
| `transaction.updated` | изменение |
| `transaction.deleted` | удаление |
| `transfer.created` | перевод |
| `debt.created` / `debt.settled` | долг |
| `credit.payment_due` | приближается платёж по займу |
| `account.balance_low` | баланс ниже порога (опционально) |

Пейлоад — JSON, версия схемы в заголовке `X-Buhgalter-Event-Version`.

## Настройка (UI)

- Настройки пользователя: URL webhook, секрет для HMAC-подписи, список подписанных событий
- Тестовая доставка «ping»
- Лог последних доставок (как `notification_log`)

## Доставка

- HTTP POST, timeout 10s, retry с backoff (3 попытки)
- Подпись: `HMAC-SHA256(secret, body)` в `X-Buhgalter-Signature`
- Идемпотентность: `event_id` UUID в теле; повтор при retry с тем же id

## Безопасность

- Только HTTPS URL (или явный opt-in для `http://` в dev)
- Секрет не показывать повторно после создания
- Rate limit исходящих на user

## Отличие от Telegram/MAX

| | Telegram/MAX | Webhook |
|---|--------------|---------|
| Формат | текст для человека | JSON для машины |
| Настройка | bot token, chat | URL + secret |
| Триггеры | долг, кредит, planned | шире, включая CRUD |

Можно переиспользовать очередь/воркер из notification pipeline ([stage_08](../stage_08_notifications.md)).

## Открытые вопросы

- [ ] Webhook на уровне user или system admin?
- [ ] Несколько URL на пользователя?
- [ ] Входящий webhook (банк пушит операции) — см. [bank-sync.md](bank-sync.md), не смешивать
