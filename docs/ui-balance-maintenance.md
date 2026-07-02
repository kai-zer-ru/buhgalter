# UI — автопополнение счёта

Связанные документы: [balance-maintenance.md](balance-maintenance.md), [ui-dialogs.md](ui-dialogs.md), [ui-row-actions.md](ui-row-actions.md), [ui-toast.md](ui-toast.md).

---

## Доступность

Только для активных счетов `type = bank`. Для `cash` и `credit_card` пункт меню и индикатор **не показываются**.

---

## Индикатор на главной

В карточке счёта (`+page.svelte`), под балансом:

- `Автопополнение: {имя счёта списания}` — включено (имя счёта-источника);
- при выключенном автопополнении строка **не показывается**.

---

## Меню счёта

Пункт **«Автопополнение»** в `RowActionsMenu` на `/accounts` и `/accounts/[id]`.

---

## Диалог

`$lib/components/AccountAutoTopupDialog.svelte`

| Элемент | Поведение |
|---------|-----------|
| `ToggleSwitch` | главный переключатель |
| Счёт списания | `Select` — только `bank`, без текущего |
| Сумма порога / поддержания | `MoneyInput` |
| Сохранить | `PUT /accounts/{id}` |

Счёт по умолчанию в селекте: основной `bank`, иначе первый активный `bank`.

---

## i18n

Ключи `accounts.autoTopup.*` в `ru.json` / `en.json`.
