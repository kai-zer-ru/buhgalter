# Кнопки в сообщениях Telegram / MAX

Релиз **v1.9.0**. Исходящие кнопки в уведомлениях + конструктор на web. Webhook / обработка callback — **отдельно, позже**. CAP-6 (`internal_url` / LAN fallback) — **не в этом релизе**.

Brief для BMAD: [`bmad_docs/planning-artifacts/telegram-max-buttons-brief.md`](../bmad_docs/planning-artifacts/telegram-max-buttons-brief.md). Overnight-промпт: [`bmad_docs/planning-artifacts/telegram-max-buttons-overnight-prompt.md`](../bmad_docs/planning-artifacts/telegram-max-buttons-overnight-prompt.md).

## Зачем

Сейчас уведомления — **текст** по шаблонам ([docs/notifications.md](../docs/notifications.md)). Кнопки в Telegram/MAX ускоряют переход в приложение и готовят почву для будущих действий из чата (без inbound в этом релизе).

## Scope v1.9.0 (зафиксировано)

### Делаем

- Исходящие **inline-кнопки** для Telegram и MAX (a161 + official).
- **Две независимые клавиатуры** (модель и UI): Telegram и MAX — разные типы по официальным докам.
- **Global-клавиатура** (на каждое исходящее) + **кнопки в шаблоне** триггера; при send — **concat рядов**: `global → trigger → balance_shortfall` (если shortfall дописан).
- Конструктор на **web** `/settings/notifications`: drag-and-drop кнопок мышью/пальцем, тип, текст/payload, ряды.
- Пресеты «добавить кнопку» + свободная раскладка внутри редактора (global и template — **два редактора**, DnD только внутри своего).
- Плейсхолдеры в URL/link как в тексте шаблонов (`{debt_url}`, `{credit_url}`, …) на базе **`system_settings.external_url`**.
- Кнопки в **тестовом** сообщении.
- Админ-триггеры (`password_reset`, `user_registration`) — тот же механизм.
- Preview текста **и** resolved keyboard (отдельно Telegram / MAX).
- Android `/settings/notifications` → **только баннер** «Настройте в веб-версии» (каналы, шаблоны, тест — только web).
- Хранение: JSON в `notification_settings` (global tg/max) + JSON у `notification_templates` (per trigger, tg/max).
- Расширить `Notifier.Send` (или эквивалент) — текст + optional keyboard; маппинг в Bot API / MAX `inline_keyboard`.

### Не делаем

- Webhook, long polling, `answerCallbackQuery` / MAX `answers`, бизнес-действия по нажатию.
- CAP-6 / `internal_url` / авто LAN-base для ссылок (остаётся в [общих планах](../ROADMAP.md#общие-планы)).
- Полный app-open-links / custom scheme для notify.
- Commit / push без явной просьбы владельца.

## Типы кнопок

### Telegram ([InlineKeyboardButton](https://core.telegram.org/bots/api#inlinekeyboardbutton))

Ровно одно действие + `text` (и опциональные style/icon по доке, если поддерживаем в модели):  
`url` | `callback_data` | `web_app` | `login_url` | `switch_inline_query` | `switch_inline_query_current_chat` | `switch_inline_query_chosen_chat` | `callback_game` | `pay` | `copy_text`.

### MAX ([клавиатура](https://dev.max.ru/docs-api/use-cases/sending-messages/keyboard))

`callback` | `link` | `message` | `open_app` | `request_contact` | `request_geo_location` | `clipboard`.

a161 (`notify.a161.ru`) — тот же формат attachments, что official (замена base URL).

### Лимиты

Валидация **per provider** (лимиты Telegram Bot API / MAX docs). UI не даёт сохранить заведомо невалидную раскладку для выбранного канала.

## URL / `external_url`

- База ссылок — только **`external_url`**.
- Если после резолва у кнопки типа url/link (и аналогов, требующих URL) URL пуст — **эту кнопку не включать** в исходящее; остальные кнопки оставить (**T1**).
- Если после фильтра рядов клавиатура пуста — **не** передавать `reply_markup` / `inline_keyboard` attachment.
- В тексте шаблонов поведение hint при пустом `external_url` **не менять** в этом релизе (CAP-6 отдельно).

## UI (web)

- Блок **«Кнопки по умолчанию»** (global): вкладки Telegram | MAX; DnD, ряды, тип, поля по типу; default **пустой**.
- В карточке каждого шаблона — **«Добавить кнопку»** / редактор keyboard для TG и MAX (тот же виджет).
- Пресеты (вставить типовую link-кнопку с плейсхолдером пути), не вместо свободного конструктора.
- Preview: текст + схема кнопок для TG и MAX.
- Переиспользовать подходы DnD из `$lib/drag-reorder` (категории), адаптировать под сетку рядов.

## Android

Страница `/settings/notifications`: убрать полный `NotificationsTab`; оставить баннер с i18n «Настройте уведомления в веб-версии» (+ ссылка/пояснение при необходимости). Пункт меню можно оставить (ведёт на баннер) или скрыть — **рекомендация агента: оставить пункт + баннер**, чтобы не терять discoverability feature flag.

## API / данные (ориентир)

- `GET`/`PUT` `/api/v1/user/notifications` — поля global keyboards + per-template keyboards.
- Preview endpoint — возвращать resolved text **и** resolved keyboards (tg/max) либо отдельный preview-кнопок.
- OpenAPI + `make copy-openapi` / openapi-check.
- Миграция goose: JSON-колонки (или TEXT JSON); sqlc.
- i18n web + android (баннер); `make ui-i18n-check`.

## Безопасность

- В URL кнопок — без API-токенов; только публичные path/id + сессия в браузере.
- Callback payload в этом релизе — как ввёл пользователь (без подписи/TTL); обработка — позже.
- Не логировать bot tokens.

## Тесты

- Server: маппинг keyboard → Telegram/MAX payload; фильтр url при пустом `external_url`; merge global+template+shortfall; валидация лимитов.
- Web unit: модель рядов / DnD helpers при выносе в `$lib`.
- Web e2e: настройки уведомлений — добавить кнопку, сохранить, preview; Android — баннер на странице.
- Не гонять `make test*` без просьбы владельца; **написать** тесты в той же сессии.

## Out of scope / later

- Inbound webhook + действия («погасил»).
- CAP-6 internal URL в плейсхолдерах.
- Смешивание кнопок global и template в одном canvas (J′2) — отложено; сейчас J′1.

## Решения владельца ( Melog )

| ID | Решение |
|----|---------|
| A | Две клавиатуры TG / MAX |
| I | Global default пустой |
| J | Два редактора; concat при send; DnD внутри зоны |
| K | Merge shortfall-кнопок: `global → trigger → balance_shortfall` |
| L | Админ-триггеры участвуют |
| M | JSON в settings + templates |
| N | Preview TG и MAX |
| D | Android — только баннер |
| F | Все типы по докам |
| G | Лимиты per provider |
| T | Drop только url/link без resolved URL |
| CAP-6 | Не в 1.9.0 |
| Git | Без commit/push |
| Автономия | При споре — рекомендация из brief + лог в `bmad_docs/` |

**Зависимости:** [docs/notifications.md](../docs/notifications.md), `external_url`, `server/internal/notify/`, web `NotificationsTab`, Android notifications route.
