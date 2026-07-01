# Дата и время — форматы и пикер

Единые правила отображения и ввода дат в UI, уведомлениях и документации API для людей.

Константы:

| Слой | Дата | Дата-время | Дата-время без секунд |
|------|------|------------|------------------------|
| Web | `DISPLAY_DATE_FORMAT` | `DISPLAY_DATETIME_FORMAT` | `DISPLAY_DATETIME_SHORT_FORMAT` |
| Go | `DisplayDateLayout` | `DisplayDateTimeLayout` | `DisplayDateTimeShortLayout` |

Примеры: `31.12.2026` · `31.12.2026 12:00:00` · `31.12.2026 12:00`

Функции: web `$lib/dates.ts` (`formatAPIDateForDisplay`, `formatAPIDateTimeForDisplay`, `formatAPIOperationDateTimeForDisplay`); Go `timeutil/display.go`.

Хранение в API/БД не меняется: `yyyy-MM-dd HH:mm:ss` и RFC3339.

---

## DateTimePicker — стандарт для момента операции

Поля, где фиксируется **момент** (операция, перевод, дата долга, погашение):

| | Создание | Редактирование |
|---|----------|----------------|
| `timeMode` | `optional` | `optional` |
| `defaultTime` | `now` | `preserve` |
| Кнопка пикера | только **дата** | только **дата** |
| Блок «Указать время» | свёрнут; в поле — **текущее** время | свёрнут; в поле — время **из БД** |
| Значение при сохранении | дата + время (если блок не раскрывали — текущее) | дата + время из значения |

Константы в коде: `$lib/datetime-picker-standards.ts`

```typescript
import {
	operationDatetimePickerCreate,
	operationDatetimePickerEdit
} from '$lib/datetime-picker-standards';

// создание
<DateTimePicker bind:value={...} {...operationDatetimePickerCreate} />

// редактирование
<DateTimePicker
	bind:value={...}
	{...(editing ? operationDatetimePickerEdit : operationDatetimePickerCreate)}
/>
```

Инициализация значения:

- **создание:** `nowDatetimeLocal(tz)`
- **редактирование:** `toDatetimeLocalValue(apiDatetime, tz)`

Где применено:

| Форма | Поле |
|-------|------|
| `TransactionForm` | дата операции |
| `TransferForm` | дата перевода |
| `DebtForm` | дата операции (дать/взять в долг) |
| `SettleDebtForm` | дата погашения |

---

## DateTimePicker — только дата

`timeMode="hidden"`, в значении `T00:00`. Константа `dateOnlyPicker`.

| Где | Поле |
|-----|------|
| `DebtForm` | плановый возврат |
| `TransactionFilters`, статистика, экспорт импорта | фильтр от/до |
| `CreditForm` | дата выдачи, даты графика |
| `recurring-operations` | дата старта |
| Кредит: оплата наперёд, завершение | дата платежа / завершения |

В API для границ суток: `fromDateLocalStart` / `fromDateLocalEnd`.

---

## Форматы вывода (не пикер)

### Дата-время без секунд

Списки операций и долгов; дата операции в долгах; кредит (`recorded_at`, график, ближайший платёж); повторяющиеся (`next_run_at`); уведомления `{date}`, `{requested_at}`.

### Только дата

Срок возврата долга; дата выдачи кредита; уведомления `{due_date}`, `{payment_date}`; статистика по дням.

### Дата-время с секундами

Админка: диагностика, бекапы.

### Вне констант

Статистика по неделям/месяцам — локализованные подписи («Июнь 2026»). Заголовок календаря в пикере — локаль браузера. Кнопка **«Сегодня»** в `DateTimePicker` — текущая дата в TZ пользователя (v1.2.4).

---

## Новые экраны — чеклист

- [ ] Момент операции → `optional` + `operationDatetimePickerCreate` / `Edit`
- [ ] Только календарная дата → `dateOnlyPicker`
- [ ] Список/уведомление → `formatAPIOperationDateTimeForDisplay` или `formatAPIDateForDisplay`
- [ ] Не форматировать даты вручную в шаблонах

См. также [ui-dialogs.md](ui-dialogs.md#datetimepicker-и-всплывающие-панели).
