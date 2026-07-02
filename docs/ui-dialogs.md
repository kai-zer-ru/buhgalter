# UI — диалоги, попапы и подтверждения

В интерфейсе **не используем** нативные `window.confirm`, `window.alert`, `window.prompt` — только модальные окна в разметке страницы (оверлей + карточка).

**Esc:** любой **попап** (модалка, выпадающий список, picker) закрывается по **Escape** — см. [Закрытие по Escape](#закрытие-по-escape).

## Компоненты

| Файл | Назначение |
|------|------------|
| `$lib/confirm.ts` | `confirm(options): Promise<boolean>` — API как у `window.confirm`, но с кастомным UI |
| `$lib/components/ConfirmDialog.svelte` | Глобальный диалог подтверждения |
| `$lib/accounts/account-transfer-confirm.ts` | `confirmAccountTransfer()` — архивация/удаление счёта с переводом остатка |
| `$lib/components/AccountTransferConfirmDialog.svelte` | Диалог архивации/удаления (выбор счёта-приёмника при `balance > 0`) |
| `$lib/accounts/account-inactive-prompt.ts` | `promptArchiveAccount()`, `promptDeleteAccount()` |
| `routes/+layout.svelte` | `<ConfirmDialog />` и `<AccountTransferConfirmDialog />` подключены один раз |

Информационные модалки (например, показ созданного API-токена) — отдельные блоки `{#if open}` на странице или будущий `$lib/modal`; принцип тот же: без системных alert.

## Использование

```typescript
import { confirm } from '$lib/confirm';
import { _ } from 'svelte-i18n';

async function remove() {
	const ok = await confirm({
		title: $_('common.confirm.title'), // опционально
		message: $_('accounts.confirm.delete'),
		confirmLabel: $_('common.delete'),
		cancelLabel: $_('common.cancel'),
		danger: true // красная кнопка подтверждения
	});
	if (!ok) return;
	// destructive action…
}
```

Клик по фону, «Отмена» или **Escape** → `false`.  
Кнопка подтверждения → `true`.

### Архивация и удаление счёта

Полное описание — [accounts-archive-delete.md](accounts-archive-delete.md). Кратко для диалогов:

- `cash` / `bank` с `balance > 0` — `promptArchiveAccount()` / `promptDeleteAccount()` и `AccountTransferConfirmDialog`;
- `credit_card` — без перевода; только при `balance >= credit_limit`; иначе информационный `confirm({ acknowledgeOnly: true })` с кнопкой «Закрыть`;
- сумма остатка — `MoneyDisplay`; счёт по умолчанию — основной.

Если других активных счетов нет — кнопка подтверждения неактивна, текст `accounts.confirm.inactiveNoTargets`.

## Закрытие по Escape

**Все попапы** приложения закрываются по **Esc** — единое правило для модалок, выпадающих списков и overlay-виджетов.

| Тип | Поведение |
|-----|-----------|
| `ConfirmDialog` | `false` (отмена) — уже реализовано |
| Формы (`TransactionForm`, `TransferForm`, `DebtForm`, `CreditForm`, …) | `onclose()` — отмена без сохранения |
| Информационные модалки (API-токен, preview и т.п.) | закрыть, без side effects |
| `CategoryIconPicker`, выбор иконки | закрыть без выбора |
| **DateTimePicker** (календарь) | закрыть попап; в режиме с временем — Esc не закрывает, если фокус в поле времени (см. ниже) |
| **Combobox / Select** (`TimezonePicker`, `Select`, `Combobox`) | закрыть список; если список уже закрыт — не перехватывать Esc у родительской модалки |
| Вложенные слои | закрывается **верхний** попап (confirm поверх формы — сначала confirm) |

**Не попап:** нативные `<select>` заменены на `Select` / `Combobox` — те же правила, что у combobox.

**Реализация:**

- Обработчик на `svelte:window` или `onkeydown` на оверлее: `e.key === 'Escape'` → закрытие верхнего слоя
- У combobox: Esc на открытом списке — только `open = false`, фокус остаётся в поле
- Не глотать Esc без действия: если виджет открыт — он должен на Esc закрыться

**Чеклист при добавлении попапа:**

- [ ] Esc вызывает тот же путь, что «Отмена» / клик по фону / blur списка
- [ ] Модалки: `role="dialog"` / `role="alertdialog"`, `aria-modal="true"`
- [ ] Combobox: `aria-expanded={open}` сбрасывается в `false` на Esc

Эталон: `ConfirmDialog.svelte`, `TransferForm.svelte` (`svelte:window`), `TimezonePicker.svelte` (список на Esc).

## DateTimePicker и всплывающие панели

Полный стандарт форматов и режимов — **[date-time-display.md](date-time-display.md)**. Константы пропсов: `$lib/datetime-picker-standards.ts`.

| Файл / класс | Назначение |
|--------------|------------|
| `$lib/components/DateTimePicker.svelte` | Выбор даты и опционально времени |
| `$lib/datetime-picker-standards.ts` | `operationDatetimePickerCreate` / `Edit`, `dateOnlyPicker` |
| `$lib/datetime-picker.ts` | Сетка календаря (`calendarCells`), разбор `datetime-local` |
| `.popover-panel` в `layout.css` | Календарь, `Select`, `Combobox`, мобильное меню — контрастный фон и тень (в т.ч. тёмная тема) |

**Режимы времени** (`timeMode`):

| Режим | Где | Поведение |
|-------|-----|-----------|
| `optional` | **Момент операции** (создание/редактирование) | На кнопке — только дата; время в свёрнутом `<details>` «Указать время» |
| `hidden` | Фильтры, срок возврата, даты кредитов | Только дата; в значении `T00:00` |
| `visible` | Не используется | Устаревший режим; не применять в новых экранах |

**Момент операции** — `optional` + `defaultTime`:

| | Создание | Редактирование |
|---|----------|----------------|
| Константа | `operationDatetimePickerCreate` | `operationDatetimePickerEdit` |
| `defaultTime` | `now` — текущее время в скрытом поле | `preserve` — время из БД |
| Инициализация | `nowDatetimeLocal(tz)` | `toDatetimeLocalValue(api, tz)` |

Формы: `TransactionForm`, `TransferForm`, `DebtForm` (дата операции), `SettleDebtForm`.

**Только дата** — `dateOnlyPicker` (`timeMode="hidden"`): фильтры, плановый возврат долга, выдача/график кредита, периодические, оплата кредита наперёд. В API — границы суток (`fromDateLocalStart` / `fromDateLocalEnd`).

**Календарь:**

- Сетка Пн–Вс; серые числа — дни **предыдущего и следующего** месяца; по клику выбирается полная дата (месяц переключается).
- Заголовок месяца — выбор месяца и года; стрелки ‹ › — соседний месяц.
- Кнопка **«Сегодня»** — выбор текущей даты в часовом поясе пользователя (v1.2.4).
- После выбора даты в режиме без раскрытого времени попап закрывается («Готово» или клик вне).

### Параметры `ConfirmOptions`

| Поле | По умолчанию |
|------|----------------|
| `title` | `common.confirm.title` |
| `message` | обязательно |
| `confirmLabel` | `common.confirm.confirm` |
| `cancelLabel` | `common.cancel` |
| `danger` | `false` — при `true` класс `btn-danger` |
| `acknowledgeOnly` | `false` — одна кнопка «Закрыть» (`common.close`), без подтверждения действия |

## i18n

Ключи подтверждений в `web/src/lib/i18n/ru.json` и `en.json`:

- `common.confirm.title`, `common.confirm.confirm`
- `accounts.confirm.creditCardNotFullyPaid`
- `accounts.confirm.archive`, `accounts.confirm.archiveWithBalance.before`, `accounts.confirm.archiveWithBalance.after`
- `accounts.confirm.delete`, `accounts.confirm.deleteWithBalance.before`, `accounts.confirm.deleteWithBalance.after`, `accounts.confirm.transferTo`, `accounts.confirm.inactiveNoTargets`
- `categories.confirm.delete`, `categories.confirm.deleteSub`
- `settings.tokens.confirm.revoke`
- `admin.users.confirm.delete` (плейсхолдер `{name}`)
- `admin.backups.confirm.restore`
- `credits.confirm.delete`, `credits.confirm.deletePayment`, `credits.confirm.deleteAppliedPayment`
- `credits.complete.payFromAccount` (плейсхолдеры `{amount}`, `{account}`)

Новые destructive-действия — добавлять ключ `*.confirm.*` и вызывать через `confirm()`.

## Восстановление БД

Раздел **Настройки → Админка → Бэкапы** — форма восстановления:

| Шаг | Поведение |
|-----|-----------|
| Выбор файла | Кнопка «Загрузить файл» → скрытый `<input type="file" accept=".db">` |
| Подтверждение | Текстовое поле: ввести **`RESTORE`** (регистр важен) |
| Кнопка «Восстановить» | **Неактивна**, пока не выбран файл и `confirm !== 'RESTORE'` |
| Перед запросом | `confirm({ message: admin.backups.confirm.restore, danger: true })` |
| После успеха API | `logout()` → полный переход на `/login` (сессия из бэкапа недействительна) |

Setup **не** повторяется: маркер `data/.configured` вне SQLite, restore его не удаляет.

API: `POST /api/v1/admin/backups/restore` (multipart `file` + `confirm`). См. [OpenAPI](api/openapi.yaml).

## Стили

- Оверлей: `z-[60]`, полупрозрачный фон
- Карточка: класс `card`, `role="alertdialog"`
- Destructive: `btn-danger` в `layout.css` (`var(--danger)`)
- Прокрутка: одна зона на модалку, кастомные скроллбары — см. [ui-stable-layout.md](ui-stable-layout.md)

## Где уже применено

- Удаление и архивация счёта (с переводом остатка при `balance > 0`)
- Удаление категории / подкатегории
- Удаление / закрытие долга (`/debts`) — удаление долга снимает **все** связанные операции
- Создание долга — при активном противоположном направлении у того же должника API вернёт **409**; `DebtForm` показывает ошибку до отправки
- Погашение долга (`SettleDebtForm`) — сумма, дата; переключатель «Не учитывать в балансе» (по умолчанию **выключен**); затем счёт (скрыт при включённом переключателе). Счёт по умолчанию — `account_id` долга или основной. Флаг `affects_balance` при погашении задаётся в форме, а не копируется из долга
- Удаление операции из списка (`/transactions`, счёт) — начальную операцию долга нельзя удалить при наличии погашений (409); операцию погашения можно удалить (остаток пересчитывается); единственную начальную — удаляет долг целиком
- Отзыв API-токена
- Удаление пользователя (админка)
- Восстановление БД из бэкапа — см. [Восстановление БД](#восстановление-бд)
- **Кредиты** (`/credits`): … подтверждение **«Удалить платёж»** (для оплаченных — `credits.confirm.deleteAppliedPayment`) и удаления кредита — `$lib/confirm`

## Требование для новых экранов

При добавлении удаления, отмены необратимых операций, предупреждений:

1. **Не** вызывать `window.confirm` / `alert` / `prompt`.
2. Использовать `confirm()` или собственную модалку в том же визуальном стиле.
3. Тексты — через i18n.
4. Destructive — `danger: true`.
5. **Escape** закрывает попап (модалку, список combobox, picker) — см. [Закрытие по Escape](#закрытие-по-escape).

## Локальный feedback действий

Для действий в конкретном блоке интерфейса (кнопки `Сохранить`, `Отправить тест`, `Сбросить`, `Импортировать` и т.п.):

- Ошибку/успех показывать **рядом с местом действия** (в том же блоке/карточке), а не только глобально вверху страницы.
- Если у страницы несколько независимых блоков, feedback хранить и рендерить раздельно по каждому блоку.
- Глобальный баннер допустим только для действительно глобальных операций страницы.

## Коды ошибок API (формы)

Ответы `4xx` с телом `{ "error": { "code", "message" } }` — показывать `message` пользователю.  
Полный список кодов — схема `ApiError` в [api/openapi.yaml](api/openapi.yaml).

Примеры для настроек → пароль:

На форме (`/settings/password`) клиент **до** запроса проверяет совпадение нового пароля с подтверждением и что новый ≠ текущий; при нарушении — локальное сообщение, API не вызывается.

| code | Когда |
|------|--------|
| `PASSWORDS_MISMATCH` | `new_password` ≠ `new_password_confirm` |
| `INVALID_CURRENT_PASSWORD` | неверный текущий пароль |
| `PASSWORD_TOO_SHORT` | короче 8 символов |
| `PASSWORD_UNCHANGED` | новый пароль совпадает с текущим |

При добавлении нового экрана — сверяться с чеклистом в этом файле (блок «Закрытие по Escape»).
