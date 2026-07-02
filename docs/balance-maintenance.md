# Поддержание баланса (автопополнение)

Автоматическое пополнение **банковского** счёта переводом с другого `bank`-счёта, когда баланс опускается ниже заданного порога.

Связанные документы: [ui-balance-maintenance.md](ui-balance-maintenance.md), [data-model.md](data-model.md), [transactions-display.md](transactions-display.md), [notifications.md](notifications.md), [accounts-archive-delete.md](accounts-archive-delete.md), [api/openapi.yaml](api/openapi.yaml).

---

## Назначение

Пользователь задаёт для счёта-получателя (`type = bank`):

- **порог** — ниже какой суммы срабатывает пополнение;
- **цель** — до какой суммы довести баланс;
- **счёт списания** — другой активный `bank`.

**Не поддерживается** для «Наличные» (`cash`) и кредитных карт (`credit_card`).

Пример: на счёте «Яндекс» порог 3 000 ₽, цель 5 000 ₽, списание с «Сбер». При балансе 2 500 ₽ создаётся перевод 2 500 ₽.

| | Уведомление «не хватает средств» | Автопополнение |
|---|----------------------------------|----------------|
| Действие | предупреждение в тексте | реальный перевод |
| Когда | перед плановым расходом | после фактического снижения баланса |
| Настройка | глобальный toggle | per-account |

---

## Условия срабатывания

1. `auto_topup_enabled = 1` на счёте-получателе.
2. Счёт-получатель: `status = active`, `type = bank`.
3. `current_balance < auto_topup_threshold` (копейки).
4. `transfer_amount = auto_topup_target - current_balance`, `transfer_amount > 0`.
5. На счёте списания: `current_balance >= transfer_amount`.
6. Оба счёта принадлежат пользователю, активны, разные.

Прогноз (`forecast_balance`) **не** учитывается.

---

## Создаваемая операция

- через `transaction.CreateTransfer`;
- категория **«Перевод»**;
- `description` = `Автопополнение` (фиксированно);
- `commission` = 0;
- `transaction_date` = текущий момент в timezone пользователя.

---

## Нехватка средств на счёте списания

Если на источнике недостаточно средств:

- перевод **не** создаётся;
- `auto_topup_enabled` сбрасывается в `0`;
- отправляется уведомление `auto_topup_disabled` (если включён toggle).

Повторное включение — только вручную в диалоге.

---

## Модель данных

Колонки `accounts` (миграция `041_account_auto_topup.sql`):

| Поле | Описание |
|------|----------|
| `auto_topup_enabled` | 1 = включено |
| `auto_topup_threshold` | порог, копейки |
| `auto_topup_target` | цель, копейки |
| `auto_topup_source_account_id` | FK на `bank`-счёт списания |

### Валидация API

- `threshold >= 0`, `target > 0`, `threshold < target`;
- получатель и источник — активные `bank`, разные;
- `PUT` с включённым автопополнением для `cash` / `credit_card` → `400` (`ERR_ACCOUNT_AUTO_TOPUP_NOT_ALLOWED`);
- при архивации/удалении получателя или источника — автопополнение сбрасывается.

---

## Когда проверяется

После `accountbalance.Refresh` в цепочках операций, переводов, периодики, кредитов, импорта, долгов — через `balancehooks.AfterRefresh` → `balancetopup.CheckAfterRefresh`.

Пакет: `server/internal/balancetopup/`.

---

## API

Поля в `Account`, `AccountBalanceSummary`, `GET /dashboard`:

- `auto_topup_enabled`, `auto_topup_threshold`, `auto_topup_target`, `auto_topup_source_account_id` (+ `*_display` для сумм).

`PUT /api/v1/accounts/{id}` принимает те же поля.

---

## Уведомления

| Toggle | `trigger_auto_topup_disabled` | Шаблон `auto_topup_disabled` |
|--------|-------------------------------|------------------------------|

Плейсхолдеры: `{account}`, `{source_account}`, `{amount}`, `{source_balance}`, `{account_url}`.
