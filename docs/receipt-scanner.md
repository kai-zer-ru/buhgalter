# Сканер чеков (54-ФЗ) — v1.10

Импорт расходов и **прикрепление чека** к уже созданной операции по QR кассового чека РФ. Состав позиций — через **proverkacheka.com** (токен инстанса в админке), не через «бесплатный» API ФНС с VPS.

Связанные документы: [data-model.md](data-model.md), [transactions-display.md](transactions-display.md), [feature-toggles.md](feature-toggles.md), [api/openapi.yaml](api/openapi.yaml), [README](../README.md#сканер-чеков-токен-proverkacheka), [receipt-scanner-proverkacheka.md](receipt-scanner-proverkacheka.md).

## Включение на инстансе

| Условие | Где |
|--------|-----|
| Feature flag `receipt_scanner` | Админка → модули ([feature-toggles.md](feature-toggles.md)) |
| Токен API proverkacheka | Админка → **API чеков** (write-only; см. README) |

**Эффективно модуль работает**, только если **оба** условия выполнены. Иначе клиенты не показывают «Чек» / attach; API `/receipts/*` → **404** `FEATURE_DISABLED` или **503** `ERR_RECEIPT_API_NOT_CONFIGURED`.

Инструкция получения токена у провайдера — в **[README](../README.md#сканер-чеков-токен-proverkacheka)** (открытая дока репозитория).

## Поток: новый импорт

1. QR читается **на устройстве** (web и Android): кнопка камеры и загрузка файла. Дальше `POST /api/v1/receipts/parse` с `{ "qr": "..." }` — на сервер уходит строка, не картинка. Сервер по ней запрашивает состав у proverkacheka. `POST /receipts/scan-jobs` клиенты для этого не вызывают.
2. Сервер вызывает **proverkacheka** `POST /api/v1/check/get` (`token` + `qrraw`), нормализует ответ в preview (`ParsedReceipt`: магазин, дата, итог, `items[]`). Успешный ответ сохраняется в **`receipt_fiscal_previews`** (ключ user + fn + i + fp); повторный preview того же чека идёт из БД без запроса к провайдеру.
3. UI подтверждения import (web и Android):
   - **Счёт** для всех создаваемых расходов.
   - Таблица позиций: вкл/выкл строку, название, сумма и одна кнопка. Она открывает окно с категорией и подкатегорией (expense, не system) и созданием подкатегории. Пустая подпись — «Выберите категорию»; после выбора — имя категории и, если подкатегория есть, « · » и её имя. Список без горизонтального скролла. Выбор сразу в строке; закрытие окна его не откатывает.
   - Блок **«Общая категория»**: категория + подкатегория (опционально).
   - Кнопка **«Проставить без категории»** — для всех **включённых** строк, у которых категория ещё не выбрана, подставляет общую категорию/подкатегорию; уже заданные вручную **не перезаписывает** (удобно для смешанных чеков: продукты + лекарства + бытовое).
   - Подсказка: сначала массово однотипное, затем вручную отдельные позиции.
   - Сохранение недоступно, пока у каждой **включённой** строки нет категории.
4. `POST /api/v1/receipts/import` — в одной транзакции БД: `stores` (getOrCreate), `merchant_id` по имени точки в каталоге `merchants` (то же разрешение, что у ручной операции: имя или алиас), `receipt_imports`, N операций `expense`. Пустое имя точки магазин не создаёт. Одинаковое имя или алиас не создаёт второй магазин. Ошибка разрешения магазина откатывает import.
5. В журнале — иконка чека; попап `GET /api/v1/receipts/{id}?transaction_id=…` с подсветкой позиции и **категорией/подкатегорией** по каждой строке (из связанных операций).

### Категория в журнале (import и attach)

Для всех операций с одним `receipt_import_id` колонка категории строится **по всем позициям чека** (операции import с `receipt_item_index` и/или строки `receipt_import_item_categories` при attach):

| Ситуация | Что показываем |
|----------|----------------|
| Одна категория, **разные** подкатегории | **Только категория** |
| **Разные** категории | **Все категории** (через запятую) |
| Одна категория и **одна** подкатегория на всех позициях | **Категория и подкатегория** |

Повторный **import** того же чека (ключ `fiscal_drive_number` + `fiscal_document_number` + `fiscal_sign`) → **409** `ERR_RECEIPT_DUPLICATE`.

## Поток: прикрепить к существующей операции

Отдельный сценарий: **не** создаёт новые операции, только связывает чек с уже созданным **расходом**.

### Условия

- `type = expense`, без `receipt_import_id`.
- Модуль включён (flag + токен).

### UX (web и Android)

1. Меню строки — **«Прикрепить чек»** (как подписка / платёж по кредиту).
2. Scan / фото → preview, как при import.
3. Таблица позиций: одна кнопка на строку — то же окно категории и подкатегории, что у import. Блок **«Общая категория»** и **«Проставить без категории»** остаются над списком (как import). По умолчанию подставляется категория/подкатегория **текущей операции**, если есть: кнопка сразу показывает эти имена.
4. **Сумма:** `transactions.amount` (коп.) vs `receipt.total_sum` (коп.) — **строгое** равенство.
5. Совпало — подтверждение; не совпало — [ui-dialogs.md](ui-dialogs.md): **«Добавить всё равно»** / **«Отмена»**. Сохранение недоступно, пока у **каждой** позиции чека не выбрана категория.
6. Успех: `receipt_import_id` на операции; `merchant_id` заполняется по имени точки, только если у расхода магазина ещё не было; категории строк — в `receipt_import_item_categories`; `receipt_item_index` на операции **не** задаётся (одна операция на весь чек).
7. **Статистика расходов:** подкатегории attach-строк попадают в `GET /stats/by-subcategory` (сумма по позиции из payload чека). Import — через `subcategory_id` на каждой созданной операции. См. [ui-stats.md](ui-stats.md).
8. **Открепить чек:** меню строки или кнопка в просмотре чека → `POST /api/v1/receipts/detach` с `{ "transaction_id" }`; сбрасывает `receipt_import_id` / `receipt_item_index` на **этой** операции; `merchant_id` и `store_id` не меняет; при отсутствии других связей категории строк attach удаляются.

### API attach

| Метод | Путь |
|-------|------|
| `POST` | `/api/v1/receipts/attach` |

Тело:

```json
{
  "transaction_id": "uuid",
  "fiscal_drive_number": "…",
  "fiscal_document_number": "…",
  "fiscal_sign": "…",
  "receipt": { },
  "items": [
    { "index": 0, "category_id": "uuid", "subcategory_id": null, "include": true }
  ],
  "acknowledge_sum_mismatch": false
}
```

Поле `items` — по одной записи на **каждую** позицию в `receipt.items`; `include` для attach всегда `true`.

| Код | Когда |
|-----|--------|
| **409** `ERR_RECEIPT_SUM_MISMATCH` | Суммы различаются и `acknowledge_sum_mismatch !== true`; тело с `transaction_amount`, `receipt_total_sum` (коп.) |
| **404** | Операция не найдена / чужая |
| **409** `ERR_RECEIPT_DUPLICATE` | Import — чек с таким fiscal-ключом уже есть; attach — чек уже привязан к другой операции |

Ответ: `{ "receipt_import_id", "transaction": { … } }`.

## API (сводка)

| Метод | Путь | Назначение |
|-------|------|------------|
| `POST` | `/api/v1/receipts/parse` | Preview по строке QR |
| `POST` | `/api/v1/receipts/scan-jobs` | Фото → async job |
| `GET` | `/api/v1/receipts/scan-jobs/{id}` | Статус job |
| `POST` | `/api/v1/receipts/import` | Import → N расходов |
| `POST` | `/api/v1/receipts/attach` | Attach к одной операции |
| `POST` | `/api/v1/receipts/detach` | Открепить чек от операции |
| `GET` | `/api/v1/receipts/{id}` | Детали чека |
| `GET` | `/api/v1/stores` | Точки чека (`stores`); в UI не показывается |
| `PUT` | `/api/v1/admin/settings/receipt-api-token` | Задать токен (admin) |

Ошибки модуля: `ERR_RECEIPT_API_NOT_CONFIGURED`, `ERR_RECEIPT_QR`, `ERR_RECEIPT_NOT_FOUND`, `ERR_RECEIPT_DUPLICATE`, `ERR_RECEIPT_SUM_MISMATCH`, `ERR_RECEIPT_ITEMS`, `ERR_FNS_UNAVAILABLE` (legacy / провайдер недоступен).

## UI (web)

- **«Чек»** в `NewTransactionButtons` (главная, `/transactions`, `/accounts/[id]`) — при включённом модуле.
- **«Прикрепить чек»** в меню строки расхода.
- Магазин чека — тот же каталог, что у ручной операции. В форме одно поле в спойлере «Время, комментарий, магазин, теги»; отдельного селекта точки нет. Фильтра «Точка из чека» нет. Блок **«По магазинам»** на `/stats` считает `merchant_id` (ручные расходы и чеки) и остаётся на `GET /stats/by-store`: в ответе по-прежнему `store_id` и `store_name`, это id и имя магазина. Блок виден, когда в ответе есть строки.

Android — те же сценарии в `android/ui/` ([android-client-receipts.md](android-client-receipts.md)): full-screen `/receipts/import`, `/receipts/attach`, `/receipts/[id]`; кнопка «Чек» на главной / операциях / счёте; **online-only**.

## Модель данных

- `stores`, `receipt_imports`, `receipt_fiscal_previews`, `receipt_scan_jobs` — см. [data-model.md](data-model.md).
- `transactions.store_id` — точка чека; `merchant_id` — каталог магазинов. Import пишет оба на каждую включённую строку. Attach пишет `store_id` как раньше и заполняет `merchant_id`, только если он ещё пуст. Detach не очищает ни магазин, ни точку.
- `receipt_import_id`, `receipt_item_index`.
- Токен API: `system_settings` (шифрование как `notification_secret_key`); в GET админки только `receipt_api_token_set`.

## Формат description (import по позициям)

```
{название позиции}. Чек №{fiscal_document_number}, сумма {total_display}
```

`total_display` — итог **всего чека** в рублях с копейками.

## Позже: ИИ и вложение фото

Не входит в v1.10. В [общих планах](../ROADMAP.md#общие-планы).

- **ИИ по фото** — распознать чек без читаемого QR или подсказать категории по снимку. В v1.10 QR читается на устройстве, состав позиций даёт только proverkacheka.
- **Вложение снимка к операции** — хранить фото чека рядом с операцией. В v1.10 файл на сервер не уходит; в БД остаётся JSON в `receipt_imports.payload_json`.

## Ограничения

- QR 54-ФЗ; зарубежные чеки — вне scope.
- Сколько чеков можно запросить у proverkacheka, задаёт тариф провайдера, не buhgalter.
- На публичном [демо-стенде](https://buhgalter-demo.kai-zer.ru/) сканер чеков не работает: лимита proverkacheka.com не хватает на общий стенд. На своём инстансе нужен свой токен.
- Без токена proverkacheka — **нет** состава позиций (не обещать silent fallback «Покупка по чеку» в v1.10).
- Фото чека на сервере не хранится; в БД — JSON снимок в `receipt_imports.payload_json`.
- Нет ручного CRUD точек `stores`. Магазин чека — каталог `merchants`.

## Legacy (до v1.10, не развивать)

Ранее использовались цепочка ФНС (`BUHGALTER_FNS_SESSION_ID`, `:9999`, kkt-online) и fallback «Покупка по чеку». Для self-host это **не** целевой путь v1.10. Подробности для миграции старых инстансов — [receipt-scanner-proverkacheka.md](receipt-scanner-proverkacheka.md) (§ Legacy). Env `BUHGALTER_FNS_SESSION_ID` в [docker/.env.example](../docker/.env.example) — только справка, не замена токена proverkacheka.
