# Категории, подкатегории и иконки

Справочник доходов и расходов: CRUD в **Настройки → Категории** (`/settings/categories`).  
Иконки общие для категорий и подкатегорий — один каталог, один picker.

## Модель данных

| Таблица | Поле | Тип | Описание |
|---------|------|-----|----------|
| `categories` | `icon` | `TEXT` | ID иконки из каталога, по умолчанию `default` |
| `categories` | `type` | `income` \| `expense` | Тип категории |
| `categories` | `sort_order` | `INTEGER` | Порядок в списке (ручная сортировка в UI) |
| `categories` | `is_primary` | `INTEGER` | `1` — главная категория для типа (миграция `006_category_primary.sql`) |
| `categories` | `is_system` | `INTEGER` | `1` — системная категория, нельзя менять/удалять и создавать подкатегории (миграция `009_category_is_system.sql`, этап 4) |
| `subcategories` | `icon` | `TEXT` | ID иконки; при создании без `icon` в API — **иконка родительской категории**, иначе `default` (миграция `004_subcategory_icons.sql`) |
| `subcategories` | `category_id` | `TEXT` | Родительская категория |
| `subcategories` | `sort_order` | `INTEGER` | Порядок в списке (миграция `007_subcategory_sort_order.sql`) |

Подкатегория наследует **тип** от родителя (отдельного поля `type` нет). В UI picker фильтрует иконки по вкладке «Расходы» / «Доходы».

Пример: категория **Маркетплейсы** (`expense`, иконка `default` или `delivery`) → подкатегории **Ozon** (`ozon`), **Wildberries** (`wildberries`), **Яндекс Маркет** (`yandex-market`).

ER-диаграмма: [data-model.md](data-model.md).

## API

Префикс `/api/v1/`, авторизация — cookie `session`.

### Категории

| Метод | Путь | Тело |
|-------|------|------|
| GET | `/categories?type=income\|expense` | — |
| POST | `/categories` | `{ "name", "type", "icon", "sort_order"? }` — без `sort_order` → в конец списка |
| PUT | `/categories/order` | `{ "type", "ids": ["id", …] }` — сохранить ручной порядок |
| PUT | `/categories/{id}` | `{ "name", "icon", "sort_order"? }` |
| POST | `/categories/{id}/primary` | — — назначить главной для типа |
| DELETE | `/categories/{id}` | — |

Ответ категории: `id`, `name`, `type`, `icon`, `sort_order`, `is_primary`, `is_system`, `subcategory_count`, `created_at`.

**Системные категории** (`is_system: true`):

| Название | Тип | Иконка | Назначение |
|----------|-----|--------|------------|
| Долги | income, expense | `loan` | Операции по долгам (этап 4) |
| Кредиты | expense | `loan` | Платежи по кредитам (этап 5) |
| Комиссия | expense | `percent` | Комиссия за перевод между счетами (v1.1) |

Seed при создании пользователя + backfill при старте БД (`categoryseed.EnsureSystemCategories`). `PUT` / `DELETE` → `403 Forbidden`; `POST /categories/{id}/subcategories` → `403 Forbidden`; `POST /categories/{id}/primary` → `403 Forbidden`. В UI настроек — только просмотр, без раскрытия подкатегорий.

**Порядок:** список в API — пользовательские категории, затем системные (по `sort_order`, `name`). При `PUT /categories/order` системные всегда остаются в конце списка (клиент передаёт только id пользовательских категорий).

**Главная категория:** не более одной с `is_primary: true` на пользователя и тип; системные категории главными быть не могут. Новые пользовательские категории получают `sort_order = max + 1` среди пользовательских.

### Подкатегории

| Метод | Путь | Тело |
|-------|------|------|
| GET | `/categories/{id}/subcategories` | — |
| POST | `/categories/{id}/subcategories` | `{ "name", "icon"? }` — без `icon` → иконка **родительской категории** |
| PUT | `/categories/{id}/subcategories/order` | `{ "ids": ["id", …] }` — порядок подкатегорий |
| PUT | `/subcategories/{id}` | `{ "name", "icon"? }` — пустой `icon` сохраняет текущий |
| DELETE | `/subcategories/{id}` | — |

Ответ подкатегории: `id`, `category_id`, `name`, `icon`, `created_at`.

Бэкенд **не валидирует** ID иконки по whitelist — хранится произвольная строка; клиент предлагает только известные ID из каталога.

OpenAPI: [api/openapi.yaml](api/openapi.yaml).

## Каталог иконок

Источник правды: [`data/category_icons.json`](../data/category_icons.json).

```json
{
  "quick": {
    "expense": ["transport", "groceries", "food", "wildberries", "ozon", "yandex-market", "default"],
    "income": ["salary", "sale", "avito", "freelance", "rental-income", "bonus", "default"]
  },
  "icons": [
    {
      "id": "transport",
      "emoji": "🚌",
      "kind": "expense",
      "name": "Транспорт",
      "tags": ["транспорт", "автобус"]
    }
  ]
}
```

| Поле | Значение |
|------|----------|
| `kind` | `expense` — только расходы; `income` — только доходы; `both` — в обеих вкладках |
| `name` | Подпись по умолчанию для авто-имени; для `default` на расходах — «Разное», на доходах — «Прочие доходы» |
| `emoji` | Символ для SVG-заглушки |
| `official_logo` | `true` — SVG из `data/category_icons/{id}.svg` |
| `brand` | Цветная заглушка с текстом (устарело для брендов с `official_logo`) |

Сгенерированные файлы: `web/static/icons/categories/{id}.svg`  
URL в UI: `/icons/categories/{id}.svg` (`categoryIconUrl` в `web/src/lib/finance.ts`).

