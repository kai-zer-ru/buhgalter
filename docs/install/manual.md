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

## Обновление

```bash
git pull
make build
./buhgalter
```

Для dev-базы после пересборки миграций (одноразовый break перед v1) — пересоздать SQLite или выполнить `make clear`.

## Переменные окружения

Основные переменные: `BUHGALTER_ADDR`, `BUHGALTER_DB_PATH`, `BUHGALTER_DATA_DIR`, `BUHGALTER_LOG_DIR`, `BUHGALTER_STATIC_EMBED`.

Подробный список и примеры — в [README.md](../../README.md).
