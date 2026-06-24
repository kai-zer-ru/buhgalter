# Home Assistant

Не входит в v1. Черновик для обсуждения.

## Зачем

Пользователи умного дома хотят видеть баланс, долги или триггерить сценарии («зарплата пришла» → свет в кабинете). REST API уже есть; интеграция HA упрощает настройку без своих скриптов.

## Референс в проекте

Уведомления **MAX** реализованы с опорой на [max-notyfy-ha](https://github.com/kai-zer-ru/max-notyfy-ha) ([stage_08](../stage_08_notifications.md)): провайдер `a161`, домен `notify.a161.ru`. Для HA логично тот же стиль: **custom component** в HACS + документация.

## Варианты интеграции

**A. Официальная custom integration (предпочтительно)**

- Python package `custom_components/buhgalter/`
- Config flow: URL сервера, API token (из настроек пользователя)
- Sensors: `sensor.buhgalter_total_balance`, `sensor.buhgalter_debt_lent`, `sensor.buhgalter_debt_borrowed`, по счетам — `sensor.buhgalter_account_<id>`
- Binary sensor: «есть просроченный долг», «платёж по кредиту завтра»
- Update interval: polling 5–15 мин (не нагружать self-hosted)

**B. Только REST + документация**

- Примеры `rest` / `template` в `docs/integrations/home-assistant.md`
- Быстрее, но хуже UX настройки

**C. Через [webhooks](webhooks.md)**

- HA ловит события; sensors обновляются по push — меньше polling, нужен webhook roadmap

## API, которые понадобятся

Уже есть (проверить полноту для HA):

- `GET /api/v1/dashboard` или балансы счетов
- `GET /api/v1/debts` (активные)
- `GET /api/v1/credits` (ближайший платёж)
- Verify token

Может понадобиться лёгкий `GET /api/v1/summary` — один ответ для всех sensors (меньше запросов).

## Безопасность

- Long-lived API token с минимальными правами (read-only scope — если добавим)
- Локальная сеть: HA → buhgalter по `http://nas:8765` без выхода в интернет
- Не логировать token в HA automations в plain text

## Действия (actions) — позже

- Не создавать операции из HA в v1 интеграции (риск случайных трат)
- Только read sensors + notify; write — отдельное обсуждение

## Открытые вопросы

- [ ] Репозиторий интеграции: в монорепе `integrations/homeassistant/` или отдельный repo?
- [ ] Публикация в HACS
- [ ] Связь с Android [клиентом](android-client.md) — независимые каналы
