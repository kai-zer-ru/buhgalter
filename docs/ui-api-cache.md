# API — кеш GET-ответов

На **сервере** — in-memory кеш успешных `GET`. В **браузере** — in-memory TTL для банков + **localStorage SWR ref-cache** для всех `GET /api/v1/*` (мгновенный экран, фоновое обновление).

Связанные документы: [api/openapi.yaml](api/openapi.yaml).

---

## Реализация

| Файл | Назначение |
|------|------------|
| `server/internal/apicache/cache.go` | Хранилище, TTL |
| `server/internal/apicache/middleware.go` | Кеширование GET, инвалидация при POST/PUT/PATCH/DELETE |
| `server/internal/httpserver/server.go` | Подключение middleware к маршрутам API |

## TTL

| Тип данных | TTL |
|------------|-----|
| Справочники (`/banks`, `/categories`, `/debtors`, `/merchants`, `/tags`, `/transaction-templates`) | 5 мин |
| Остальные GET (счета, дашборд, операции, статистика и т.д.) | 1 мин |

TTL — страховка; при любой мутации кеш пользователя сбрасывается сразу.

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

На **клиенте** (ref-cache) дополнительно не кладутся в SWR: `GET /setup/status` (флаг регистрации на /login — иначе pre-mutation snapshot), `GET /sync/transaction-changes` (курсор обрабатывается отдельно и патчит уже лежащие списки операций). `GET /credits/{id}` и `GET /debtors/{id}` **кешируются** — офлайн-карточки в Android. Серверный кеш `GET /setup/status` остаётся; сброс при `PUT /admin/settings` и `PUT /admin/features`.

## Инвалидация

- Любой `POST` / `PUT` / `PATCH` / `DELETE` авторизованного пользователя — сброс всех ключей `u:{user_id}:*`
- `POST /setup`, restore — полная очистка кеша
- Logout, настройки, админка — через тот же middleware

## Клиент (браузер)

Два слоя:

| Слой | Что | TTL / поведение |
|------|-----|-----------------|
| In-memory | `GET /api/v1/banks` | 24 ч (`web/src/lib/api/cache.ts`) |
| **ref-cache (localStorage)** | `GET /api/v1/*` (кроме health, **setup/status**, export, preview, version, **sync/transaction-changes**) | **Stale-while-revalidate:** экран сразу из кеша, сеть в фоне. Включая `GET /credits/{id}` и `GET /debtors/{id}` (офлайн-карточки в Android). |

Ключ ref-cache: `buhgalter.ref_cache.web.v1::{user_id}::{path}` — при смене пользователя старый кеш не читается. Очистка при logout и session expired.

Фоновое обновление: `refCacheUpdate` (path-aware) → страницы перезагружают только затронутый блок; `assignIfChanged` не триггерит лишний re-render при идентичном JSON. `writeRefCache` **не пишет** и не уведомляет UI, если `JSON.stringify(next) ===` уже лежащая в памяти строка; для `/api/v1/dashboard` дополнительно сравнение по стабильному отпечатку **без** полей `*_display` (шум форматирования).

**Инвалидация на клиенте:** любой успешный `POST` / `PUT` / `PATCH` / `DELETE` через `client.ts` сбрасывает in-memory TTL (`invalidateApiCache`). **Web** — `clearRefCache` (последующий `load()` идёт в сеть, не pre-mutation snapshot). **Android** — `clearRefCache({ preserveAuthMe: true })` **оставляет** все persistable GET (дашборд, операции, долги, карточки должников/кредитов/счетов, статистика, словари, `/auth/me`); следующий онлайн-GET по этим путям форсирует сеть (`pendingNetworkRefresh`), офлайн продолжает читать снимок. Списки кредитов после write **не** засеваются одним обновлённым кредитом (`onCreditUpdated` патчит только уже лежащий список); ручной sync обходит SWR и перечитывает GET с сервера. После прогрева Android запрашивает `GET /sync/transaction-changes` (курсор `since_id` в ref-cache) и патчит кеш операций по id — добавление/правка/удаление (в т.ч. старая операция, долг/кредит) подтягивается, даже если URL списка не входил в warm. Если `/accounts` пуст — заполнение из `ui/meta` (`seedAccountsFromUIMetaIfEmpty`), иначе офлайн нельзя выбрать счёт в форме операции. Словари только перезаписываются свежим GET / `seedDictionariesFromUIMeta`. Дополнительно Android хранит профиль в `buhgalter.last_user.v1` (не привязан к URL сервера). `GET /debtors/{id}` кешируется и при miss собирается из списков долгов.

Прогрев при входе: `warmRefCache()` в фоне после `loadUser()`.

Реализация: `web/src/lib/ref-cache.ts`, хук в `client.ts` `request()`, `state-utils.ts` (`assignIfChanged`).

Иконки банков и категорий — статические файлы в `web/static/`, кешируются браузером.
