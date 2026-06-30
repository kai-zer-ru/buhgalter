# UI и API — отображение операций

Соглашения для списков операций, главной и ответов API по транзакциям. Реализация: `web/src/lib/transaction-display.ts`, `server/queries/transactions.sql`.

## Поля API (обогащение в SELECT, не колонки БД)

OpenAPI-схемы: [`Transaction`](api/openapi.yaml#/components/schemas/Transaction), [`AccountBalanceSummary`](api/openapi.yaml#/components/schemas/AccountBalanceSummary), [`Dashboard`](api/openapi.yaml#/components/schemas/Dashboard).

В ответах `Transaction` (списки, дашборд, `GET /transactions/{id}`):

| Поле | Тип | Описание |
|------|-----|----------|
| `account_name` | string | Имя счёта ноги (`accounts.name` по `account_id`) |
| `transfer_account_name` | string | Имя второго счёта перевода (`JOIN` по `transfer_account_id`) |
| `transfer_is_out` | bool | Только для `type=transfer`: `true` — исходящая нога (первая в `transfer_group_id` по `created_at ASC`), `false` — входящая |
| `category_name`, `category_icon` | | Из `categories` |
| `amount_display` | string | `"1234.56"` для UI |
| `commission` | integer | Только для перевода: комиссия в копейках (v1.1) |
| `commission_display` | string | Отформатированная комиссия |
| `credit_payment_linked` | bool | `true` — операция привязана к платежу по кредиту; редактирование через `PUT /transactions/{id}` запрещено |

`transfer_is_out` вычисляется в sqlc-запросах (`CASE` + подзапрос по `transfer_group_id`), не хранится в таблице `transactions`.

В ответах **дашборда** и **сводки счетов** (`AccountBalance` / `GET /dashboard`, `/accounts/summary`, `/accounts/{id}/balance`):

| Поле | Описание |
|------|----------|
| `type` | `cash` \| `bank` |
| `bank_icon` | Slug логотипа банка (для `AccountIcon` на главной) |

## Модуль `$lib/transaction-display.ts`

| Функция | Назначение |
|---------|------------|
| `transferOutLeg(tx, siblings)` | Исходящая нога пары (min `created_at` в группе) |
| `dedupeTransferLegs(txs)` | Оставить одну строку на перевод, если в списке обе ноги (главная, `/transactions`) |
| `transferRoute(tx, siblings)` | `{ from, to }` — имена счетов **в направлении движения денег** |
| `transferAccountIds(tx, siblings?)` | `{ fromAccountId, toAccountId }` для формы редактирования перевода (одна видимая нога + `transfer_is_out`) |
| `canEditTransaction(tx)` | `false` при `credit_payment_linked` — скрыть «Изменить» в списке |
| `canRepeatTransaction(tx)` | `false` при `credit_payment_linked` или системной категории (доход/расход); переводы — без ограничения по категории |
| `formatTransactionAccount(tx, siblings, mode)` | Текст колонки «Счёт» |
| `transactionAmountSign(tx, opts?)` | Префикс суммы: `+`, `−` или пусто |

## Колонка «Счёт» в таблицах

Режим `mode: 'prefix'` (главная, `/transactions`, `/accounts/[id]`):

| Тип | Пример |
|-----|--------|
| Перевод | `WB → Наличка` (всегда откуда → куда, даже на странице счёта-получателя) |
| Расход | `с Кошелёк` |
| Доход | `на Зарплатная карта` |

Для перевода при **одной ноге** в списке (фильтр `account_id`) направление берётся из `transfer_is_out`, а не из локального `min(created_at)` среди видимых строк.

## Префикс суммы

| Контекст | income | expense | transfer |
|----------|--------|---------|----------|
| Главная, `/transactions` | `+` | `−` | *(пусто)* |
| `/accounts/[id]` (`singleAccount: true`) | `+` | `−` | `−` исходящая / `+` входящая (`transfer_is_out`) |

Символ `↔` у переводов **не используется**.

## Дедупликация переводов в списках

На главной и в `/transactions` в одном списке могут попасть обе ноги перевода → `dedupeTransferLegs` оставляет только исходящую ногу.

На `/accounts/[id]` API отдаёт только ногу выбранного счёта — дедупликация не меняет состав, но `transferRoute` всё равно опирается на `transfer_is_out`.

## Комиссия перевода (v1.1)

В форме перевода (`TransferForm.svelte`) — опциональное поле **комиссии**. При сохранении:

- Создаётся расход на счёте-источнике в системной категории «Комиссия»
- В ответе `GET /transactions/{id}` для перевода — поля `commission`, `commission_display` (сумма ноги комиссии в группе)

OpenAPI: `CreateTransferRequest.commission`, схема `Transfer`.

## Главная (`/`)

- Карточки счетов: `AccountIcon` (`type`, `bank_icon` из API), имя, баланс и меню «⋯» — как на `/accounts` ([ui-row-actions.md](ui-row-actions.md)).
- В шапке — три кнопки **Доход / Расход / Перевод** (`NewTransactionButtons`); тип новой операции задаётся кнопкой, без переключателя в форме.
- «Последние операции»: колонки дата, счёт, категория (с `CategoryIcon`), сумма, описание; меню «⋯» в каждой строке (как на `/transactions`).
- Ссылка «Все операции» → `/transactions`.

## Создание операций

Компонент `$lib/components/NewTransactionButtons.svelte` — три `IconButton` в шапке:

| Экран | Кнопки |
|-------|--------|
| `/` (главная) | Доход, Расход, Перевод |
| `/transactions` | Доход, Расход, Перевод |
| `/accounts/[id]` | **≥ md:** Доход, Расход, Перевод (рядом с меню «⋯» счёта). **&lt; md:** операции в меню «⋯» |

Иконки — Font Awesome Solid (`f067` доход, `f068` расход, `f0ec` перевод); видимый текст только в `title` / `aria-label`.

`TransactionForm` при **создании**: без вкладок типа; заголовок модалки — «Доход» или «Расход» (`defaultType` из родителя). При **редактировании** — заголовок «Изменить операцию», тип текстом, смена типа запрещена API.

`TransferForm` — отдельная модалка по кнопке перевода на тех же экранах. **(v1.2.3)** В селектах «Откуда» и «Куда» выбранный в одном поле счёт **не показывается** в другом (`transferAccountOptions` в `$lib/transfer-accounts.ts`); при совпадении значений (например, один активный счёт) поле «Куда» сбрасывается на другой счёт. При создании перевода со **страницы счёта** (`/accounts/[id]`) в «Откуда» подставляется **текущий счёт** (как счёт в `TransactionForm` для дохода/расхода); на главной и в «Все операции» — **основной** счёт (`defaultAccountId` без явного id).

## Редактирование дохода и расхода

Из списка операций (`TransactionList` — главная, `/transactions`, `/accounts/[id]`) — пункт меню «Изменить» открывает `TransactionForm`:

- API: `PUT /api/v1/transactions/{id}` (`updateTransaction` в клиенте)
- Можно менять: счёт, категорию, подкатегорию, сумму, описание, дату
- **Тип** (`income` / `expense`) не меняется — в форме показывается текстом; при смене типа в теле запроса — `ERR_TX_TYPE_CHANGE`
- Дата в будущем (в TZ пользователя) → `kind=future` (плановая операция)
- Операции с `credit_payment_linked` **не** редактируются из списка (`canEditTransaction`)

## Редактирование перевода (v1.1)

Из списка операций (`TransactionList`, страница счёта, `/transactions`) — кнопка «Изменить» открывает `TransferForm` в режиме редактирования:

- API: `PUT /api/v1/transfers/{group_id}` (`updateTransfer` в клиенте)
- Счета «откуда/куда» — `transferAccountIds()` (корректно при одной ноге на `/accounts/[id]`)
- Операции с `credit_payment_linked` **не** редактируются из списка (`canEditTransaction`)

## Фильтры

`TransactionFilters` внутри `FilterPanel`: на мобильных — спойлер «Фильтры» с chevron; на десктопе — всегда открыта. Поля с `dateOnlyPicker`: только дата, границы суток — `fromDateLocalStart` / `toDateLocalEnd` (`$lib/dates.ts`) в часовом поясе пользователя. См. [date-time-display.md](date-time-display.md). Используется на `/transactions`, `/stats`, странице счёта.

Подробнее: [ui-row-actions.md](ui-row-actions.md).

## Действия в списке операций

`TransactionList` — меню «⋯» в каждой строке (повторить, сделать периодической, изменить, удалить). На мобильных меню в шапке карточки рядом с суммой. Используется на **главной** («Последние операции»), `/transactions`, странице счёта.

**Повторить (v1.2.3)** — открывает форму **создания** новой операции с полями из выбранной строки (счёт, категория, сумма, описание; для перевода — счета, сумма, комиссия); дата — текущая. Работает для **дохода**, **расхода** и **перевода**. Доход/расход — `TransactionForm` (`repeatFrom`); перевод — `TransferForm` (`repeatFrom`). Недоступно для операций с `credit_payment_linked` и для дохода/расхода в **системных категориях** (как «Сделать периодической»).

## Категории с одинаковым именем

`$lib/category-label.ts`: `categorySelectLabel` / `duplicateCategoryNames` — суффиксы «(Доход)» / «(Расход)» в фильтрах и статистике, когда имя совпадает (например системные «Долги»).

## Формат денег

`web/src/lib/money.ts`: отображение и ввод с разделителем тысяч **пробелом** (`10 000.00`). Компонент `MoneyInput.svelte` — при вводе курсор сохраняется при форматировании (`mapMoneyInputCursor`).

**Поля ввода суммы (v1.2.3):** пустое значение и ноль не показываются как `0.00` — поле остаётся пустым, подсказка `placeholder="0.00"`. При потере фокуса нулевой ввод очищается (`formatMoneyInput`). Для подстановки значений из API в формы используйте `formatMoneyForInput`, не `formatMoneyDisplay`.

## Тесты

- `TestTransferRollbackOnError` — атомарность перевода при сбое второй ноги (SQLite-триггер в интеграционном тесте).
- `money.test.ts` — стабильность курсора в `MoneyInput`, `formatMoneyForInput` / `formatMoneyInput` (ноль → пусто).
- `accounts.test.ts` — `defaultAccountId` (контекстный счёт vs основной).
- `e2e/money-input.spec.ts` — пустые поля суммы и placeholder в формах счёта, операций, перевода, периодических операций.
- `e2e/account-actions.spec.ts` — подстановка текущего счёта в формах расхода и перевода на странице счёта.
- `transaction-display.test.ts` — `canRepeatTransaction` (ограничения как у редактирования).

## Требование для новых экранов

При добавлении таблицы операций:

1. Использовать `formatTransactionAccount` и `transactionAmountSign` вместо дублирования логики.
2. Для списка одного счёта передавать `{ singleAccount: true }` в `transactionAmountSign`.
3. Передавать полный массив `siblings` в хелперы маршрута (для случая двух ног в одном ответе).
4. На общих списках применять `dedupeTransferLegs` перед `{#each}`.
5. Соблюдать порядок колонок — [ui-table-columns.md](ui-table-columns.md) (дата → счёт → … → сумма).

## Общий компонент списка

Для снижения дублирования разметки таблиц операций на этапе релизной полировки допускается вынести общий компонент:

- `$lib/components/TransactionList.svelte` — единая таблица строк операций;
- входные данные: `transactions`, `siblings`, режим отображения (`singleAccount`, скрытие/показ колонки описания и действий);
- логика отображения берётся только из `$lib/transaction-display.ts` (`dedupeTransferLegs`, `formatTransactionAccount`, `transactionAmountSign`), без копирования в страницах.

Статус: реализовано в `$lib/components/TransactionList.svelte` (см. раздел выше).

## Связанные документы

| Документ | Содержание |
|----------|------------|
| [date-time-display.md](date-time-display.md) | Форматы даты/времени, DateTimePicker |
| [ui-table-columns.md](ui-table-columns.md) | Порядок колонок (дата → счёт → … → сумма) |
| [ui-navigation.md](ui-navigation.md) | `BackLink` на `/transactions` → главная; кликабельные счета |
| [data-model.md](data-model.md) | `transfer_group_id`, вычисляемые поля в sqlc |
| [api/openapi.yaml](api/openapi.yaml) | Схемы `Transaction`, `Dashboard`, `AccountBalanceSummary`, `StatsSummary`, `/stats/*`, `ErrorResponse.error.field` |
| [ui-row-actions.md](ui-row-actions.md) | `RowActionsMenu`, `FilterPanel` |
| [ui-navigation.md](ui-navigation.md) | Фильтры и пагинация списков — те же хелперы отображения строк |
| [ui-stats.md](ui-stats.md) | Страница `/stats` |
