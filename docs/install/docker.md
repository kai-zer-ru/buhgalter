# Установка через Docker

Запуск готового образа из GHCR. Краткий обзор — в [README.md](../../README.md#docker).

---

## Быстрый старт

```bash
cd docker
cp .env.example .env
docker pull ghcr.io/kai-zer-ru/buhgalter:latest
docker compose up -d
```

Откройте [http://localhost:8765/setup](http://localhost:8765/setup).

Тег образа: `BUHGALTER_IMAGE_TAG` в `.env` (по умолчанию `latest`).

Локальная сборка — в `docker-compose.yml` раскомментируйте `build:` и закомментируйте `image:`:

```bash
docker compose up --build -d
```

## Теги образа

| Тег      | Назначение                              |
| -------- | --------------------------------------- |
| `latest` | последний стабильный релиз              |
| `vX.Y.Z` | точная версия                           |
| `X.Y.Z`  | semver alias                            |

Платформы: **linux/amd64**, **linux/arm64** (multi-arch manifest). С v1.2.4 в CI отключены provenance attestation manifests — в GHCR снова корректно отображаются архитектуры (раньше могло показываться `unknown/unknown`).

## Данные на хосте

| На хосте (по умолчанию) | В контейнере | Назначение |
| ----------------------- | ------------ | ---------- |
| `./data`                | `/app/data`  | БД, маркер `.configured`, **бэкапы** (`data/backups/`) |
| `./logs`                | `/app/logs`  | логи       |

Пути настраиваются через `BUHGALTER_HOST_DATA_DIR` и `BUHGALTER_HOST_LOGS_DIR` в `.env`.

При первом `docker compose up` Docker создаёт каталоги на хосте. Entrypoint выставляет владельца **uid 1000** (`buhgalter`).

> **Бэкапы:** приложение сохраняет их в `/app/data/backups` — на хосте это `./data/backups/`. Отдельный mount `./backups` в compose зарезервирован в образе, но приложение туда не пишет.

Порт по умолчанию в compose: `127.0.0.1:8765:8765` (только localhost). Для LAN — `"8765:8765"` и `BUHGALTER_ALLOWED_HOSTS` в `.env`.

## Обновление

```bash
cd docker
docker compose pull
docker compose up -d
```

Перед обновлением сделайте бэкап (`./data/backups/` или через админку).

## Переход с named volumes

Если раньше использовались тома `buhgalter-data` / `buhgalter-backups`, а теперь bind mounts:

```bash
docker compose down
mkdir -p data logs
docker run --rm -v buhgalter-data:/from -v "$(pwd)/data:/to" alpine sh -c 'cp -a /from/. /to/'
docker compose up -d
```

Проверьте, что в `./data` есть `buhgalter.db` и данные на месте.

## Уведомления MAX (official API)

В образе уже установлены сертификаты НУЦ Минцифры для `platform-api2.max.ru` (`docker/certs/`). Бинарник без Docker — [README](../../README.md#уведомления-max--сертификаты-минцифры).

## Reverse proxy

Для HTTPS — [nginx.md](nginx.md).
