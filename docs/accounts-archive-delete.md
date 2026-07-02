# Архивация и удаление счётов

Реализовано в v1.3.0.

Связанные документы: [data-model.md](data-model.md), [ui-dialogs.md](ui-dialogs.md), [ui-row-actions.md](ui-row-actions.md), [ui-credit-cards.md](ui-credit-cards.md), [transactions-display.md](transactions-display.md), [api/openapi.yaml](api/openapi.yaml).

---

## Статусы счёта

| `accounts.status` | Поведение |
|-------------------|-----------|
| `active` | обычная работа |
| `archived` | счёт в архиве; операции в истории; **новые операции недоступны**; вкладка «Архивные» на `/accounts` |
| `deleted` | мягкое удаление; операции и история сохраняются; вкладка «Удалённые»; просмотр по `/accounts/{id}` |

Архив и удаление **не удаляют** строки операций из БД.

---

## Правила по типу счёта

### `cash` / `bank`

| Действие | Остаток `balance > 0` | Остаток `balance = 0` |
|----------|------------------------|------------------------|
| Архивация | обязателен автоперевод остатка на другой активный счёт | простое подтверждение |
| Удаление | то же | простое подтверждение |

Перед сменой статуса сервер создаёт **перевод** всей суммы остатка:

- категория **«Перевод»**;
- комментарий `Архивация счёта "{name}"` или `Удаление счёта "{name}"`;
- параметр API `transfer_to_account_id` (query или JSON body).

Сумма перевода — `max(current_balance, пересчёт по операциям)`; в UI отображается через `MoneyDisplay`.

Без `transfer_to_account_id` при положительном остатке API вернёт `400` (`ERR_ACCOUNT_TRANSFER_REQUIRED`).

### `credit_card`

| Действие | Перевод | Условие |
|----------|---------|---------|
| Архивация / удаление | **не выполняется** | только при **полном погашении**: `balance >= credit_limit` |

При непогашенной карте — `400` (`ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID`). Подробнее о балансе карты — [ui-credit-cards.md](ui-credit-cards.md).

---

## API

| Метод | Назначение |
|-------|------------|
| `POST /api/v1/accounts/{id}/archive` | `status → archived` |
| `DELETE /api/v1/accounts/{id}` | `status → deleted` (soft-delete) |
| `POST /api/v1/accounts/{id}/unarchive` | восстановление из архива |

Оба endpoint (`archive`, `delete`) принимают опциональный `transfer_to_account_id` для `cash` / `bank` с остатком.

Схема и коды ошибок — [api/openapi.yaml](api/openapi.yaml).

---

## UI

### Подтверждение

- `$lib/accounts/account-inactive-prompt.ts` — `promptArchiveAccount()`, `promptDeleteAccount()`;
- `$lib/accounts/account-transfer-confirm.ts` + `AccountTransferConfirmDialog.svelte` — выбор счёта-приёмника при остатке;
- подключение в `routes/+layout.svelte`.

**Счёт по умолчанию** в селекте перевода: основной (`is_primary`); если архивируется/удаляется он же — первый в списке активных (`defaultTransferAccountId`).

Точки входа: меню «⋯» на `/accounts` и на `/accounts/[id]` — см. [ui-row-actions.md](ui-row-actions.md).

### Удалённый счёт (просмотр)

На `/accounts/{id}` при `status = deleted`:

- редактирование счёта и шапочные действия скрыты;
- в меню операций — только **«Повторить»** (без изменить / удалить / «Сделать периодической»);
- баннер `accounts.banner.deleted`.

В списке операций для счетов `archived` / `deleted` — суффиксы «(архив)» / «(удалён)» в `TransactionAccountCell` ([transactions-display.md](transactions-display.md)).

### i18n (основные ключи)

- `accounts.confirm.archive`, `accounts.confirm.delete`
- `accounts.confirm.archiveWithBalance.before` / `.after`, `accounts.confirm.deleteWithBalance.before` / `.after`
- `accounts.confirm.transferTo`, `accounts.confirm.inactiveNoTargets`
- `accounts.confirm.creditCardNotFullyPaid`
- `errors.ERR_ACCOUNT_TRANSFER_REQUIRED`, `errors.ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID`

Паттерн диалогов и Esc — [ui-dialogs.md](ui-dialogs.md).

---

## Сервер (реализация)

| Файл | Назначение |
|------|------------|
| `server/internal/httpserver/account_archive.go` | `POST .../archive` |
| `server/internal/httpserver/account_delete.go` | `DELETE .../{id}` |
| `server/internal/httpserver/account_inactive_transfer.go` | перевод остатка, `transfer_to_account_id` |
| `server/internal/account/account.go` | `SetStatus`, `validateCreditCardFullyPaid`, soft-delete |
| `server/internal/transaction/transfer.go` | `CreateTransferForAccountDelete` |

Миграция статуса `deleted`: `040_account_deleted_status.sql`.

---

## Тесты

- `server/internal/httpserver/accounts_integration_test.go` — архив/удаление с переводом, кредитная карта, drift `current_balance`
- `server/internal/account/account_delete_test.go` — `RequiresBalanceTransfer`, `validateCreditCardFullyPaid`
- `web/e2e/accounts-management.spec.ts` — архивация и удаление с выбором счёта-приёмника
- `web/src/lib/credit-card.test.ts` — `isCreditCardFullyPaid`
