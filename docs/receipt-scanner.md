# Сканер чеков (54-ФЗ)

Импорт расходов из российских кассовых чеков: QR на чеке → API ФНС → подтверждение позиций → пакетное создание операций.

Связанные документы: [data-model.md](data-model.md), [transactions-display.md](transactions-display.md), [api/openapi.yaml](api/openapi.yaml).

## Поток

1. Пользователь сканирует QR **камерой** (QR в браузере → `POST /receipts/parse`) или загружает **фото чека** (`POST /receipts/scan-jobs`, async на сервере).
2. Сервер запрашивает данные у ФНС (цепочка бэкендов, см. ниже); для фото — после распознавания QR на сервере.
3. UI показывает магазин, дату, итог и таблицу позиций; пользователь выбирает счёт, категории (в т.ч. массово для выбранных строк).
4. `POST /api/v1/receipts/import` — в одной транзакции БД: магазин (`getOrCreate`), запись `receipt_imports`, N операций `expense`.
5. В списке операций у импортированных позиций — иконка чека; попап `GET /api/v1/receipts/{id}` с полным чеком и ссылкой на ФНС.

Фото обрабатывается асинхронно (как CSV-импорт): клиент опрашивает `GET /receipts/scan-jobs/{id}` до `done`/`failed`. Изображение не хранится. На сервере учитывается EXIF-поворот и несколько проходов распознавания (кроп нижней части, порог, масштаб).

## Ограничения v1

- Только QR 54-ФЗ (без OCR без QR, без зарубежных чеков).
- Нужен интернет для ФНС.
- Магазины создаются автоматически из чеков; ручного CRUD магазинов нет.
- Изображение чека не хранится — только JSON-снимок ответа ФНС в `receipt_imports.payload_json`.

## Формат description операции

Для каждой включённой позиции:

```
{название позиции}. Чек №{fiscal_document_number}, сумма {total_display}
```

`total_display` — итог **всего чека** (не позиции), в рублях с копейками.

## API

| Метод | Путь | Назначение |
|-------|------|------------|
| `POST` | `/api/v1/receipts/parse` | `{ "qr": "..." }` → preview (камера) |
| `POST` | `/api/v1/receipts/scan-jobs` | multipart `image` → `202`, async job |
| `GET` | `/api/v1/receipts/scan-jobs/{id}` | статус job; `result` — `ParsedReceipt` при `done` |
| `POST` | `/api/v1/receipts/import` | Подтверждение и создание операций |
| `GET` | `/api/v1/receipts/{id}` | Полные данные чека; query `transaction_id` — подсветка позиции |
| `GET` | `/api/v1/stores` | Список магазинов пользователя |

Повторный импорт того же чека (ключ `fiscal_drive_number` + `fiscal_document_number` + `fiscal_sign`) → **409** `ERR_RECEIPT_DUPLICATE`.

## UI

- Кнопка **«Чек»** в `NewTransactionButtons` на главной, `/transactions`, `/accounts/[id]`.
- Магазин в форме расхода (`TransactionForm`) — если есть хотя бы один магазин.
- Фильтр «Магазин» в операциях и статистике — только при `stores.length > 0`.
- Блок **«По магазинам»** на `/stats` — только расходы, при наличии данных.

## Зависимость от ФНС

Сервер пробует несколько источников (в порядке приоритета):

| Источник | Позиции | Доступ с сервера | Примечание |
|----------|---------|------------------|------------|
| `irkkt-mobile.nalog.ru:8888` (v2) | да | да | Нужен `BUHGALTER_FNS_SESSION_ID` (сессия приложения «Проверка чеков») |
| `proverkacheka.nalog.ru:9999` (GET) | да | часто нет (timeout) | Работает из некоторых сетей |
| `kkt-online.nalog.ru` (openapikkt) | нет | да | Только проверка подлинности чека |
| QR-fallback | 1 строка | всегда | После успешной проверки на kkt-online: сумма и дата из QR, позиция «Покупка по чеку» |

Переменная окружения `BUHGALTER_FNS_SESSION_ID` — `sessionId` из мобильного API ФНС (SMS-авторизация в приложении «Проверка чеков»). С ней сервер получает полный чек с позициями через `irkkt-mobile`.

### Как получить `BUHGALTER_FNS_SESSION_ID`

Это **не** сессия buhgalter (cookie в браузере). Это токен мобильного API ФНС (`irkkt-mobile.nalog.ru:8888`), тот же, что приложение «Проверка чеков» шлёт в заголовке `sessionId`.

**Важно:**

- Сессия **чувствительная** — как пароль к вашему номеру в API ФНС. Не коммитьте в git, храните только в `.env` на сервере.
- Обычно **живёт ~24 часа**, потом запросы к ФНС снова без позиций (fallback). Нужно обновить значение в `.env` и перезапустить buhgalter.
- API неофициальный (как у сторонних клиентов); ФНС может менять правила без предупреждения.

