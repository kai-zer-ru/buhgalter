# Правила категорий

Постоянные правила пользователя: подстановка счёта по категории, автокатегоризация по тексту описания, очередь «разобрать». Продуктовый план — [category-rules-inbox.md](../roadmap/category-rules-inbox.md).

## Модель

Таблица `category_rules` (миграция `054_category_rules.sql`):

| Поле | Описание |
|------|----------|
| `kind` | `category_account` — категория (+ опц. подкатегория) → счёт по умолчанию; `text_category` — подстрока в описании → категория |
| `pattern` | Только `text_category`, до 200 символов |
| `category_id` | Условие или целевая категория |
| `subcategory_id` | Опционально; для `category_account` NULL = любая подкатегория этой категории |
| `default_account_id` | Только `category_account`, активный счёт пользователя |
| `priority` | Меньше — раньше при разрешении конфликтов |

Лимит — 100 правил на пользователя. При `DELETE /user/data` правила удаляются вместе с учётом.

## API

Префикс `/api/v1/category-rules` (см. OpenAPI):

| Метод | Путь |
|-------|------|
| GET | `/category-rules` |
| POST | `/category-rules` |
| GET/PUT/DELETE | `/category-rules/{id}` |
| PUT | `/category-rules/order` — `{ "ids": [...] }` |
| POST | `/category-rules/evaluate` — подсказки для формы операции |
| GET | `/transactions/inbox`, `/transactions/inbox/count` — очередь без категории |

Кеш GET — TTL 5 мин (как каталоги); мутации сбрасывают `/api/v1/category-rules` и inbox/dashboard hints.

При `POST /transactions` без категории: если ровно одно правило `text_category` подходит — категория проставляется на сервере (silent apply).

Коды валидации: `ERR_CATEGORY_RULE_*` (kind, pattern, category, subcategory, account, system category, limit, order).

## Правило «категория → счёт»

Тип `category_account` **не** меняет сохранённую операцию на сервере. Клиент при смене категории в форме дохода/расхода выбирает счёт из первого подходящего правила (`priority`, затем `id`):

- совпадает `category_id`;
- если в правиле задана `subcategory_id` — нужно точное совпадение;
- если `subcategory_id` в правиле NULL — подходит любая подкатегория или отсутствие подкатегории.

Домен: `categoryrule.SuggestDefaultAccount`.

## UI

| Платформа | Правила | Inbox |
|-----------|---------|-------|
| Web | «Прочее» → `/category-rules` | «Прочее» → `/transactions/inbox`, только при `inbox_count > 0` (badge на «Прочее» и у пункта) |
| Android | drawer → «Правила категорий» | drawer → «Разобрать», только при `inbox_count > 0` |

Web `/category-rules`: две группы по `kind` («Категория → счёт», «Текст → категория»), номер строки — место в своей группе. Строка «категория → счёт» показывает только «Категория › Подкатегория» (или «Категория»), счёт виден при открытии правила; строка «текст → категория» — «шаблон» и ту же категорию. Названия подкатегорий подгружаются после списка правил (`GET /categories/{id}/subcategories` для категорий с правилом на подкатегорию). Перетаскивание за ⠿ двигает правило только внутри группы: виды разрешаются независимо, в `PUT /category-rules/order` уходит изменённая группа, за ней вторая в прежнем порядке. В форме тип правила и тип категории — сегментированные переключатели.

Форма операции вызывает `POST /category-rules/evaluate` (подсказка категории и подстановка счёта). Счётчик inbox — `inbox_count` на `GET /dashboard`.
