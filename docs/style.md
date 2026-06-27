# Стиль документации

Единые правила оформления Markdown в публичной документации репозитория (`docs/`, `roadmap/`). На этапе релиза все файлы из [списка для ревизии](#охват-ревизии) приводятся к этому стилю.

## Язык и тон

- Основной язык — **русский** (кроме идентификаторов кода, путей, имён API).
- Короткие предложения; списки задач — глагол в инфинитиве («Добавить», «Проверить»).
- Термины проекта единообразно: **счёт**, **операция**, **категория**, **self-hosted**, API — `GET /api/v1/...`.

## Структура файла

### Общий документ (`docs/*.md`, `roadmap/*.md`)

```markdown
# Заголовок (один H1 на файл)

Краткое введение в 1–3 предложения.

---

## Раздел

### Подраздел
```

- После H1 — смысловой абзац, не сразу таблица.
- Крупные блоки разделять горизонтальной линией `---`.
- Глубина заголовков — не глубже `####` без необходимости.

### Индекс

| Файл | Роль |
|------|------|
| [README.md](README.md) | Индекс публичной документации |
| [ROADMAP.md](../ROADMAP.md) | Только список идей |
| [roadmap/](../roadmap/) | Детали идей после v1 |
| [style.md](style.md) | Единый стиль Markdown |
| [data-model.md](data-model.md) | ER-диаграмма, SQL, миграции |
| [categories-and-icons.md](categories-and-icons.md) | Категории, подкатегории, иконки |
| [ui-dialogs.md](ui-dialogs.md) | Диалоги, подтверждения, Esc |
| [ui-navigation.md](ui-navigation.md) | Навигация и кликабельные сущности |
| [ui-row-actions.md](ui-row-actions.md) | Меню «⋯» в строках и спойлер фильтров |
| [ui-stats.md](ui-stats.md) | Страница `/stats` |
| [ui-credits.md](ui-credits.md) | UI кредитов |
| [ui-stable-layout.md](ui-stable-layout.md) | Стабильная шапка и EmptyState |
| [ui-empty-states.md](ui-empty-states.md) | Пустые состояния |
| [ui-table-columns.md](ui-table-columns.md) | Порядок колонок таблиц |
| [transactions-display.md](transactions-display.md) | Отображение операций |
| [import/cubux.md](import/cubux.md) | Импорт формата Cubux |
| [api/openapi.yaml](api/openapi.yaml) | OpenAPI v1 |
| [api/authentication.md](api/authentication.md) | Авторизация, сессии, API-токены |
| [install/manual.md](install/manual.md) | Ручная установка |
| [install/docker.md](install/docker.md) | Docker-установка |
| [install/nginx.md](install/nginx.md) | Reverse proxy и HTTPS |

## Оформление

### Ссылки

- Относительные пути внутри `docs/`: `[ui-dialogs](ui-dialogs.md)`, `[OpenAPI](api/openapi.yaml)`.
- Якоря для разделов: `#ui--диалоги` (GitHub-style, строчные, дефисы).
- Внешние URL — полные `https://…`.

### Код

- Блоки с языком: ` ```bash `, ` ```sql `, ` ```go `, ` ```json `, ` ```yaml `, ` ```typescript `.
- Пути и команды — в `inline code`, не курсивом.
- SQL — ссылка на `server/internal/db/migrations/`; длинные снимки — в `<details>` только при необходимости.

### Таблицы

- Заголовок строки `| Колонка | … |`, разделитель `|---|`.
- В ячейках API: `` `GET /path` ``; статусы: `✅` / `[x]` / `[ ]`.

### Чеклисты

- Задачи: `- [ ]` / `- [x]`.
- Группы «Проверка», «Критерии приёмки» — в конце документа, где уместно.

### UI-документы (`docs/ui-*.md`)

- H1: `# UI — тема` (как [ui-dialogs.md](ui-dialogs.md)).
- Таблица «где применяется» / компоненты в начале.
- Попапы (модалки, выпадающие списки) — закрытие по **Esc**: [ui-dialogs.md](ui-dialogs.md#закрытие-по-escape).
- Перекрёстные ссылки между `ui-*.md`, без дублирования полных требований.

## Охват ревизии

**Корень репозитория:**

- [README.md](../README.md), [ROADMAP.md](../ROADMAP.md)

**`docs/`:**

- Все `*.md`, включая [import/](import/cubux.md)
- [api/openapi.yaml](api/openapi.yaml) — русские `description` (отдельный чеклист OpenAPI)

**`roadmap/`:**

- Все `*.md` — тот же стиль, что `docs/`; в шапке: «Не входит в v1»

**Вне охвата:** `web/README.md` (шаблон Svelte), `.github/*`, комментарии в коде.

## Проверка

- [ ] У каждого `.md` один H1, нет «висящих» заголовков без текста
- [ ] Битые относительные ссылки (`rg '\]\([^h]'` + ручной проход)
- [ ] README ссылается на актуальный набор `docs/`
- [ ] Дубли требований между `docs/` сведены к ссылкам
