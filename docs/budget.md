# Бюджет

Модуль планирования расходов: лимиты на категорию, подкатегорию или все расходы **за календарный месяц**, сравнение «план vs факт», копирование между месяцами, уведомления при пересечении порога.

Связанные документы: [ui-budget.md](ui-budget.md), [data-model.md](data-model.md), [ui-stats.md](ui-stats.md), [notifications.md](notifications.md), [api/openapi.yaml](api/openapi.yaml).

---

## Назначение

Бюджет не заменяет [статистику](ui-stats.md): факт (`spent`) считается из тех же `transactions`, что и `GET /stats/by-category`, с теми же фильтрами.

| | Статистика | Бюджет |
|---|------------|--------|
| Вопрос | «Сколько потратил?» | «Сколько **можно** потратить?» |
| Период | любой | календарный месяц |
| Гранулярность | категория / подкатегория | категория, подкатегория, все расходы |
| Проактивность | нет | предупреждение при пороге (по умолчанию 90%) и 100% |

---

## Scope (область лимита)

| `scope` | Описание | Уникальность в месяце |
|---------|----------|------------------------|
| `category` | Расходы с `category_id` | один бюджет на категорию |
| `subcategory` | Расходы с `subcategory_id` | один бюджет на подкатегорию |
| `all_expense` | Все операции `type = expense` | один общий бюджет |

`all_income` зарезервирован в схеме; не используется.

Опциональный `account_id` **сужает расчёт факта** (только операции по счёту), но **не создаёт отдельный бюджет**: два лимита на одну категорию в одном месяце невозможны, даже если счета разные.

Системные категории расходов в форме бюджета не выбираются. Платежи кредита с `exclude_from_stats = 1` **не входят** в факт (как в статистике).

---

## Помесячные записи

Каждая строка `budgets` относится к **одному** календарному месяцу:

| Поле | Описание |
|------|----------|
| `month` | `YYYY-MM` в timezone пользователя |
| `copy_forward` | `1` — при первом открытии **следующего** месяца создать копию (один раз) |

Бюджет **не** действует на все будущие месяцы автоматически. Просмотр другого месяца без записи в `budgets` для этого `month` — пустой список (с опцией «Скопировать» с прошлого месяца).

Миграция `038_budget_month_copy.sql`: существующим записям проставляется `month` из раннего `budget_periods` или `created_at`.

---

## Копирование на следующий месяц

| Способ | Поведение |
|--------|-----------|
| `copy_forward = 1` при создании | При загрузке **следующего** месяца (`summary` / `list`) — ленивое создание копии |
| Копия | Новая запись с `copy_forward = 0` (дальше не тиражируется) |
| `POST …/budgets/{id}/copy-next` | Ручное копирование одного бюджета на месяц `month + 1` |
| `POST …/budgets/copy-from-previous?month=` | Все активные бюджеты прошлого месяца → указанный месяц (если нет конфликта) |
| UI «Скопировать» | То же, что `copy-from-previous`; видна, если в выбранном месяце нет бюджетов, а в предыдущем есть (`can_copy_from_previous`) |

Авто-копирование срабатывает только с **непосредственно предыдущего** месяца (не цепочкой через несколько месяцев).

При конфликте scope в целевом месяце копия пропускается (массовое копирование) или API возвращает `409 ERR_BUDGET_COPY_EXISTS` (`copy-next`).

---

## Уникальность

Один **активный** бюджет на комбинацию `(user_id, scope, category_id, subcategory_id, month)` — частичный уникальный индекс в БД (`039_budget_scope_unique.sql`).

Запрещено в одном месяце:

- два `all_expense`;
- два бюджета на одну категорию (например, «Магазины»);
- два бюджета на одну подкатегорию (например, «Озон»).

Сумма лимита, порог уведомления и счёт **не** делают бюджет «другим» для целей уникальности.

API: `409 ERR_BUDGET_DUPLICATE` при создании/активации дубликата.

---

## Расчёт факта (`spent`)

- Период: `[period_start, period_end)` — границы месяца в `users.timezone`, UTC в БД.
- Только `kind = manual`.
- Тип: `expense`.
- **Переводы между своими счетами не учитываются:** `transfer_group_id IS NOT NULL` (в т.ч. комиссия в «Комиссия»).
- Исключение платежей кредита: `credit_payments.exclude_from_stats = 1`.
- Опционально: фильтр `account_id` из бюджета.

