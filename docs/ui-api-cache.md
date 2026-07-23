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
| Справочники (`/banks`, `/categories`, `/debtors`) | 5 мин |
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

На **клиенте** (ref-cache) дополнительно не кладутся в SWR: `GET /setup/status` (флаг регистрации на /login — иначе pre-mutation snapshot), `GET /credits/{id}` (полный график). Серверный кеш `GET /setup/status` остаётся; сброс при `PUT /admin/settings`.

## Инвалидация

- Любой `POST` / `PUT` / `PATCH` / `DELETE` авторизованного пользователя — сброс всех ключей `u:{user_id}:*`
- `POST /setup`, restore — полная очистка кеша
- Logout, настройки, админка — через тот же middleware

## Клиент (браузер)

Два слоя:

| Слой | Что | TTL / поведение |
|------|-----|-----------------|
| In-memory | `GET /api/v1/banks` | 24 ч (`web/src/lib/api/cache.ts`) |
| **ref-cache (localStorage)** | `GET /api/v1/*` (кроме health, **setup/status**, export, preview, version, **`GET /credits/{id}`**) | **Stale-while-revalidate:** экран сразу из кеша, сеть в фоне. Деталь кредита с полным графиком не кладётся в localStorage — иначе на мобиле возможен долгий freeze главного потока. |

Ключ ref-cache: `buhgalter.ref_cache.web.v1::{user_id}::{path}` — при смене пользователя старый кеш не читается. Очистка при logout и session expired.

Фоновое обновление: `refCacheUpdate` (path-aware) → страницы перезагружают только затронутый блок; `assignIfChanged` не триггерит лишний re-render при идентичном JSON.

**Инвалидация на клиенте:** любой успешный `POST` / `PUT` / `PATCH` / `DELETE` через `client.ts` сбрасывает ref-cache и in-memory TTL (`clearRefCache` + `invalidateApiCache`), чтобы последующий `load()` на странице шёл в сеть, а не рисовал pre-mutation snapshot. In-flight SWR revalidate после сброса не записывает устаревший ответ (epoch). То же правило в **Android** (`android/ui/src/lib/api/client.ts`) — с `clearRefCache({ preserveAuthMe: true })`, чтобы офлайн cold start всё ещё находил `/auth/me` для PIN/биометрии. Дополнительно Android хранит профиль в `buhgalter.last_user.v1` (не привязан к URL сервера).

Прогрев при входе: `warmRefCache()` в фоне после `loadUser()`.

Реализация: `web/src/lib/ref-cache.ts`, хук в `client.ts` `request()`, `state-utils.ts` (`assignIfChanged`).

Иконки банков и категорий — статические файлы в `web/static/`, кешируются браузером.