### Официальные логотипы (маркетплейсы, Авито)

| ID | Источник |
|----|----------|
| `wildberries` | `wildberries.ru/apple-touch-icon.png` |
| `ozon` | favicon `ozon.ru` (через `favicon.yandex.net`) |
| `yandex-market` | favicon `market.yandex.ru` |
| `avito` | `avito.ru/apple-touch-icon.png` |

Файлы: `data/category_icons/wildberries.svg`, `ozon.svg`, `yandex-market.svg`, `avito.svg`.

### Команды Make

| Команда | Назначение |
|---------|------------|
| `make download-marketplace-logos` | Скачать официальные логотипы WB/Ozon/Яндекс Маркет/Авито |
| `make generate-category-icons` | Собрать все SVG в `web/static/icons/categories/` |

Редактирование каталога:

1. Изменить [`scripts/build_category_icons_json.py`](../scripts/build_category_icons_json.py) (иконки, `kind`, `name`, quick-ряды).
2. `python3 scripts/build_category_icons_json.py` → обновит `data/category_icons.json`.
3. Для маркетплейсов при смене URL: `make download-marketplace-logos`.
4. `make generate-category-icons`.

### Git и CI

- [`data/category_icons.json`](../data/category_icons.json) **отслеживается в git** — в [`.gitignore`](../.gitignore) для `data/*` есть исключение `!/data/category_icons.json`, чтобы сборка и CI не зависели от локального прогона скрипта.
- В **CI** (job `build` в [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)) перед `make build` выполняется `python3 scripts/build_category_icons_json.py` — каталог всегда согласован со скриптом на runner.
- После правки [`scripts/build_category_icons_json.py`](../scripts/build_category_icons_json.py) закоммитьте обновлённый JSON вместе с изменениями скрипта.

Логотипы банков — отдельно: `data/banks_ru.json`, `make download-bank-logos` (см. [README](../README.md#содержание)).

## Поведение UI

Страница: `web/src/routes/settings/categories/+page.svelte`  
Компонент picker: `web/src/lib/components/CategoryIconPicker.svelte`.

| Поведение | Описание |
|-----------|----------|
| Вкладки | `?type=income` в URL сохраняется при обновлении страницы |
| Фильтр иконок | Быстрый ряд (7 шт.) и попап «Ещё» — только иконки с подходящим `kind` |
| Быстрый ряд | Иконка из «Ещё» встаёт **на 1 место**, последняя из дефолтного quick-ряда скрывается; повторный выбор из «Ещё» заменяет первую; клик по иконке в быстром ряду сбрасывает закрепление. Текущее значение вне quick (редактирование) тоже показывается первым |
| Авто-имя | Подставляется только пока поле имени **пустое** или совпадает с последним авто-именем. Сначала ввели название вручную — смена иконки **не трогает** текст; очистка поля снова включает автоподстановку |
| Редактирование | `lockName={true}` — смена иконки не перезаписывает существующее имя (категория / подкатегория) |
| Подкатегории — раскрытие | Клик по названию с **▶ / ▼ после текста** (одна кнопка); список подгружается лениво (`GET …/subcategories`) |
| Иконка категории | Декоративная, не кликабельна (без `btn-icon`) |
| Подкатегории — иконка | Тот же picker (компактный, 28px); **по умолчанию иконка родителя**; после добавления фокус остаётся в поле ввода без перезагрузки страницы |
| Подкатегории — состояние | Обновления `subs` и `expanded` через пересоздание объекта (`{ ...obj, [id]: … }`) — требование реактивности Svelte 5 |
| Сортировка | Перетаскивание за **⠿** (`web/src/lib/drag-reorder.ts`, `ReorderDragGhost.svelte`): ghost **следует за курсором** с **той же шириной**, что у карточки (фиксированные `width`/`height`); атрибут `data-drag-row` на всей строке; линия вставки на целевой карточке |
| Главная категория | Галочка у названия; «Сделать главной» — в меню «⋯» (`POST /categories/{id}/primary`); одна на вкладку; дефолт в форме операции |
| Enter | Сохраняет подкатегорию при редактировании; добавляет новую в форме создания |
| Удаление | `$lib/confirm` + i18n (`categories.confirm.delete`, `categories.confirm.deleteSub`) |

Логика авто-имени в picker: внутренние флаги `nameLocked` и `lastAutoName`; отличие текущего имени от `lastAutoName` трактуется как ручной ввод.

Дефолтные категории при регистрации: `server/internal/categoryseed/defaults.go` (без подкатегорий).

## Файлы (шпаргалка)

```
data/category_icons.json          # каталог ID, kind, name, tags
data/category_icons/*.svg         # официальные логотипы маркетплейсов
web/static/icons/categories/      # все SVG для фронта
web/src/lib/category-icons.ts     # фильтрация по kind, quickIconsDisplay, defaultCategoryNameForIcon
web/src/lib/components/CategoryIconPicker.svelte  # авто-имя, lockName
web/src/lib/drag-reorder.ts                        # pointer DnD, ghost с фиксированной шириной
web/src/lib/components/ReorderDragGhost.svelte
web/src/routes/settings/categories/+page.svelte
web/src/routes/accounts/+page.svelte              # вкладки active/archived, inline-edit, меню «⋯»
web/src/lib/settings/CategoriesTab.svelte         # раскрытие по названию, главная в меню «⋯»
web/src/lib/accounts.ts                           # defaultAccountId для форм
web/src/routes/+layout.svelte                     # шапка, порядок nav, ConfirmDialog
server/queries/categories.sql
server/internal/category/category.go              # CreateSubcategory — icon от родителя
```
