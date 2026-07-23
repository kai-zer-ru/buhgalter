# Документация Бухгалтер

Справочники по установке, модели данных, соглашениям интерфейса и API. Стиль оформления — [style.md](style.md).

**Спецификация API:** [api/openapi.yaml](api/openapi.yaml) (интерактивно — `/docs` на запущенном сервере).

История изменений по версиям — [CHANGELOG.md](../CHANGELOG.md). Последний релиз: [release-notes-v1.4.0.md](release-notes-v1.4.0.md).

---

## Установка и эксплуатация

| Документ | Описание |
|----------|----------|
| [install/manual.md](install/manual.md) | Сборка и запуск бинарника |
| [install/docker.md](install/docker.md) | Docker и compose |
| [install/nginx.md](install/nginx.md) | Reverse proxy и HTTPS |
| [android-client.md](android-client.md) | Android-приложение: обзор |
| [android-client-ui.md](android-client-ui.md) | Android: интерфейс и навигация |
| [android-client-platform.md](android-client-platform.md) | Android: вход, офлайн, виджеты, i18n, discovery, логи, сборка |

Краткий обзор и переменные окружения — в [README.md](../README.md) в корне репозитория.

---

## Данные

| Документ | Описание |
|----------|----------|
| [data-model.md](data-model.md) | Схема БД, миграции, связи сущностей |
| [sql-access.md](sql-access.md) | Где писать SQL: sqlc vs inline, исключения, миграция legacy |
| [categories-and-icons.md](categories-and-icons.md) | Категории, подкатегории, иконки |
| [budget.md](budget.md) | Бюджет: помесячные лимиты, копирование, план vs факт, API |
| [../roadmap/budget-post-mvp.md](../roadmap/budget-post-mvp.md) | (план) Бюджет: rollover и прогноз из recurring |
| [transactions-display.md](transactions-display.md) | Отображение операций в UI и API |
| [../roadmap/multicurrency.md](../roadmap/multicurrency.md) | (план) Настоящая мультивалютность |
| [../roadmap/transaction-enrichment.md](../roadmap/transaction-enrichment.md) | (план) Теги, контрагент, split, вложения |
| [../roadmap/savings-reports-import.md](../roadmap/savings-reports-import.md) | (план) Цели накопления, PDF-отчёты, расширенный импорт |
| [accounts-archive-delete.md](accounts-archive-delete.md) | Архивация, мягкое удаление счётов, автоперевод остатка |
| [balance-maintenance.md](balance-maintenance.md) | Автопополнение банковских счетов по порогу баланса |

---

## UI-соглашения

Общие правила интерфейса — для разработчиков и при добавлении новых экранов.

| Документ | Описание |
|----------|----------|
| [date-time-display.md](date-time-display.md) | Форматы даты/времени и DateTimePicker |
| [ui-dialogs.md](ui-dialogs.md) | Диалоги, подтверждения, Esc |
| [ui-navigation.md](ui-navigation.md) | Хлебные крошки и кликабельные сущности |
| [ui-row-actions.md](ui-row-actions.md) | Меню «⋯» в строках и спойлер фильтров |
| [ui-stable-layout.md](ui-stable-layout.md) | Стабильная шапка и вкладки |
| [ui-empty-states.md](ui-empty-states.md) | Пустые состояния |
| [ui-table-columns.md](ui-table-columns.md) | Порядок колонок таблиц |
| [ui-pagination.md](ui-pagination.md) | Постраничная навигация (`TransactionPagination`) |
| [ui-toast.md](ui-toast.md) | In-app toast (успех, ошибка, предупреждение) |
| [ui-sveltekit-checks.md](ui-sveltekit-checks.md) | `resolve` / typed routes, vitest-моки, частые поломки check |
| [ui-stats.md](ui-stats.md) | Страница `/stats` |
| [../roadmap/stats-charts-net-worth.md](../roadmap/stats-charts-net-worth.md) | (план) Графики /stats и нетто-капитал |
| [ui-budget.md](ui-budget.md) | Страница `/budget`, форма, копирование, виджет на главной (спойлер категорий) |
| [ui-credits.md](ui-credits.md) | Страницы кредитов |
| [ui-credit-cards.md](ui-credit-cards.md) | Кредитные карты (тип счёта) |
| [ui-balance-maintenance.md](ui-balance-maintenance.md) | Автопополнение банковского счёта |

---

## API, кеш и импорт

| Документ | Описание |
|----------|----------|
| [api/openapi.yaml](api/openapi.yaml) | OpenAPI v1 |
| [api/authentication.md](api/authentication.md) | Сессии, API-токены, сброс пароля |
| [api/user-status.md](api/user-status.md) | Статус пользователя, модерация, блокировка |
| [notifications.md](notifications.md) | Уведомления: настройки, периоды, шаблоны, блокировка UI/API |
| [../roadmap/balance-shortfall-notifications.md](../roadmap/balance-shortfall-notifications.md) | Недостаток средств в тексте уведомлений |
| [ui-api-cache.md](ui-api-cache.md) | In-memory кеш GET на сервере и справочники в браузере |
| [import/cubux.md](import/cubux.md) | Импорт и экспорт формата Cubux |
| [../roadmap/category-rules-inbox.md](../roadmap/category-rules-inbox.md) | (план) Правила категорий и inbox неразнесённого |
