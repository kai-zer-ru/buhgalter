# PostgreSQL

Не входит в v1. Черновик для обсуждения.

## Зачем

v1 — **SQLite** (один файл, простой self-hosted, `VACUUM INTO` для бэкапов). PostgreSQL имеет смысл, если:

- одна БД на много пользователей / [командную работу](team-collaboration.md)
- нужны конкурентные записи без блокировок файла
- деплой в Kubernetes / managed DB
- репликация и point-in-time recovery из коробки

Для типичного «один пользователь на Raspberry Pi» SQLite остаётся достаточным.

## Подход

**Опциональный драйвер**, не замена по умолчанию:

```
BUHGALTER_DB_DRIVER=sqlite|postgres
BUHGALTER_DATABASE_URL=postgres://...
```

- Миграции goose — один набор SQL с учётом отличий (`?` → `$1` через goose dialect или дублирование минимум)
- sqlc — отдельный `sqlc.yaml` или `engine: postgresql` при переключении
- Типы: `INTEGER` денег, `TEXT` UUID — совместимы; проверить `datetime` → `TIMESTAMPTZ`

## Отличия от SQLite (проверить при реализации)

| Область | SQLite сейчас | PostgreSQL |
|---------|---------------|------------|
| Бэкап | файл + VACUUM INTO | `pg_dump`, не upload .db в админке как сейчас |
| FTS | опционально в stage_7 | `tsvector` / другой синтаксис |
| `CHECK` / FK | есть | то же |
| Одновременность | writer lock | нормальная |

## Бэкапы админки

При Postgres — другой путь restore: не замена файла `.db`, а `pg_restore` / SQL dump. UI админки ([stage_09](../stage_09_release.md)) нужно ветвить по драйверу.

## Этапы

1. Абстракция `database/sql` уже есть; вынести DSN и dialect в config
2. Прогнать миграции и интеграционные тесты на Postgres в CI (service container)
3. Документация `docs/install/postgres.md`
4. Docker compose profile `postgres`

## Открытые вопросы

- [ ] Один бинарник с обоими драйверами или build tags?
- [ ] Миграция данных sqlite → postgres утилитой?
- [ ] Поддерживать только одну БД на инстанс (как сейчас)?
