# UX. Android: «Принять» в шторке → форма операции (не silent create)

Статус: **исправлен** (часть [Улучшение уведомлений в шторке](notification-intercept.md#quick-actions-в-шторке-v160)). Пункт в [ROADMAP](../ROADMAP.md#v190).

## Зачем

Кнопка **Принять** при полном черновике молча создавала операцию. Снова открывает форму с prefill (как тап по телу), чтобы проверить магазин, категорию и сохранить явно.

## Поведение

| Действие | Цель |
|----------|------|
| Принять (расход/доход) | Открыть форму операции с prefill; черновик до Save |
| Принять (пара перевода) | Открыть форму перевода с prefill |
| Отклонить | Без изменений (удалить черновик, без Activity) |
| Тап по телу | Без изменений (форма) |

## Ручной чеклист (CAP-4)

На устройстве, тумблер «Уведомления в шторке» включён:

1. Полный черновик расхода (счёт есть) → **Принять** → форма `/transactions/new` с prefill; Save создаёт операцию; до Save черновик в списке.
2. Полный черновик дохода → то же.
3. Пара перевода (оба счёта) → **Принять** → форма `/transfers/new` с prefill; не silent create.
4. Черновик без счёта → **Принять** → форма (как раньше).
5. **Отклонить** → черновик удалён, Activity не открывается.
6. Тап по телу → форма (без регрессии).

JVM-тестов на `DraftNotifyReceiver` нет; silent-POST ветка Accept удалена.

## BMAD

- Спека: [SPEC.md](../bmad_docs/specs/spec-shade-accept-open-form/SPEC.md)
- Анализ: [analysis.md](../bmad_docs/specs/spec-shade-accept-open-form/analysis.md)
- Oneshot build: [spec-shade-notification-improvements.md](../bmad_docs/implementation-artifacts/spec-shade-notification-improvements.md)

Связано: [notification-merchant-strip.md](notification-merchant-strip.md), [shade-accept-sync-failed.md](shade-accept-sync-failed.md).
