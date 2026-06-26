# Установка через Docker

Продакшен-сценарий: запуск через готовый образ из registry.

---

## Быстрый старт

```bash
cd docker
cp .env.example .env
docker pull ghcr.io/kai-zer-ru/buhgalter:latest
docker compose up -d
```

Тег образа: `BUHGALTER_IMAGE_TAG` в `.env` (по умолчанию `latest`). В GitHub Release рядом с `docker-compose.yaml` лежат `.env` (с версией релиза) и `.env.example`.

Локальная сборка без pull:

```bash
# в docker/docker-compose.yml раскомментируйте build: и закомментируйте image:
docker compose -f docker/docker-compose.yml up --build -d
```

## Теги образа

| Тег | Назначение |
|-----|------------|
| `latest` | последний стабильный релиз (без pre-release) |
| `vX.Y.Z` | точная версия |
| `X.Y.Z`, `X.Y`, `X` | semver alias |

## Volumes

| Путь | Назначение |
|------|------------|
| `/app/data` | база SQLite и runtime-данные |
| `/app/backups` | архивы бэкапов |

## Обновление контейнера

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d
```

## Проверка

```bash
docker run --rm -p 8765:8765 -v buhgalter-data:/app/data ghcr.io/kai-zer-ru/buhgalter:latest
```

Откройте `http://localhost:8765/setup` и проверьте `GET /api/v1/health`.
На `/setup` можно сразу восстановить БД из `.db` (endpoint: `POST /api/v1/setup/restore`).

Пока **внешний URL** в разделе **Настройки → Админка** не задан, доступ разрешён с **localhost** (всегда) и с Host из `BUHGALTER_ALLOWED_HOSTS` в `.env`. Порт в compose по умолчанию `127.0.0.1:8765:8765` — только с этого хоста; для доступа с других устройств замените на `8765:8765` и добавьте их IP в `.env`. Для HTTPS из интернета — nginx и `external_url` в админ-настройках.

### Уведомления MAX (official API)

Образ содержит **Russian Trusted Root CA** и **Sub CA 2024** (цепочка для `platform-api2.max.ru`); файлы лежат в репозитории (`docker/certs/`), без скачивания с gu-st.ru при сборке. Обновление: `scripts/vendor_russian_ca_certs.sh`. Бинарник без Docker — [README](../../README.md#уведомления-max--сертификаты-минцифры-обязательно).

## Reverse proxy

Для HTTPS за nginx см. [nginx.md](nginx.md).
