# Установка вручную

Сборка бинарника и запуск на хосте без Docker.

---

## Требования

- Linux, macOS или Windows
- **Go 1.26+** и **Node.js 22+** — только если собираете из исходников
- Права на запись в каталог запуска (`data/`, `logs/`)

## Установка

```bash
git clone https://github.com/kai-zer-ru/buhgalter.git
cd buhgalter
make build
./buhgalter
```

Откройте [http://localhost:8765/setup](http://localhost:8765/setup) и создайте учётную запись администратора.

При наличии резервной копии её можно восстановить на `/setup` из файла `.db` **до** завершения первичной настройки.

### Raspberry Pi / linux arm64 (aarch64)

На x86/amd64-хосте можно кросс-собрать бинарник для Pi 3/4/5 и других ARM64:

```bash
make build-arm
# → bin/buhgalter-linux-arm64
```

Скопируйте файл на устройство и запускайте там. CGO не требуется (`modernc.org/sqlite`).

## Каталоги данных

| Путь            | Назначение                          |
| --------------- | ----------------------------------- |
| `data/`         | SQLite (`buhgalter.db`), `.configured` |
| `data/backups/` | файлы бэкапов                       |
| `logs/`         | логи                                |

## Обновление

```bash
git pull
make build
# остановите старый процесс, запустите ./buhgalter снова
```

Перед обновлением сделайте бэкап в **Настройки → Админка → Бэкапы** или скопируйте `data/buhgalter.db`.

## Переменные окружения

Создайте `.env` в каталоге запуска (шаблон для Docker: [docker/.env.example](../../docker/.env.example) — пути внутри контейнера замените на локальные).

Основные переменные: `BUHGALTER_ADDR`, `BUHGALTER_DATA_DIR`, `BUHGALTER_LOG_DIR`, `BUHGALTER_ALLOWED_HOSTS`.

Полный список и примеры — в [README.md](../../README.md#переменные-окружения).

`BUHGALTER_STATIC_EMBED=false` нужен только при разработке с отдельным Vite (`make dev-server` + `make dev-web`).

## Разработка

См. раздел [Разработка](../../README.md#разработка) в корневом README.
