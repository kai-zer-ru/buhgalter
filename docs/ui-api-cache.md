# API — кеш GET-ответов

На **сервере** — in-memory кеш успешных `GET`. В **браузере** — in-memory TTL для банков + **localStorage SWR ref-cache** для всех `GET /api/v1/*` (мгновенный экран, фоновое обновление).

Связанные документы: [api/openapi.yaml](api/openapi.yaml). План уйти от фонового шторма GET — [realtime-updates.md](../roadmap/realtime-updates.md).

---

## Реализация

| Файл | Назначение |
|------|------------|
| `server/internal/apicache/cache.go` | Хранилище, TTL, эпоха |
| `server/internal/apicache/middleware.go` | Кеширование GET (`SetIfEpoch`), инвалидация при успешных POST/PUT/PATCH/DELETE (2xx) |
| `server/internal/httpserver/server.go` | Подключение middleware к маршрутам API |

## TTL

| Тип данных | TTL |
|------------|-----|
| Справочники (`/banks`, `/categories`, `/debtors`, `/merchants`, `/tags`, `/transaction-templates`) | 5 мин |
| Остальные GET (счета, дашборд, операции, статистика и т.д.) | 1 мин |

TTL — страховка; при **успешной** мутации (ответ 2xx) кеш пользователя сбрасывается сразу. Ошибки 4xx/5xx кеш не трогают — данные на сервере не менялись.

GET, начатый **до** мутации, не должен снова заполнить кеш **после** инвалидации: иначе следующий запрос до 1 мин отдаёт старый дашборд/список (операция «есть на одном экране и нет на другом»). У кеша есть **эпоха**: ответ GET кладётся через `SetIfEpoch` только если с момента miss не было `InvalidateUser` / `Clear`. На клиенте то же для in-flight GET и фонового SWR: `writeRefCache` пропускается, если за время запроса выросли `cacheEpoch` (`clearRefCache` или `invalidateRefCachePrefix`).

## Кешируемые GET

- `GET /banks`, `GET /setup/status`
- Все авторизованные `GET` в `/api/v1/*` (кроме исключений ниже)

Ключ: `u:{user_id}:{path}?{query}` (для публичных — `g:...`).

## Без кеша

| Эндпоинт | Причина |
|----------|---------|
| `GET /health` | Диагностика |
| `GET /version/check` | Собственный кеш в `versioncheck` |
| `GET /export` | Файловая выгрузка |
| `POST .../preview`, `GET .../preview` | Разовые расчёты (в т.ч. `GET /budgets/spent-preview`) |
| `GET /import/jobs/{id}` | Статус меняется |
| `GET /sync/transaction-changes` | Курсорная лента изменений операций; не должна отдавать устаревший пустой пакет |
| `GET /realtime` | WebSocket upgrade |

На **клиенте** (ref-cache) дополнительно не кладутся в SWR: `GET /setup/status` (флаг регистрации на /login — иначе pre-mutation snapshot), `GET /version/check` (версия сборки сервера — иначе stale SWR + суточный throttle клиента фиксировали старый `current_version`), `GET /sync/transaction-changes` (курсор обрабатывается отдельно и патчит уже лежащие списки операций), `GET /realtime`. `GET /credits/{id}` и `GET /debtors/{id}` **кешируются** — офлайн-карточки в Android. Серверный кеш `GET /setup/status` остаётся; сброс при `PUT /admin/settings` и `PUT /admin/features`.

## Инвалидация

- Успешный (2xx) `POST` / `PUT` / `PATCH` / `DELETE` авторизованного пользователя — сброс всех ключей `u:{user_id}:*`
- Исключение: `POST /api/v1/import/jobs` — кеш не сбрасывается (фоновый commit ещё не меняет данные). Сброс на сервере (`InvalidateUser`) и на клиенте — когда job `done`/`failed`
- `POST /setup`, restore — полная очистка кеша
- Logout, настройки, админка — через тот же middleware

## Клиент (браузер)

Два слоя:

| Слой | Что | TTL / поведение |
|------|-----|-----------------|
| In-memory | `GET /api/v1/banks` | 24 ч (`web/src/lib/api/cache.ts`) |
| **ref-cache (localStorage)** | `GET /api/v1/*` (кроме health, **setup/status**, export, preview, version, **sync/transaction-changes**, **realtime**) | **Stale-while-revalidate:** экран сразу из кеша, сеть в фоне. Включая `GET /credits/{id}` и `GET /debtors/{id}` (офлайн-карточки в Android). **Web:** при живом WebSocket (`GET /realtime`) фоновый revalidate на каждый GET **выключен**; иначе cooldown **60 с** + отложенный старт 3 с, скрытая вкладка не revalidate. **Android:** cooldown 60 с (сокет пока нет). |

