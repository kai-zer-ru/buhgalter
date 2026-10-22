# Будущие обязательства при архивации / удалении счёта

Планируется в **v1.10.0** ([ROADMAP](../ROADMAP.md#v1100)). Канон после реализации — расширить [docs/accounts-archive-delete.md](../docs/accounts-archive-delete.md).

Связанные документы: [accounts-archive-delete.md](../docs/accounts-archive-delete.md), [subscriptions.md](../docs/subscriptions.md), [balance-maintenance.md](../docs/balance-maintenance.md), [ui-credits.md](../docs/ui-credits.md), [data-model.md](../docs/data-model.md), [transactions-display.md](../docs/transactions-display.md).

---

## Проблема

При архивации и мягком удалении счёта меняется только `accounts.status` (плюс перевод остатка для `cash`/`bank` и отключение auto-topup). Документация обещает: на архивном / удалённом счёте **новые операции недоступны**.

Фактически:

- пользовательский create/update по API отклоняет неактивный счёт (`ERR_ACCOUNT_ARCHIVED` и аналоги);
- **scheduler / ApplyDue** по-прежнему создаёт операции на `archived` / `deleted` для подписок, периодических и автоплатежей кредита;
- уже существующие `kind=future` активируются по дате без проверки статуса счёта;
- `forecast_balance` учитывает эффекты по `account_id` / `debit_account_id` без фильтра `accounts.status`.

Итог: после «закрытия» счёта баланс и прогноз могут снова двигаться от плановых сущностей — это баг относительно продукта и дыра в lifecycle счёта.

---

## As-is (на момент анализа)

Archive / soft-delete: `POST .../archive`, `DELETE .../{id}` → `inactivateAccount` → `SetStatus`. Жёсткого удаления строки `accounts` нет — FK `ON DELETE` не срабатывают.

| Сущность | FK на счёт | Что делает archive/delete сегодня | После inactivation |
|----------|------------|-----------------------------------|--------------------|
| **subscriptions** | `account_id` NOT NULL | ничего | `active=1` остаётся; ApplyDue списывает |
| **recurring_operations** | `account_id` NOT NULL | ничего | то же |
| **transactions `kind=future`** | `account_id` | ничего | остаются; ActivateDue активирует |
| **credits** | `debit_account_id` (+ опц. `down_payment_account_id`) | ничего | auto-pay пишет на неактивный счёт |
| **debts** | нет колонки; счёт в open/settle tx | ничего | settle с `affects_balance` требует active |
| **budgets** | опц. `account_id` | ничего | scope остаётся; create/update на inactive — 400 |
| **transaction_templates** | опц. `account_id` / `to_account_id` | ничего | stale FK до правки (`ON DELETE SET NULL` только при hard delete) |
| **category_rules** | `default_account_id` | ничего | suggest может вернуть archived |
| **auto_topup** | source / beneficiary | **`enabled=0`** | порог/цель/source колонки не чистятся |
| **credit_card.payment_account_id** | на cash/bank | ничего | может указывать на archived |
| **notification intercept bindings** | `account_id` | ничего | строки остаются (CASCADE только при hard DELETE) |

Ключевые файлы:

- `server/internal/httpserver/account_inactive_transfer.go`
- `server/internal/account/account.go` (`SetStatus`, `disableAutoTopupForInactive`)
- `server/internal/subscription/subscription.go` (`ApplyDue`)
- `server/internal/recurring/recurring.go` (`ApplyDue`)
- `server/internal/credit/credit.go` (`ApplyDuePayments`)
- `server/internal/accountbalance/scheduled.go` (forecast)

Единственный уже существующий паттерн «cleanup при inactive» — [auto-topup](../docs/balance-maintenance.md): сброс `enabled`.

---

## Целевая политика (решение)

**Вариант:** один confirm-диалог при archive / delete (расширение текущего flow с `transfer_to_account_id`).

### Принципы

1. Archive и soft-delete — **одна** политика разрешения обязательств.
2. По умолчанию будущие обязательства **переезжают вместе с деньгами** на счёт-приёмник (тот же, что для остатка).
3. Пользователь может вместо rebind выбрать **приостановить** (pause) то, что pause допускает.
4. **Кредиты** — только rebind `debit_account_id` (и связанных account FK); тихое pause запрещено.
5. Unarchive **не** откатывает rebind и **не** воскрешает сущности, которые пользователь приостановил.
6. Защита в глубину: даже если UI/API ошиблись, ApplyDue / ActivateDue / forecast **игнорируют** счета со `status != active`.

### Счёт-приёмник по умолчанию

Тот же, что сегодня для перевода остатка:

- если есть `transfer_to_account_id` (остаток > 0) — он;
- иначе основной (`is_primary`);
- если архивируется/удаляется основной — первый активный (`defaultTransferAccountId` в UI).

### Матрица действий

| Сущность | Rebind (default) | Pause (альтернатива в диалоге) | Без target |
|----------|------------------|--------------------------------|------------|
| Подписки (`active=1`) | `account_id` → target | `active=0` | нельзя завершить inactive, пока есть активные |
| Периодические | то же | то же | то же |
| Плановые операции (`kind=future`, ещё не активированные) | `account_id` → target | отмена / снятие future (эквивалент pause) | то же |
| Кредиты (`status=active`) с `debit_account_id` = этот счёт | обязательный rebind `debit_account_id` (+ при необходимости `down_payment_account_id`, если указывает сюда) | **запрещён** | блок: без target нельзя |
| Долги | нет stored account; в диалоге — информативный hint | — | settle позже только на active |
| Бюджеты с `account_id` | rebind scope **или** очистить (`NULL` = все счета) — default: clear scope | — | не блокируют |
| Шаблоны / category_rules | очистить FK (как `ON DELETE SET NULL`) | — | не блокируют |
| Auto-topup | уже `enabled=0` | — | — |
| `payment_account_id` карты | если указывает на этот счёт — clear или rebind | — | не блокирует archive карты (карта и так только при полном погашении) |
| Intercept bindings | clear / rebind на target | — | не блокируют |

«Нельзя завершить inactive» = API возвращает 400 с кодом вроде `ERR_ACCOUNT_HAS_FUTURE_OBLIGATIONS` и counts, пока клиент не передаст разрешение (`rebind` + `target` или `pause` где разрешено).

---

## UX

Точки входа те же: меню «⋯» на `/accounts` и `/accounts/[id]` — см. [ui-row-actions](../docs/ui-row-actions.md), [ui-dialogs](../docs/ui-dialogs.md).

Расширить `promptArchiveAccount` / `promptDeleteAccount` (+ Android-аналоги):

1. Как сейчас: при остатке — выбор счёта-приёмника.
2. Если есть будущие обязательства — блок сводки:
   - N подписок, M периодических, K плановых, C кредитов;
   - radio / сегмент: «Перенести на счёт …» (default) / «Приостановить» (скрыт или disabled, если есть кредиты без rebind);
   - при кредитах — обязательный target, без опции pause.
3. Подтверждение → один запрос archive/delete с параметрами разрешения.

Web и Android — одинаковая семантика; offline outbox для archive уже есть — параметры rebind/pause должны уезжать в том же payload.

---

## API (спека, без реализации в этом доке)

Расширить `POST /accounts/{id}/archive` и `DELETE /accounts/{id}`:

| Поле | Смысл |
|------|--------|
| `transfer_to_account_id` | как сейчас — остаток |
| `future_action` | `rebind` \| `pause` (default `rebind`, если есть обязательства) |
| `future_target_account_id` | куда rebind; может совпадать с `transfer_to_account_id` |

Либо отдельный preview:

`GET /accounts/{id}/future-obligations` → counts + ids для UI.

Ошибки:

- `ERR_ACCOUNT_HAS_FUTURE_OBLIGATIONS` — нет action / нет target при необходимости;
- `ERR_ACCOUNT_ARCHIVED` — target не active;
- существующие: `ERR_ACCOUNT_TRANSFER_REQUIRED`, `ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID`.

Всё в **одной** DB-транзакции: перевод остатка (если нужен) + rebind/pause + `SetStatus`.

---

## Серверные инварианты

1. `ListDueSubscriptions` / `ListDueRecurringOperations` / credit due / `ActivateDueFutureTransactions` — join или проверка `accounts.status = 'active'`; иначе skip (не писать tx, для подписок/recurring — **не** сдвигать `next_run_at` вслепую; для кредита — не apply payment).
2. Forecast (`ScheduledEffectsByUser` и аналоги) — не суммировать эффекты на inactive accounts.
3. Suggest default account (category rules) — только `active`.
4. После успешного inactive: auto-topup как сейчас (`enabled=0`).

---

## Unarchive

`POST .../unarchive` → `status=active`.

- Rebind **не** откатывается.
- Сущности с `active=0` после pause пользователя **остаются** выключенными.
- Пользователь сам включает подписки/периодические и правит счёт при необходимости.

---

## Тесты (чеклист на реализацию)

- Integration: archive счёта с активной due-подпиской без action → 400 + counts.
- Archive + `future_action=rebind` → подписка на target; следующий ApplyDue пишет на target.
- Archive + `pause` → `active=0`; ApplyDue не трогает.
- Archive с активным кредитом + только pause → 400; + rebind debit → OK.
- После archive (даже «дырявого» старого состояния) ApplyDue **не** создаёт tx на `archived`/`deleted`.
- Forecast не показывает эффекты на inactive.
- E2E web: диалог показывает сводку и перенос; Android — тот же сценарий вручную / vitest где уместно.

---

## Вне scope

- Hard-delete счёта и физический CASCADE.
- Автовоскрешение paused при unarchive.
- Отдельная сущность «перенос всех обязательств» вне archive/delete.
- Цели накопления (`savings_goals`, v1.9) — учесть в той же матрице, когда фича появится (тот же rebind/pause).

---

## Порядок внедрения (когда дойдём)

1. Защита scheduler/forecast (можно раньше UX — сразу чинит дыру).
2. Preview / counts API + расширение archive/delete.
3. Web диалог → Android.
4. Тесты + перенос правил из этого файла в [docs/accounts-archive-delete.md](../docs/accounts-archive-delete.md).
