# UI и API — отображение операций

Соглашения для списков операций, главной и ответов API по транзакциям. Реализация: `web/src/lib/transaction-display.ts`, `server/queries/transactions.sql`.

## Поля API (обогащение в SELECT, не колонки БД)

OpenAPI-схемы: [`Transaction`](api/openapi.yaml#/components/schemas/Transaction), [`AccountBalanceSummary`](api/openapi.yaml#/components/schemas/AccountBalanceSummary), [`Dashboard`](api/openapi.yaml#/components/schemas/Dashboard).

В ответах `Transaction` (списки, дашборд, `GET /transactions/{id}`):

| Поле | Тип | Описание |
|------|-----|----------|
| `account_name` | string | Имя счёта ноги (`accounts.name` по `account_id`) |
| `account_status` | string | Статус счёта ноги: `active` \| `archived` \| `deleted` |
| `transfer_account_name` | string | Имя второго счёта перевода (`JOIN` по `transfer_account_id`) |
| `transfer_account_status` | string | Статус второго счёта перевода |
| `transfer_is_out` | bool | Только для `type=transfer`: `true` — исходящая нога (первая в `transfer_group_id` по `created_at ASC`), `false` — входящая |
| `category_name`, `category_icon` | | Из `categories` |
| `category_is_system` | bool | `categories.is_system` — скрывает «Повторить» для дохода/расхода |
| `subcategory_name`, `subcategory_icon` | | Из `subcategories`; без подкатегории `subcategory_icon` пустой |
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

Для счетов со статусом `archived` или `deleted` компонент `TransactionAccountCell` добавляет суффикс «(архив)» / «(удалён)» по полям `account_status` и `transfer_account_status`.

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

## Колонка «Дата» в таблицах операций

**(v1.2.4)** Дата операции — `formatAPIOperationDateTimeForDisplay` (`$lib/dates.ts`): **`дд.мм.гггг чч:мм`** без секунд, в часовом поясе пользователя. То же правило — долги, кредиты (график, `next_run_at`), уведомления (`{date}`, `{requested_at}`). С секундами — админка (диагностика, бекапы). См. [date-time-display.md](date-time-display.md).

## Главная (`/`)

- **Сводка (v1.2.4):** на десктопе (`sm:` и шире) **«Общий баланс»** и **«Долги»** — в одной строке (`grid sm:grid-cols-2`); на мобильных — два блока столбцом.
- **Счета на главной:** **два спойлера** `AccountGroupPanel` — **«Мои средства»** и **«Кредитные средства»**; на мобильных свёрнуты по умолчанию, на десктопе развёрнуты. **На `/accounts`** — те же блоки с заголовками `h2`, **без спойлеров**. В «Мои средства»: сначала `cash`, затем `bank`; внутри типа основной (`is_primary`) первым. Кредитную карту нельзя сделать основной. Карточки: `AccountIcon`, имя, баланс, лимит кредитки, автопополнение (`bank`), прогноз; меню «⋯» — [ui-row-actions.md](ui-row-actions.md). Группировка — `groupAccountsByType` ([ui-credit-cards.md](ui-credit-cards.md)).
- В шапке — три кнопки **Доход / Расход / Перевод** (`NewTransactionButtons`); тип новой операции задаётся кнопкой, без переключателя в форме.
- «Последние операции» — спойлер `CollapsibleSection` (свёрнут по умолчанию на всех экранах; в `summary` — общее число операций). Внутри карточка с двумя группами в спойлерах `<details>`:
  - **Плановые** (`kind=future`) — **выше** прошлых, спойлер **свёрнут** по умолчанию, сортировка по дате убыванию (от новых к старым), до 10 записей;
  - **Прошлые** (`kind=manual`) — спойлер **открыт** по умолчанию, сортировка по дате убыванию, до 10 записей (`GET /transactions`, `limit=10`).
- **Android (офлайн):** после merge outbox + кэш список снова сортируется `date_desc` (`mergeTransactionLists` / `sortTransactionsDateDesc`) — pending из очереди не должны подниматься в порядке FIFO (от старых к новым).
- Внизу карточки — кнопка «Все операции» → `/transactions`.
- В заголовке каждого спойлера — общее число операций в группе (`meta.total`).
- Колонки: дата, счёт, категория (с `CategoryIcon`), сумма, описание; меню «⋯» в каждой строке (как на `/transactions`).

## Все операции (`/transactions`)

- Две группы в спойлерах `<details>` внутри одной карточки (как на главной), если фильтр **«Вид»** (`kind`) не задан:
  - **Плановые** (`kind=future`) — **выше** прошлых, спойлер **свёрнут** по умолчанию, сортировка `date_desc`, до 20 записей первой страницы (без пагинации);
  - **Прошлые операции** (`kind=manual`) — спойлер **открыт** по умолчанию, сортировка `date_desc`, пагинация **20** на страницу (`?page=`, `TransactionPagination` только под блоком прошлых).
- При фильтре `kind=manual` или `kind=future` — один плоский список с пагинацией (как раньше).
- Общие фильтры (даты, тип, счёт, категория, поиск) применяются к обеим группам; `TransactionContextStats` учитывает `kind`, если он задан; без `kind` — `include_future=true` (доход/расход по плановым и фактическим). Число **«Операций»** в сводке — сумма `meta.total` групп (как в заголовках спойлеров), не `transaction_count` из stats API (тот считает только доход/расход).
- Заголовок страницы — «Все операции» (`transactions.all`); пункт верхнего меню — «Операции» (`nav.transactions`).

## Операции счёта (`/accounts/[id]`)

- Список — плоский, с пагинацией (20 на страницу); без фильтра `kind` в списке **и** плановые, и фактические операции.
- `TransactionContextStats`: без `kind` — `include_future=true`; число **«Операций»** — `meta.total` списка (`txTotal`), как на `/transactions`, а не `transaction_count` из stats API.

## Создание операций

Компонент `$lib/components/NewTransactionButtons.svelte` — три `IconButton` в шапке:

| Экран | Кнопки |
|-------|--------|
| `/` (главная) | Доход, Расход, Перевод |
| `/transactions` | Доход, Расход, Перевод |
| `/accounts/[id]` | **≥ md:** Доход, Расход, Перевод (рядом с меню «⋯» счёта). **&lt; md:** операции в меню «⋯» |

Иконки — Font Awesome Solid (`f067` доход, `f068` расход, `f0ec` перевод); видимый текст только в `title` / `aria-label`.

`TransactionForm` при **создании**: без вкладок типа; заголовок модалки — «Доход» или «Расход» (`defaultType` из родителя). При **редактировании** — заголовок «Изменить операцию», тип текстом, смена типа запрещена API. Порядок полей: счёт → категория → подкатегория → сумма → описание → дата → время (свёрнуто, `timeMode: optional`). Поле **«Новая подкатегория»** под селектом подкатегории показывается **только если** подкатегория в списке не выбрана (`{#if !subcategoryId}`).

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

## Иконки в выпадающих списках

`Select` и `Combobox` поддерживают опциональное поле `icon` в элементах списка (`$lib/select-options.ts`, `SelectOptionIcon.svelte`):

| Тип | Отображение |
|-----|-------------|
| Счёт | `AccountIcon` (`cash` / `bank` / `credit_card`, логотип банка) |
| Категория / подкатегория | `CategoryIcon` по slug |

Хелперы: `accountSelectOptions`, `categorySelectOptions`, `subcategorySelectOptions`. В селектах счетов **основной** (`is_primary`) — первым (`sortAccountsForSelect`). Используется в формах операций, переводов, долгов, кредитов, бюджета, фильтрах, импорте, периодических операциях.

## Пагинация

Общий компонент — [ui-pagination.md](ui-pagination.md). На `/transactions` (режим с `kind`-фильтром или блок «Прошлые операции» в раздельном режиме) и в операциях счёта: **20** записей на страницу (`limit=20`), номер страницы в URL (`?page=`). Кнопки «В начало», «Назад», «Вперёд», «В конец» всегда на месте; на первой/последней странице недоступные — `disabled`. Блок скрыт, если страница одна.

## Действия в списке операций

`TransactionList` — меню «⋯» в каждой строке (повторить, сделать периодической, изменить, удалить). На мобильных меню в шапке карточки рядом с суммой. Используется на **главной** («Последние операции»), `/transactions`, странице счёта.

На странице **удалённого** счёта (`/accounts/[id]`, `status = deleted`) в меню операций доступно только **«Повторить»** — см. [accounts-archive-delete.md](accounts-archive-delete.md).

**Повторить (v1.2.3)** — открывает форму **создания** новой операции с полями из выбранной строки (счёт, категория, сумма, описание; для перевода — счета, сумма, комиссия); дата — текущая. Работает для **дохода**, **расхода** и **перевода**. Доход/расход — `TransactionForm` (`repeatFrom`); перевод — `TransferForm` (`repeatFrom`). Недоступно для операций с `credit_payment_linked` и для дохода/расхода в **системных категориях** (как «Сделать периодической»).

## Категории с одинаковым именем

`$lib/category-label.ts`: `categorySelectLabel` / `duplicateCategoryNames` — суффиксы «(Доход)» / «(Расход)» в фильтрах и статистике, когда имя совпадает (например системные «Долги»).

## Формат денег

`web/src/lib/money.ts`: отображение и ввод с разделителем тысяч **пробелом** (`10 000.00`).

**Отображение сумм в UI:** компонент `MoneyDisplay.svelte` (`web/src/lib/components/MoneyDisplay.svelte`) — единая точка для read-only сумм. Принимает `value` (строка `_display` из API), `cents` (копейки) и опционально `currency` (добавляет символ валюты, например `₽`). Логика форматирования — в `$lib/money-display.ts` (`formatMoneyForDisplay`); для интерполяции в i18n (`tr(..., { values: { amount } })`) используйте ту же функцию.

Валюта сейчас берётся из `users.currency` (**только отображение**). Учёт нескольких валют на счетах — план: [multicurrency.md](../roadmap/multicurrency.md).

**Ввод:** компонент `MoneyInput.svelte` — при вводе курсор сохраняется при форматировании (`mapMoneyInputCursor`).

**Поля ввода суммы (v1.2.3):** пустое значение и ноль не показываются как `0.00` — поле остаётся пустым, подсказка `placeholder="0.00"`. При потере фокуса нулевой ввод очищается (`formatMoneyInput`). Для подстановки значений из API в формы используйте `formatMoneyForInput`, не `formatMoneyDisplay`.

## Тесты

- `TestTransferRollbackOnError` — атомарность перевода при сбое второй ноги (SQLite-триггер в интеграционном тесте).
- `money.test.ts` — стабильность курсора в `MoneyInput`, `formatMoneyForInput` / `formatMoneyInput` (ноль → пусто).
- `money-display.test.ts` — `formatMoneyForDisplay` (разделитель тысяч, валюта).
- `accounts.test.ts` — `defaultAccountId` (контекстный счёт vs основной).
- `select-options.test.ts` — иконки и подписи опций селектов.
- `e2e/money-input.spec.ts` — пустые поля суммы и placeholder в формах счёта, операций, перевода, периодических операций.
- `e2e/transaction-filters.spec.ts` — фильтры и пагинация на `/transactions` (20 на страницу).
- `e2e/transaction-actions.spec.ts` — спойлеры плановых/прошлых на `/transactions` и главной.
- `transaction-display.test.ts` — `canRepeatTransaction` (ограничения как у редактирования).

## Требование для новых экранов

При добавлении таблицы операций:

1. Использовать `formatTransactionAccount` и `transactionAmountSign` вместо дублирования логики.
2. Для списка одного счёта передавать `{ singleAccount: true }` в `transactionAmountSign`.
3. Передавать полный массив `siblings` в хелперы маршрута (для случая двух ног в одном ответе).
4. На общих списках применять `dedupeTransferLegs` перед `{#each}`.
5. Суммы в разметке — через `MoneyDisplay`; в строках i18n — `formatMoneyForDisplay` из `$lib/money-display.ts`.
6. Соблюдать порядок колонок — [ui-table-columns.md](ui-table-columns.md) (дата → счёт → … → сумма).

## Общий компонент списка

`TransactionList` — единая таблица строк операций на **главной** («Последние операции»), `/transactions`, странице счёта, `/stats` (поиск), `/debtors/[id]` («Последние операции»).

- `$lib/components/TransactionList.svelte` — разметка desktop-таблицы и mobile-карточек;
- `$lib/components/TransactionCategoryCell.svelte` — колонка «Категория»: `иконка Категория → иконка Подкатегория` (если есть подкатегория); иконка подкатегории — `subcategory_icon` из API, fallback — `category_icon`;
- входные данные: `transactions`, `siblings`, `showCategory` (на странице должника — `false`), `showAmountSign`, скрытие/показ колонки описания и действий;
- логика отображения — `$lib/transaction-display.ts` (`dedupeTransferLegs`, `formatTransactionAccount`, `transactionAmountSign`, `canDeleteTransaction`).

### Страница должника

Блок «Последние операции» (`debts.recentTransactions`): спойлер `CollapsibleSection` (свёрнут по умолчанию); без колонки «Категория», с префиксом `+/−` у суммы. В меню «⋯» — только **Удалить** (где API вернул `deletable: true`). Начальную операцию долга нельзя удалить при наличии погашений — пункт скрыт; при прямом DELETE — `409` / `ERR_LINKED_TX_DELETE`. См. [ui-dialogs.md](ui-dialogs.md).

Статус: реализовано в `$lib/components/TransactionList.svelte`.

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
