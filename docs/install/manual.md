# Установка вручную

Пошаговая установка без Docker: сборка бинарника и запуск на хосте.

---

## Требования

- Linux/macOS
- Go и Node.js (если собираете web-часть локально)
- Права на запись в каталог проекта (`data/`, `logs/`)

## Установка

```bash
git clone https://github.com/kai-zer-ru/buhgalter
cd buhgalter
make build
./buhgalter
```

После старта открыть `http://localhost:8765` и пройти `/setup`.
Если есть резервная копия, её можно восстановить прямо на `/setup` из файла `.db` до завершения первичной настройки.

## Обновление

```bash
git pull
make build
./buhgalter
```

Для dev-базы после пересборки миграций (одноразовый break перед v1) — пересоздать SQLite или выполнить `make clear`.

## Переменные окружения

Приложение при старте **только читает** `.env` (путь — `BUHGALTER_ENV_FILE`, по умолчанию `.env` в каталоге запуска). Файл создаёт и редактирует пользователь (`docker/.env.example` → `docker/.env`).

Основные переменные: `BUHGALTER_ADDR`, `BUHGALTER_DB_PATH`, `BUHGALTER_DATA_DIR`, `BUHGALTER_LOG_DIR`, `BUHGALTER_ALLOWED_HOSTS`, `BUHGALTER_CORS_ORIGINS`.
Локали `ru/en` встроены в бинарник, отдельная настройка `BUHGALTER_LOCALES_DIR` для типового запуска не нужна.

`BUHGALTER_STATIC_EMBED` обычно не нужен в релизном запуске (по умолчанию `true` и фронтенд отдаётся из бинарника). Используется в dev-сценариях с отдельным Vite (`make dev-server` + `make dev-web`).

Подробный список и примеры — в [README.md](../../README.md).
