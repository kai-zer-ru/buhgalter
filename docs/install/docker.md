# Установка через Docker

Продакшен-сценарий: запуск через готовый образ из registry.

---

## Быстрый старт

```bash
docker pull ghcr.io/kai-zer-ru/buhgalter:latest
docker compose -f docker/docker-compose.yml up -d
```

Тег образа: переменная `BUHGALTER_IMAGE_TAG` (по умолчанию `latest`), например `BUHGALTER_IMAGE_TAG=1.0.0 docker compose ...`. В GitHub Release к compose прилагается `.env` с версией релиза.

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
| `/app/locales` | опционально: переопределение локалей (в образе уже есть `ru`/`en`) |

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

Пока **внешний URL** в админке не задан, приложение отвечает на `localhost` и адреса **локальной сети** (`192.168.x.x`, `10.x`, `172.16–31.x`). Порт в compose по умолчанию `127.0.0.1:8765:8765` — только с этого хоста; для доступа с других устройств в LAN замените на `8765:8765`. Для доступа из интернета настройте nginx, укажите `external_url` в `/admin`.

### Уведомления MAX (official API)

Образ содержит **Russian Trusted Root CA** и **Sub CA 2024** (цепочка для `platform-api2.max.ru`); файлы лежат в репозитории (`docker/certs/`), без скачивания с gu-st.ru при сборке. Обновление: `scripts/vendor_russian_ca_certs.sh`. Бинарник без Docker — [README](../../README.md#уведомления-max--сертификаты-минцифры-обязательно).

## Reverse proxy

Для HTTPS за nginx см. [nginx.md](nginx.md).
