# Документация Buhgalter

Публичные справочники по данным, UI, API и установке. Стиль оформления — [style.md](style.md).

**Актуальная спецификация API:** [api/openapi.yaml](api/openapi.yaml).

---

## Релизы

| Документ | Описание |
|----------|----------|
| [release-notes-v1.2.2.md](release-notes-v1.2.2.md) | Проверка обновлений (админ), кнопки доход/расход/перевод, форма операции, статистика по категориям |
| [release-notes-v1.2.1.md](release-notes-v1.2.1.md) | Docker, логирование, ипотека, API `error.field`, UI (счета, категории, меню «⋯», редактирование операций, фильтры, кеш, статистика, погашение долга, оплата кредита) |
| [release-notes-v1.2.0.md](release-notes-v1.2.0.md) | Периодические операции, кредиты, ипотека MVP, setup restore |
| [release-notes-v1.1.1.md](release-notes-v1.1.1.md) | `.env`, документация и улучшения UI |
| [release-notes-v1.1.md](release-notes-v1.1.md) | Сброс пароля, кредиты, комиссия переводов, mobile UI |
| [../CHANGELOG.md](../CHANGELOG.md) | Полный changelog |

---

## Данные и домен

| Документ | Описание |
|----------|----------|
| [data-model.md](data-model.md) | ER-диаграмма, SQL, миграции |
| [categories-and-icons.md](categories-and-icons.md) | Категории, подкатегории, иконки |
| [transactions-display.md](transactions-display.md) | Отображение операций в UI |

## UI-соглашения

| Документ | Описание |
|----------|----------|
| [ui-dialogs.md](ui-dialogs.md) | Диалоги, подтверждения, Esc |
| [ui-navigation.md](ui-navigation.md) | Навигация и кликабельные сущности |
| [ui-row-actions.md](ui-row-actions.md) | Меню «⋯» в строках и спойлер фильтров |
| [ui-api-cache.md](ui-api-cache.md) | In-memory кеш справочников в веб-клиенте |
| [ui-stats.md](ui-stats.md) | Страница `/stats` |
| [ui-credits.md](ui-credits.md) | UI кредитов |
| [ui-stable-layout.md](ui-stable-layout.md) | Стабильная шапка и вкладки |
| [ui-empty-states.md](ui-empty-states.md) | Пустые состояния |
| [ui-table-columns.md](ui-table-columns.md) | Порядок колонок таблиц |

## API и импорт

| Документ | Описание |
|----------|----------|
| [api/openapi.yaml](api/openapi.yaml) | Актуальная спецификация OpenAPI |
| [api/authentication.md](api/authentication.md) | Сессии и API-токены |
| [import/cubux.md](import/cubux.md) | Импорт формата Cubux |

## Установка

| Документ | Описание |
|----------|----------|
| [install/manual.md](install/manual.md) | Ручная установка |
| [install/docker.md](install/docker.md) | Docker-установка |
| [install/nginx.md](install/nginx.md) | Reverse proxy и HTTPS |

