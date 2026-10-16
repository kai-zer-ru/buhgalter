# БАГ. Android: в «Магазин» попадает почти весь текст push

Статус: **исправлен** (часть [Улучшение уведомлений в шторке](notification-intercept.md#quick-actions-в-шторке-v160)). Вошло в **v1.9.0** — [ROADMAP](../ROADMAP.md#v190), [release-notes](../docs/release-notes-v1.9.0.md).

## Симптомы

При создании расхода из черновика перехвата поле **Магазин** заполнялось мусором вроде `Оплата р Карта MAGNIT GM Petro Бала…` вместо имени точки (`MAGNIT GM Petro`). Сумма и тип операции корректны.

## Причина

`extractMerchant` (`parsers.ts`): после расширения голого суффикса `р` как валюты (income-patterns) сумма снималась без валютного маркера → сирота `р`; `\\b` не работал с кириллицей → не снимались «Оплата»/«Карта»; хвост «Баланс…» оставался.

## Фикс

Cleanup в `extractMerchant` (сумма+валюта одним куском, orphan currency, кириллице-безопасные границы, хвост Баланс/Доступно) + жёсткие vitest M1–M11. Голый `р` для kind/amount не откатывался.

## BMAD

- Спека: [SPEC.md](../bmad_docs/specs/spec-notification-merchant-strip/SPEC.md)
- Анализ: [analysis.md](../bmad_docs/specs/spec-notification-merchant-strip/analysis.md)
- Фикстуры/тесты: [fixtures.md](../bmad_docs/specs/spec-notification-merchant-strip/fixtures.md)
- Oneshot build: [spec-shade-notification-improvements.md](../bmad_docs/implementation-artifacts/spec-shade-notification-improvements.md)

## Связано

- UX Accept → форма: [shade-accept-open-form.md](shade-accept-open-form.md)