Ключ ref-cache: `buhgalter.ref_cache.web.v1::{user_id}::{path}` — при смене пользователя старый кеш не читается. Очистка при logout и session expired.

Фоновое обновление: `refCacheUpdate` (path-aware) → страницы перезагружают только затронутый блок; `assignIfChanged` не триггерит лишний re-render при идентичном JSON. `writeRefCache` **не пишет** и не уведомляет UI, если `JSON.stringify(next) ===` уже лежащая в памяти строка; для `/api/v1/dashboard` дополнительно сравнение по стабильному отпечатку **без** полей `*_display` (шум форматирования).

**Инвалидация на клиенте:** любой успешный `POST` / `PUT` / `PATCH` / `DELETE` через `client.ts` сбрасывает in-memory TTL (`invalidateApiCache`), кроме `POST /api/v1/import/jobs` (сброс после `done`/`failed`). **Web** — полный `clearRefCache()` (последующий `load()` идёт в сеть, не pre-mutation snapshot; словари при необходимости снова прогревает `warmRefCache` / `getUIMeta`). **Не** писать счета из `ui/meta` в ключи SWR `/api/v1/accounts*` — у meta нет балансов (`0.00`), а при живом WebSocket фоновый revalidate выключен, и список счетов «застывает» на нулях. Офлайн-fallback формы читает meta через `readAccountsFromOfflineCache`, без записи в SWR. **Android** — `clearRefCache({ preserveAuthMe: true })` **оставляет** все persistable GET (дашборд, операции, долги, карточки должников/кредитов/счетов, статистика, словари, `/auth/me`); следующий онлайн-GET по этим путям форсирует сеть (`pendingNetworkRefresh`), офлайн продолжает читать снимок. Исключение: `DELETE /api/v1/user/data` полностью сбрасывает ref-cache, иначе после сброса учёта остались бы старые снимки ledger. Списки кредитов после write **не** засеваются одним обновлённым кредитом (`onCreditUpdated` патчит только уже лежащий список); ручной sync обходит SWR и перечитывает GET с сервера. После прогрева Android запрашивает `GET /sync/transaction-changes` (курсор `since_id` в ref-cache) и патчит кеш операций по id — добавление/правка/удаление (в т.ч. старая операция, долг/кредит) подтягивается, даже если URL списка не входил в warm. Если `/accounts` пуст — заполнение из `ui/meta` (`seedAccountsFromUIMetaIfEmpty`) на **Android**; на вебе seed в SWR отключён (см. выше). Словари только перезаписываются свежим GET / `seedDictionariesFromUIMeta`. Дополнительно Android хранит профиль в `buhgalter.last_user.v1` (не привязан к URL сервера). `GET /debtors/{id}` кешируется и при miss собирается из списков долгов.

Прогрев при входе: `warmRefCache()` в фоне после `loadUser()`. Веб дополнительно открывает `GET /api/v1/realtime` (WebSocket). **Первый** `onopen` сокета сбрасывает ledger-пути (`/dashboard`, `/accounts*`, операции, бюджеты) и soft-reload — иначе SWR с прошлой сессии при живом сокете (revalidate выключен) мог показывать устаревшие балансы на главной, пока страница счёта уже взяла свежий `/accounts/{id}/balance`. По `invalidate` с **`hint_paths`** — сброс только этих путей в ref-cache и `refCacheUpdate` с `paths` (страницы soft-reload только совпадающие блоки). Без `hint_paths` — грубый `notifyRealtimeInvalidate` (полный `clearRefCache` + `path: '*'`). После reconnect — догон `GET /sync/transaction-changes` и soft-reload экрана. Запись `GET /dashboard` в ref-cache патчит сохранённые `/accounts*` и `/accounts/{id}/balance` (как на Android). `listAccounts` / `getAccount` обогащают балансы из последнего dashboard. Импорт: события `import.progress` / `done` / `failed` по сокету; poll 1,2 с — fallback.

Реализация: `web/src/lib/ref-cache.ts`, `web/src/lib/realtime.ts`, хук в `client.ts` `request()`, `state-utils.ts` (`assignIfChanged`). План/детали — [realtime-updates.md](../roadmap/realtime-updates.md).

Иконки банков и категорий — статические файлы в `web/static/`, кешируются браузером.