`planned` = `budget_periods.planned_amount` (период создаётся лениво из `budgets.amount` при первом обращении к месяцу).

`remaining` = `planned - spent` (может быть отрицательным).

`percent` = `round(spent * 100 / planned)`.

`status`:

| Значение | Условие |
|----------|---------|
| `ok` | &lt; `alert_at_percent` |
| `warning` | ≥ порога и ≤ 100% |
| `exceeded` | &gt; 100% |

---

## Сводка (`GET /budgets/summary`)

### Порядок карточек

1. `all_expense` — всегда первая.
2. Остальные — по имени (`name`).

### Дочерние суммы в общем бюджете

Для `scope = all_expense` в ответ добавляются поля (сумма по всем активным бюджетам `category` и `subcategory` **этого же месяца**):

| Поле | Смысл |
|------|--------|
| `children_planned` | Сумма лимитов дочерних бюджетов |
| `children_spent` | Сумма факта дочерних бюджетов |
| `*_display` | Форматированные строки для UI |

В UI: «По категориям: {children_spent} / {children_planned}».

### Тело ответа

```json
{
  "month": "2026-07",
  "can_copy_from_previous": false,
  "items": [ /* BudgetSummaryItem[] */ ]
}
```

---

## Поля бюджета

| Поле | По умолчанию | Описание |
|------|--------------|----------|
| `name` | — | Отображаемое имя |
| `scope` | — | `category` \| `subcategory` \| `all_expense` |
| `amount` | — | Лимит, копейки |
| `alert_at_percent` | **90** | Порог уведомления, 0 = выкл |
| `is_active` | `true` | Неактивный не попадает в `summary` |
| `copy_forward` | `false` | Авто-копия на следующий месяц |
| `month` | текущий месяц пользователя | При `POST` — query `month` или текущий |
| `account_id` | `null` | Все счета |

Редактирование `amount` обновляет `budget_periods.planned_amount` текущего месяца (query `month` в `PATCH`).

---

## API

```
GET    /api/v1/budgets?month=2026-06
POST   /api/v1/budgets?month=2026-06
PATCH  /api/v1/budgets/{id}?month=2026-06
DELETE /api/v1/budgets/{id}
GET    /api/v1/budgets/summary?month=2026-06
POST   /api/v1/budgets/{id}/copy-next
POST   /api/v1/budgets/copy-from-previous?month=2026-06
```

`month` — `YYYY-MM` в timezone пользователя.

`BudgetUpsertRequest`: `name`, `scope`, `amount`, опционально `category_id`, `subcategory_id`, `account_id`, `alert_at_percent`, `is_active`, `copy_forward`.

Коды ошибок: `ERR_BUDGET_DUPLICATE`, `ERR_BUDGET_COPY_EXISTS`, `ERR_BUDGET_NOTHING_TO_COPY`, `ERR_BUDGET_MONTH`, `ERR_BUDGET_SCOPE`.

---

## Уведомления

Триггер `budget_threshold` — см. [notifications.md](notifications.md).

Пороги: `alert_at_percent` и 100%. Дедупликация: `budget_alert_sent (budget_id, period_start, threshold_percent)`.

Hook после расходной операции и worker по расписанию пересчитывают `summary` **текущего** месяца пользователя.

---

## Миграции

| Файл | Содержание |
|------|------------|
| `034_budgets.sql` | Таблица `budgets` |
| `035_budget_periods.sql` | `budget_periods` |
| `036_budget_alert_sent.sql` | Дедуп уведомлений |
| `037_notification_budget_threshold.sql` | `trigger_budget` |
| `038_budget_month_copy.sql` | `month`, `copy_forward` |
| `039_budget_scope_unique.sql` | Уникальность без `account_id` |

---

## Вне scope

- Прогноз по периодическим операциям до конца месяца
- Rollover остатка (`rollover`, `rollover_amount`)
- Бюджет на доход (`all_income`)
- Группы категорий
- Включение комиссий переводов в факт (сейчас исключены)

Виджет на главной — [ui-budget.md](ui-budget.md#виджет-на-главной).
