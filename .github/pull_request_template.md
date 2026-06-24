## Описание

<!-- Что изменено и зачем -->

## Связанные issues

<!-- Closes #123 -->

## Область

<!-- Отметьте затронутые части; при merge workflow может добавить `area:*` labels по путям файлов -->

- [ ] API (`server/`)
- [ ] UI (`web/`)
- [ ] БД / миграции (`server/internal/db/migrations/`)
- [ ] Docker / CI (`.github/`, `docker/`)
- [ ] Документация (`README.md`, комментарии API, `*.md`)

## Чеклист

- [ ] `make ci` проходит локально (или релевантные `make test`, `make lint-go`, `make lint-web`)
- [ ] Миграции БД добавлены, если менялась схема (`server/internal/db/migrations/`)
- [ ] `make sqlc` / `make sqlc-check`, если менялись `server/queries/`
- [ ] OpenAPI обновлён, если менялся API (`docs/api/openapi.yaml`, затем `make copy-openapi`)
- [ ] i18n обновлён, если менялись строки UI (`web/src/lib/i18n/`, `server/locales/`)

## Скриншоты

<!-- Для UI-изменений -->