#### Способ 1 — перехват трафика приложения (рекомендуется)

Подходит для Android/iOS без root.

1. Установите официальное приложение **«Проверка чеков»** ФНС:
   - [Google Play](https://play.google.com/store/apps/details?id=ru.fns.billchecker) (`ru.fns.billchecker`)
   - [App Store](https://apps.apple.com/ru/app/id1169353005)
2. На телефоне войдите по номеру телефона и SMS-коду (как в обычном приложении).
3. Настройте перехват HTTPS на телефоне:
   - **Android:** [mitmproxy](https://mitmproxy.org/), [HTTP Toolkit](https://httptoolkit.com/) или Charles; установите CA-сертификат прокси (в Android 7+ для приложений иногда нужен root или эмулятор — проще эмулятор Android Studio).
   - **iOS:** Charles / Proxyman / mitmproxy + доверие сертификату в настройках.
4. В приложении отсканируйте любой чек (или откройте уже сохранённый).
5. В логах прокси найдите запрос к `irkkt-mobile.nalog.ru:8888`, например:
   - `POST /v2/ticket`
   - `GET /v2/tickets/...`
6. Скопируйте значение заголовка **`sessionId`** (длинная строка, не путать с cookie сайта ФНС).

Пример заголовка в mitmproxy:

```http
POST https://irkkt-mobile.nalog.ru:8888/v2/ticket
sessionId: a1b2c3d4e5f6....ваша_строка....
Content-Type: application/json
```

#### Способ 2 — файл настроек Android (root / эмулятор)

Если есть root или Android-эмулятор с доступом к данным приложения:

1. Войдите в «Проверка чеков» на устройстве.
2. Прочитайте файл (через `adb shell` с root или из бэкапа):

```text
/data/data/ru.fns.billchecker/shared_prefs/ru.fns.billchecker_preferences.xml
```

3. Найдите ключ **`KEY_ACCESS_TOKEN`** — его значение и есть `sessionId` для `BUHGALTER_FNS_SESSION_ID`.

На нерутованном телефоне иногда помогает `adb backup` (устаревающий способ; на новых Android часто недоступен).

#### Способ 3 — SMS-авторизация через API (для продвинутых)

Тот же поток, что у скриптов вроде [get-receipts](https://github.com/valiotti/get-receipts):

1. `POST https://irkkt-mobile.nalog.ru:8888/v2/auth/phone/request` — тело JSON: `phone`, `client_secret`, `os`.
2. Ввести код из SMS.
3. `POST https://irkkt-mobile.nalog.ru:8888/v2/auth/phone/verify` — `phone`, `code`, `client_secret`, `os`.
4. В ответе JSON: **`sessionId`** и **`refresh_token`**.

`client_secret` привязан к версии приложения и периодически меняется; его извлекают из APK «Проверка чеков» (декомпиляция / анализ `VERSION`). buhgalter **не** выполняет этот вход сам — только использует уже готовый `sessionId`.

Обновление без повторного SMS (пока жив `refresh_token`):

`POST https://irkkt-mobile.nalog.ru:8888/v2/mobile/users/refresh` с `refresh_token` и `client_secret` → новый `sessionId`.

Автообновление в buhgalter пока **не реализовано** — вручную копируйте новый `sessionId` в `.env`.

### Куда прописать и как проверить

**Локально / бинарник** — в `.env` рядом с `buhgalter` или в окружении:

```bash
BUHGALTER_FNS_SESSION_ID=ваш_sessionId_из_приложения
```

**Docker** — в `.env` рядом с `docker-compose.yml` (см. [docker/.env.example](../docker/.env.example)).

**Dev:**

```bash
export BUHGALTER_FNS_SESSION_ID='...'
make dev-server
```

Проверка без UI (из каталога `server/`):

```bash
BUHGALTER_FNS_SESSION_ID='...' go run ./cmd/fns-fetch/
```

Успех: в JSON несколько `items`, заполнен `store_name`. Ошибка `mobile unauthorized` — сессия пустая, просрочена или скопирована неверно.

Проверка через API buhgalter:

```bash
curl -sS -X POST 'http://localhost:8765/api/v1/receipts/parse' \
  -H 'Content-Type: application/json' \
  -H 'Cookie: session=ВАША_СЕССИЯ_BUHGALTER' \
  -d '{"qr":"t=20260706T1753&s=4047.50&fn=7384440900988131&i=6614&fp=0538715847&n=1"}'
```

В ответе должны быть реальные названия товаров, а не одна строка «Покупка по чеку».

Без session ID и при недоступности `:9999` импорт всё равно возможен: чек валидируется через kkt-online, в UI одна позиция на всю сумму (магазин пустой — можно указать вручную при импорте).

При полной недоступности ФНС — `502` `ERR_FNS_UNAVAILABLE`.
