# UI — пустые списки

Когда вкладка или фильтр списка не содержит записей, показывается **карточка** с текстом по центру — не голая строка под табами.

Связанно: загрузка и ошибка страницы — `PageLoadGate` + `reportPageLoadFailure` (`$lib/page-load.ts`).

## Компоненты

| Файл | Назначение |
|------|------------|
| `$lib/components/EmptyStateCard.svelte` | Карточка `.card` с `min-h-[7rem]`, текст `--text-muted`; слот `children` — только если нет кнопки создания в шапке |
| `$lib/components/PageLoadGate.svelte` | Loading / ошибка с «Повторить» / контент |

## Правила

1. **Всегда карточка** — `class="card"`, текст по центру (`text-center`, flex center).
2. **Текст в i18n** — ключи вида `*.empty` или `*.empty.<контекст>` (например `debts.empty.settled` для вкладки «Закрытые»).
3. **Шапка и табы остаются** — пустое состояние заменяет только тело списка (таблицу / сетку карточек). Кнопки создания в шапке не скрываются при смене вкладки — см. [ui-stable-layout.md](ui-stable-layout.md).
4. **Повторная загрузка при смене вкладки** — тот же блок с `message={$_('common.loading')}` и `ariaBusy` (см. `/accounts`, фильтр активные/архивные); не показывать empty-state с текстом «пусто» и не дублировать кнопки, пока идёт загрузка.
5. **Действие в пустом списке** — на `/accounts` **без** второй кнопки «Новый счёт» в карточке; создание — только из шапки. На других экранах слот `children` у `EmptyStateCard` — только если нет дублирующей кнопки в шапке.
6. **Загрузка страницы** — обернуть данные в `PageLoadGate` (`loading` / `error` / `onretry`), не оставлять пустой экран при ошибке API.

## Пример

```svelte
<script lang="ts">
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { _ } from 'svelte-i18n';
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if items.length === 0}
		<EmptyStateCard message={$_('debts.empty')} />
	{:else}
		<!-- таблица или сетка -->
	{/if}
</PageLoadGate>
```

## Где применено

### EmptyStateCard (пустые списки)

- `/` — пустой блок счетов
- `/accounts` — активные / архивные / удалённые
- `/debts`, `/debtors/[id]` — вкладки долгов
- `/credits` — активные / завершённые
- `/budget` — нет бюджетов за месяц
- `/stats` — пустые секции категорий / периодов / поиска (`TransactionList`)
- `/transactions` — через `TransactionList`
- `/settings/recurring-operations`, `/settings/categories`, `/settings/tokens`
- `/admin/backups` (архив), `/admin/users`

### PageLoadGate (загрузка / ошибка)

- Главная, счета, операции, долги, должник, кредиты (+ деталь), бюджет, статистика
- Настройки: профиль, категории, уведомления, токены, периодические операции
- Админка: система (`/admin`), пользователи, бэкапы, диагностика
- `/accounts/new`, контекстная сводка `TransactionContextStats`

Формы login / register / setup / import / password и модалки создания — без page-level gate (там action loading / toast).
