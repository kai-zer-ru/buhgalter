# Правила доступа к SQL

Краткое руководство: где писать SQL в проекте buhgalter.

Полная модель данных: [data-model.md](data-model.md).

## Основное правило

**Любой запрос к таблице из production-кода** (handlers, services, workers, schedulers) — только через sqlc:

1. Добавить или изменить запрос в `server/queries/<table>.sql`.
2. Выполнить `make sqlc`.
3. Закоммитить `server/queries/` и сгенерированный `server/internal/db/sqlc/`.
4. Вызывать из Go через `sqlcdb.New(db).MethodName(...)`.

Файлы `server/internal/db/sqlc/*.sql.go` **не редактировать вручную** — это артефакт генерации.

Паттерн в пакете (как в `internal/account`):

```go
import sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"

func queries(db *sql.DB) *sqlcdb.Queries {
    return sqlcdb.New(db)
}
```

## Исключения (inline SQL в Go допустим)

| Место | Пример | Почему |
|-------|--------|--------|
| `*_test.go` | `UPDATE ... SET next_run_at = ?` в тесте | фикстуры, не production |
| `server/internal/db/manager.go` | `VACUUM INTO`, `PRAGMA wal_checkpoint` | операции SQLite, не доменные таблицы |
| `server/internal/accountbalance/hook.go` | `pragma_table_info(...)` | метаданные схемы SQLite |
| `admin/handler.go` | `SELECT version_id FROM goose_db_version` | инфраструктура миграций |

Новые исключения добавлять только с явным обоснованием в PR (и строкой в `FILE_EXCEPTIONS` в `scripts/check_inline_sql.py`).

## CI-проверка (фаза 4)

`make inline-sql-check` (`scripts/check_inline_sql.py`) сканирует `server/**/*.go`, кроме `*_test.go` и `internal/db/sqlc/`.  
Находит вызовы `.Exec` / `.Query` / `.QueryRow` с raw string, содержащим SQL-ключевые слова.  
Проверка входит в `make ci` и workflow `ci.yml`.

## Запрещено

- Писать `SELECT` / `INSERT` / `UPDATE` / `DELETE` к таблицам в `.go`-файлах production-кода (кроме исключений выше).
- Дублировать один запрос и в sqlc, и inline.
- Редактировать `server/internal/db/sqlc/*.sql.go` вручную.

## Правило на будущее

- **Новый запрос** к таблице → сразу в `server/queries/`.
- **Правка существующего inline** → перенести в sqlc в том же PR (или в ближайшем follow-up по той же таблице).

## Workflow при изменении схемы

1. Миграция goose → `server/internal/db/migrations/`.
2. Обновить `server/schema.sql`.
3. Добавить/изменить запросы в `server/queries/`.
4. `make sqlc` → закоммитить генерацию.
5. `make sqlc-check` — CI проверяет, что генерация актуальна.
6. `make inline-sql-check` — CI запрещает новый inline SQL в production-коде (см. [исключения](#исключения-inline-sql-в-go-допустим)).

## План миграции legacy inline SQL

Сейчас inline SQL в production остаётся только в [исключениях](sql-access.md#исключения-inline-sql-в-go-допустим) (`manager.go`, `hook.go`, `goose_db_version` в admin). Перенос завершён по фазам 0–3.

| Фаза | Файлы запросов | Пакеты | Статус |
|------|----------------|--------|--------|
| 1 | `users.sql`, `sessions.sql`, `api_tokens.sql`, `password_reset_requests.sql` | auth, admin, user, setup, categoryseed, notify | готово |
| 2 | `system_settings.sql` | admin, auth, settingscache, backup, notify, db, cmd, setup | готово |
| 3 | расширение `import.sql`, `recurring_operations.sql`, `credits.sql`, `accounts.sql`, `banks.sql` | importexport, recurring, credit, accountbalance, notify/worker, bank | готово |
| 4 | CI-grep против inline SQL (`scripts/check_inline_sql.py`) | `Makefile`, `.github/workflows/ci.yml` | готово |

## Критерии готовности (ROADMAP v1.3.0 «Навести порядок в SQL»)

| Критерий | Статус |
|----------|--------|
| Правило в `data-model.md` + `docs/sql-access.md` + локальный `sql-access.mdc` | готово |
| Фазы 1–3: все доменные таблицы в sqlc (см. таблицу фаз выше) | готово |
| Фаза 4: `make inline-sql-check` в CI | готово |
| `make sqlc-check`, `make inline-sql-check` и тесты зелёные | проверять перед коммитом |
| В production-коде нет inline SQL к таблицам (кроме [исключений](#исключения-inline-sql-в-go-допустим)) | готово |

Пункт ROADMAP закрыт после коммита фаз 0–4 и зелёного CI.
