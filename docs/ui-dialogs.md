# UI — диалоги, попапы и подтверждения

В интерфейсе **не используем** нативные `window.confirm`, `window.alert`, `window.prompt` — только модальные окна в разметке страницы (оверлей + карточка).

**Esc:** любой **попап** (модалка, выпадающий список, picker) закрывается по **Escape** — см. [Закрытие по Escape](#закрытие-по-escape).

## Компоненты

| Файл | Назначение |
|------|------------|
| `$lib/confirm.ts` | `confirm(options): Promise<boolean>` — API как у `window.confirm`, но с кастомным UI |
| `$lib/components/ConfirmDialog.svelte` | Глобальный диалог подтверждения |
| `routes/+layout.svelte` | `<ConfirmDialog />` подключён один раз для всего приложения |

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

## Закрытие по Escape

**Все попапы** приложения закрываются по **Esc** — единое правило для модалок, выпадающих списков и overlay-виджетов.

| Тип | Поведение |
|-----|-----------|
| `ConfirmDialog` | `false` (отмена) — уже реализовано |
| Формы (`TransactionForm`, `TransferForm`, `DebtForm`, `CreditForm`, …) | `onclose()` — отмена без сохранения |
| Информационные модалки (API-токен, preview и т.п.) | закрыть, без side effects |
| `CategoryIconPicker`, выбор иконки | закрыть без выбора |
| **Combobox / Select** (`TimezonePicker`, будущий `Combobox`) | закрыть список; если список уже закрыт — не перехватывать Esc у родительской модалки |
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

### Параметры `ConfirmOptions`

| Поле | По умолчанию |
|------|----------------|
| `title` | `common.confirm.title` |
| `message` | обязательно |
| `confirmLabel` | `common.confirm.confirm` |
| `cancelLabel` | `common.cancel` |
| `danger` | `false` — при `true` класс `btn-danger` |

## i18n

Ключи подтверждений в `web/src/lib/i18n/ru.json` и `en.json`:

- `common.confirm.title`, `common.confirm.confirm`
- `accounts.confirm.delete`
- `categories.confirm.delete`, `categories.confirm.deleteSub`
- `settings.tokens.confirm.revoke`
- `admin.users.confirm.delete` (плейсхолдер `{name}`)
- `admin.backups.confirm.restore`
- `credits.confirm.delete`, `credits.confirm.deletePayment`, `credits.confirm.deleteAppliedPayment`
- `credits.complete.payFromAccount` (плейсхолдеры `{amount}`, `{account}`)

Новые destructive-действия — добавлять ключ `*.confirm.*` и вызывать через `confirm()`.

## Восстановление БД

Страница `/admin/backups` — форма восстановления:

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

## Где уже применено (этапы 1–5)

- Удаление счёта
- Удаление категории / подкатегории
- Удаление / закрытие долга (`/debts`) — удаление долга снимает **все** связанные операции
- Создание долга — при активном противоположном направлении у того же должника API вернёт **409**; `DebtForm` показывает ошибку до отправки
- Удаление операции из списка (`/transactions`, счёт) — для долга после погашения API вернёт ошибку; показать текст от сервера (409)
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
Полный список кодов этапа 1 — схема `ApiError` в [api/openapi.yaml](api/openapi.yaml).

Примеры для настроек → пароль:

На форме (`/settings?tab=password`) клиент **до** запроса проверяет совпадение нового пароля с подтверждением и что новый ≠ текущий; при нарушении — локальное сообщение, API не вызывается.

| code | Когда |
|------|--------|
| `PASSWORDS_MISMATCH` | `new_password` ≠ `new_password_confirm` |
| `INVALID_CURRENT_PASSWORD` | неверный текущий пароль |
| `PASSWORD_TOO_SHORT` | короче 8 символов |
| `PASSWORD_UNCHANGED` | новый пароль совпадает с текущим |

Чеклист в каждом `stage_*.md` (блок «UI — диалоги»).
