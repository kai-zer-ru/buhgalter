# БАГ. Android: Accept в шторке → «Не удалось отправить часть операций»

Статус: **исправлен** (2026-10-16). Пункт в [ROADMAP](../ROADMAP.md#v181).

Фикс: native `DraftNotifyHttp.toApiTransactionDate` (POST + offline payload) + JS `toApiUtcDateTime` в `applyNativeDraftNotifySync`; vitest на ISO→Layout.

## Симптомы

После **Принять** в уведомлении черновика банка (шторка) в приложении:

- баннер «Не удалось отправить часть операций» / «Повторить» (`offline.syncFailed`);
- у операции ⚠️ (failed outbox).

Если ту же операцию открыть и сохранить в форме приложения — синхронизация проходит.

## Причина

Shade Accept (`DraftNotifyHttp`) отправляет `transaction_date` в RFC3339 (ISO из `occurredAt`). API `POST /transactions` / `/transfers` принимает только `YYYY-MM-DD HH:MM:SS` (`timeutil.ParseUTC`). Ответ 400 → native fallback в JS outbox с тем же ISO → `markOutboxFailed`.

Форма использует `fromDatetimeLocalValue` → Layout, поэтому правка помогает.

## Фикс

Конвертация даты на границе native/JS перед POST и в offline payload (`DraftNotifyHttp.toApiTransactionDate`, `toApiUtcDateTime` в `applyNativeDraftNotifySync`). Ручная проверка Accept на устройстве — после сборки APK.

## BMAD

- Спека: [SPEC.md](../bmad_docs/specs/spec-shade-accept-date-format/SPEC.md)
- Анализ: [analysis.md](../bmad_docs/specs/spec-shade-accept-date-format/analysis.md)
- Промпт для нового чата: [DEV_PROMPT.md](../bmad_docs/specs/spec-shade-accept-date-format/DEV_PROMPT.md)

## Примечание по релизу

Подготовка к релизу v1.8.1 (CHANGELOG / release notes и т.п.) отложена: `git stash` с сообщением `wip: release prep v1.8.1 (отложено из-за бага шторки Accept)`.
