# Android — точечное обновление UI при refresh данных

**Версия:** v1.4.0 · **Статус:** реализовано (Android + web SWR).

**Веб:** тот же SWR ref-cache в `web/src/lib/ref-cache.ts` (ключ с `user_id`), без outbox.

Документация клиента: [android-client-ui.md](../docs/android-client-ui.md), [android-client-platform.md](../docs/android-client-platform.md).

## Зачем

При фоновом обновлении данных (SWR ref-cache, sync outbox, `refCacheTick`) пользователь не должен ощущать «перерисовку всей страницы». Достаточно обновить изменившиеся цифры, добавить новую строку в список, убрать удалённую — без спиннера, без мигания всего экрана и без сброса раскрытых спойлеров / режима редактирования.

## Уже есть (база)

Часть поведения уже заложена в v1.4.0:

| Механизм | Эффект |
|----------|--------|
| `{#each items as item (item.id)}` | Svelte патчит DOM по ключам — новые/удалённые строки, а не весь список |
| `load({ background: true })` | Фоновый refresh **без** `PageLoadGate` / полноэкранного «Загрузка…» |
| `refCacheTick` | Срабатывает только если JSON в кеше **реально изменился** |
| Убран `{#key dataRefreshTick}` из layout | Страница не перемонтируется целиком при sync |
| `outboxTick` / `localDataTick` | Мгновенный пересчёт балансов и списков из outbox до ответа сервера |
| SWR (`fetchWithRefCache`) | Сначала кеш на экране, сеть — в фоне |

То есть при смене баланса `-500` → `-520` теоретически обновляется только `MoneyDisplay` в нужной карточке; при добавлении операции — одна новая строка в `TransactionList`.

## Что ещё ощущается как «полная перерисовка»

1. **Смена вкладки/фильтра** — `filterLoading` + `opacity-60` на весь блок (намеренно; не относится к фоновому SWR).
2. **Фоновый `load()` перезаписывает целые массивы** — `accounts = list`, `dashBase = …`; Svelte пересчитывает дерево, даже если изменилось одно поле.
3. **Нет patch по полям** — API отдаёт целый объект; не обновляем только `balance_display`.
4. **Глобальный `refCacheTick`** — любое изменение в ref-cache дергает `load()` на открытой странице, даже если изменился другой endpoint (например dashboard vs accounts).

## План доработок

По нарастающей сложности:

### 1. Не присваивать state, если данные не изменились

Хелпер `assignIfChanged(prev, next)` (сравнение через `JSON.stringify` или shallow compare) перед записью в `$state` в `load({ background: true })`.

**Эффект:** меньше лишних reactive-прогонов при идентичном ответе сервера.

**Файлы:** новый `android/ui/src/lib/state-utils.ts` (или расширение `page-load.ts`); главная, счета, операции, кредиты, долги.

### 2. Подписка на конкретный path кеша (реализовано)

Вместо одного глобального `refCacheTick` — `refCacheUpdate` с path (`/api/v1/dashboard`, `/api/v1/accounts?status=active`, …). Хелпер `refCachePathMatches` в `ref-cache-watch.ts`.

**Эффект:** главная реагирует только на dashboard; список счетов — только на accounts.

**Файлы:** `ref-cache.ts`, `$effect` на страницах.

### 3. Независимые виджеты на экране

Разбить тяжёлые страницы (главная, `/accounts/[id]`) на блоки со своим `$state` и своим `load()`: сводка, группы счетов, блок «последние операции».

**Эффект:** обновление транзакций не трогает карточки счетов.

### 4. Анимация только новых элементов

`flip` / `slide` на `{#each}` для появления/исчезновения строк (опционально, только Android).

**Эффект:** визуально видно «добавилась строка», а не «мигнул весь список».

### 5. Entity-store (долгосрочно)

`Map<id, Account>` с точечным `patch(id, { balance_display })` — максимально хирургично, больше кода и тестов.

## Рекомендуемый первый шаг (MVP в рамках v1.4.0)

1. `assignIfChanged` + unit-тест.
2. Применить в `load({ background: true })` на: `+page.svelte` (главная), `accounts/+page.svelte`, `transactions/+page.svelte`.
3. Убедиться, что `filterLoading` / `opacity-60` **не** включаются при `background: true` (аудит остальных страниц).

## Критерии готовности

- [x] Фоновый SWR на главной: меняется только сумма в карточке счёта, спойлеры и скролл не сбрасываются.
- [x] Фоновый refresh списка операций: новая строка появляется без `opacity-60` на всём блоке.
- [x] `refCacheUpdate` по изменению чужого endpoint не вызывает `load()` на текущей странице.
- [x] Документация в [android-client-platform.md](../docs/android-client-platform.md) и [ui-api-cache.md](../docs/ui-api-cache.md) обновлена.

## Связанные документы

- [android-client.md](android-client.md) — обзор Android-релиза v1.4.0
- [android-client-ui.md](../docs/android-client-ui.md) — `PageLoadGate`, ошибки загрузки
- [ui-empty-states.md](../docs/ui-empty-states.md) — пустые списки (отдельно от ошибки загрузки)
