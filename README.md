# Бухгалтер

Self-hosted учёт финансов: один бинарник с веб-интерфейсом, API и SQLite плюс отдельное Android-приложение к вашему серверу. Данные остаются у вас — на своей машине или VPS.

**Стек:** Go (API), SQLite, SvelteKit (веб и Android UI).

[![GitHub](https://img.shields.io/badge/GitHub-kai--zer--ru%2Fbuhgalter-blue?logo=github)](https://github.com/kai-zer-ru/buhgalter)
[![Поддержать](https://img.shields.io/badge/донат-Tinkoff-FFDD2D.svg)](https://www.tbank.ru/rm/r_wKLcbFgjYa.ncgWMwrHSA/vyQvd5941/)

---

## Сообщество и поддержка

Новости, обновления и помощь:

- **Telegram** — [@kai_zer_ru_ha](https://t.me/kai_zer_ru_ha)
- **Max** — [kai_zer_ru_ha](https://max.ru/id251603503331_biz)
- **Дзен** — [kai_zer_ru_ha](https://dzen.ru/kai_zer_ru_ha)
- **VK** — [kai_zer_ru_ha](https://vk.com/kai_zer_ru_ha)
- **Обсуждение** — [чат в Max](https://max.ru/join/KoCsTSA3VGOCiIFdSAW0myVJEwXZi-rt9fTfGxdgk6A)
- **Поддержка автора** — [Т-Банк](https://www.tbank.ru/rm/r_wKLcbFgjYa.ncgWMwrHSA/vyQvd5941/)

---

## Содержание

- [О проекте](#о-проекте)
- [Демо](#демо)
- [Android-клиент](#android-клиент)
- [Установка](#установка)
- [Nginx (reverse proxy + HTTPS)](#nginx-reverse-proxy--https)
- [Первый запуск (/setup)](#первый-запуск-setup)
- [Переменные окружения](#переменные-окружения)
- [Уведомления MAX — сертификаты Минцифры](#уведомления-max--сертификаты-минцифры)
- [Бэкапы](#бэкапы)
- [Обновление](#обновление)
- [Отладочные логи](#отладочные-логи)
- [Документация](#документация)
- [API-документация](#api-документация)
- [Разработка](#разработка)
- [Лицензия](#лицензия)

## О проекте

Бухгалтер помогает вести счета, операции, долги и кредиты, смотреть статистику и бюджет, получать уведомления в Telegram или MAX. Подходит для одного пользователя или небольшой семьи на собственном хостинге.

**Основные возможности:**

- **Счета и операции** — наличные, банковские и кредитные карты; группы «Мои средства» и «Кредитные средства»; архивация и мягкое удаление с переводом остатка; автопополнение банковского счёта по порогу. Категории с иконками; доходы, расходы и переводы (в том числе с комиссией). Плановые операции, повтор из меню строки, периодические списания.
- **Долги** — дать или взять в долг; при погашении можно учитывать или не учитывать движение по счёту.
- **Кредиты** — потребительские и ипотека: график платежей, опциональный учёт суммы займа в балансе, правка будущих сумм, оплата вручную и автосписание по расписанию.
- **Бюджет** — помесячные лимиты по категориям, план vs факт, подсказка «уже потрачено» при создании лимита, уведомления при пересечении порога.
- **Статистика** — сводка, разбивка по периодам и категориям (подкатегории под спойлером, колонки бюджета), поиск по операциям.
- **Импорт и экспорт** — формат Cubux (CSV/XLSX).
- **Уведомления** — Telegram и MAX: напоминания о долгах, кредитах, плановых операциях и бюджете; предупреждение о нехватке средств; ссылки на разделы в тексте. Для official API MAX нужны [сертификаты Минцифры](#уведомления-max--сертификаты-минцифры).
- **Админка** — пользователи (модерация регистрации, блокировка), сброс пароля, бэкапы, диагностика, внешний URL для reverse proxy.
- **Android-клиент** — приложение к вашему инстансу: вход по логину/паролю или API-токену, офлайн-очередь, виджеты, share-intent, поиск сервера в LAN (mDNS), PIN и биометрия. APK — в [GitHub Releases](https://github.com/kai-zer-ru/buhgalter/releases). Подробнее — [Android-клиент](#android-клиент) и [docs/android-client.md](docs/android-client.md).
- **Интерфейс** — светлая / тёмная / «Как на устройстве»; веб и Android со stale-while-revalidate кешем справочников.

Детали интерфейса, API и модели данных — в [документации](docs/README.md).

## Демо

Попробовать без установки: **[buhgalter-demo.kai-zer.ru](https://buhgalter-demo.kai-zer.ru/)**

| Логин | Пароль      |
| ----- | ----------- |
| demo  | demo_1_demo |

На демо-стенде в админке задан **внешний URL** (тот же домен). При подключении Android-приложения по LAN домен из настроек показывается в списке найденных серверов рядом с локальным адресом.

## Android-клиент

Мобильное приложение к вашему self-hosted инстансу. Исходники — каталог `android/` (UI в `android/ui/`; веб `web/` в APK не входит). Требуется **Android 8.0+**.

**Распространение:** APK из [GitHub Releases](https://github.com/kai-zer-ru/buhgalter/releases) (universal и per-ABI). Google Play не планируется.

**Первый запуск:**

1. **Адрес сервера** — поиск в Wi‑Fi (mDNS `_buhgalter._tcp` + скан подсети на порту API) или ручной ввод **LAN URL**; проверка `GET /api/v1/health`.
2. **Вход** — логин и пароль **или** API-токен из веб-интерфейса (Настройки → API-токены).

Позже в **Настройки → Сервер**: внешний URL (для мобильной сети), до 5 домашних Wi‑Fi (SSID), fallback LAN→внешний, доверие к self-signed HTTPS, отладочные логи.

**Основное:**

- Счета, операции, переводы, долги, кредиты (пошаговое создание), бюджет, статистика, настройки, админка (без управления пользователями).
- Офлайн: очередь изменений (операции, переводы, категории, долги, счета, бюджет) с replay при появлении сети; кеш справочников.
- Виджеты: быстрые действия, баланс, бюджет, ближайшие платежи, один счёт.
- «Поделиться» из других приложений открывает форму расхода; ярлыки лаунчера — расход / доход / перевод.
- Блокировка PIN + биометрия; при выходе, отключении сервера или истечении сессии блокировка сбрасывается.
- Тема: светлая / тёмная / «Как на устройстве»; status bar следует теме; тематическая иконка лаунчера (Material You).
- При сбое загрузки экрана — сообщение и «Повторить».
- Если APK отстаёт от сервера — при необходимости подтягиваются строки интерфейса (`GET /api/v1/ui/i18n/{lang}`).

**Сборка:** `make android-sync`, `make android-apk`, `make android-install`. Документация: [docs/android-client.md](docs/android-client.md), [android/README.md](android/README.md).

Сервер для discovery публикует mDNS (`BUHGALTER_MDNS_ENABLED`, по умолчанию `true`). Для доступа по LAN-IP добавьте хост в `BUHGALTER_ALLOWED_HOSTS` или настройте внешний URL в админке.

## Установка

Образ Docker: `ghcr.io/kai-zer-ru/buhgalter` (тег `latest`). Порт приложения — **8765**.

После любого способа ниже откройте [http://localhost:8765/setup](http://localhost:8765/setup) и пройдите [первый запуск](#первый-запуск-setup).

**Каталоги по умолчанию:**

| Каталог         | Содержимое                             |
| --------------- | -------------------------------------- |
| `data/`         | SQLite, маркер установки `.configured` |
| `data/backups/` | файлы бэкапов БД                       |
| `logs/`         | логи приложения                        |

Настройки (порт, allowed hosts) — через `.env`, см. [переменные окружения](#переменные-окружения). Подробнее: [docs/install/docker.md](docs/install/docker.md), [docs/install/manual.md](docs/install/manual.md).

Пока **внешний URL** в админке не задан, доступны **localhost** (всегда) и Host из `BUHGALTER_ALLOWED_HOSTS`.

### docker run

Один контейнер без compose — удобно для быстрой проверки:

```bash
docker pull ghcr.io/kai-zer-ru/buhgalter:latest

docker run -d --name buhgalter \
  -p 8765:8765 \
  -v buhgalter-data:/app/data \
  -v buhgalter-logs:/app/logs \
  --env-file .env \
  --restart unless-stopped \
  ghcr.io/kai-zer-ru/buhgalter:latest
```

`-p 8765:8765` — доступ с любого интерфейса хоста. Только с этой машины: `-p 127.0.0.1:8765:8765`.

Локальная сборка образа: `make docker-build` (тег `buhgalter:local`), подставьте имя образа в `docker run`.

### docker compose

Рекомендуемый способ для постоянной установки — [docker/docker-compose.yml](docker/docker-compose.yml):

```bash
cd docker
cp .env.example .env
docker pull ghcr.io/kai-zer-ru/buhgalter:latest
docker compose up -d
```

Данные на хосте: `./data`, `./logs`; бэкапы — в `./data/backups/`.

Порт в compose по умолчанию: `127.0.0.1:8765:8765`. Для доступа с других устройств в LAN замените на `"8765:8765"` и добавьте IP в `BUHGALTER_ALLOWED_HOSTS`.

Локальная сборка — в `docker-compose.yml` раскомментируйте `build:` и закомментируйте `image:`:

```yaml
build:
  context: ..
  dockerfile: docker/Dockerfile
```

```bash
docker compose up --build -d
```

### Бинарник с GitHub Releases

На странице [Releases](https://github.com/kai-zer-ru/buhgalter/releases) скачайте архив под свою ОС (Linux, Windows, macOS), распакуйте и запустите:

```bash
tar -xzf buhgalter_*_linux_amd64.tar.gz
./buhgalter
```

Проверьте контрольную сумму из `checksums.txt` в том же релизе.

Запускайте из каталога, где создадутся `data/` и `logs/`, или укажите пути в `.env`.

### Сборка из исходников

Нужны **Go 1.26+**, **Node.js 22+** (сборка фронтенда) и `make`.

```bash
git clone https://github.com/kai-zer-ru/buhgalter.git
cd buhgalter
make build
./buhgalter
```

Бинарник также попадает в `bin/buhgalter`. Для Raspberry Pi / linux arm64: `make build-arm` → `bin/buhgalter-linux-arm64`. Подробнее — [docs/install/manual.md](docs/install/manual.md).

Если `make build` падает с `EACCES` на `web/build` — `make fix-build-perms`.

## Nginx (reverse proxy + HTTPS)

Приложение слушает HTTP на `:8765` без встроенного TLS. Для HTTPS поставьте nginx (или другой reverse proxy) перед бинарником или контейнером.

Готовый пример: [docker/nginx.conf.example](docker/nginx.conf.example).

```nginx
server {
    server_name buhgalter.my-site.ru;

    location / {
        proxy_pass http://127.0.0.1:8765;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    listen 443 ssl;
    ssl_certificate /etc/ssl/fullchain.pem;
    ssl_certificate_key /etc/ssl/privkey.pem;
}
```

В **Настройки → Админка** укажите **внешний URL** — например `https://buhgalter.my-site.ru` — для ссылок в уведомлениях и доступа из интернета.

Подробнее: [docs/install/nginx.md](docs/install/nginx.md).

## Первый запуск (/setup)

При первом запуске откроется `/setup`:

1. Имя и логин администратора
2. Пароль (дважды): минимум 8 символов, хотя бы одна буква и одна цифра, не совпадает с логином
3. При необходимости — восстановление БД из `.db` бэкапа до завершения настройки

Факт завершения установки хранится в `data/.configured` (вне SQLite). Восстановление бэкапа **не** сбрасывает этот маркер.

Если приложение за reverse proxy — см. [Nginx](#nginx-reverse-proxy--https) и поле **внешний URL** в админке.

## Переменные окружения

Файл `.env` читается при старте (путь — `BUHGALTER_ENV_FILE`, по умолчанию `.env` в **текущем рабочем каталоге**). Приложение **не изменяет** `.env`. Уже заданные в shell переменные не перезаписываются.

**Где разместить `.env`:**

- **Бинарник** — рядом с `./buhgalter` (каталог запуска). Или укажите путь через `BUHGALTER_ENV_FILE`.
- **Docker Compose** — `cp docker/.env.example docker/.env` и отредактируйте.

**Пример для доступа по IP/домену без reverse proxy:**

```env
BUHGALTER_ALLOWED_HOSTS=["192.168.1.100","example.com"]
```

| Переменная                | По умолчанию                      | Описание |
| ------------------------- | --------------------------------- | -------- |
| `BUHGALTER_ADDR`          | `:8765`                           | Адрес и порт HTTP-сервера |
| `BUHGALTER_DB_PATH`       | `./data/buhgalter.db`             | Путь к файлу SQLite |
| `BUHGALTER_DATA_DIR`      | `./data`                          | Каталог данных (БД, `.configured`, `backups/`) |
| `BUHGALTER_LOG_DIR`       | `./logs`                          | Каталог логов |
| `BUHGALTER_LOG_MODE`      | `prod`                            | `prod` — редактирование чувствительных заголовков; `dev` — полные request-логи |
| `BUHGALTER_CORS_ORIGINS`  | `*`                               | CORS: `*` отражает Origin запроса (нужно для cookie-сессий) |
| `BUHGALTER_ENV_FILE`      | `.env`                            | Путь к файлу `.env` |
| `BUHGALTER_ALLOWED_HOSTS` | `["127.0.0.1","localhost","::1"]` | Host для прямого доступа (JSON-массив). localhost всегда разрешён |
| `BUHGALTER_STATIC_EMBED`  | `true`                            | Встроенный фронтенд (`false` — отдельный Vite dev) |
| `BUHGALTER_MDNS_ENABLED`  | `true`                            | Публикация `_buhgalter._tcp` в LAN для Android discovery |
| `BUHGALTER_MDNS_NAME`     | `Buhgalter`                       | Имя экземпляра в mDNS |

Миграции БД применяются автоматически при старте (goose).

## Уведомления MAX — сертификаты Минцифры

Официальный API MAX (`platform-api2.max.ru`) использует TLS-сертификаты **НУЦ Минцифры**. Без доверия к ним отправка через провайдер **official** завершится ошибкой TLS.

**Важно:** нужен **Sub CA 2024** (`russian_trusted_sub_ca_2024_pem.crt`), а не старый выпуск 2022 года.

Скачайте архивы (Linux):

```bash
mkdir -p ~/certs && cd ~/certs
curl -fsSLO https://gu-st.ru/content/lending/linux_russian_trusted_root_ca_pem.zip
curl -fsSLO https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.zip
unzip -o linux_russian_trusted_root_ca_pem.zip
unzip -o russian_trusted_sub_ca_pem.zip
```

Портал для всех ОС: **[gosuslugi.ru/crt](https://www.gosuslugi.ru/crt)**.

### Образ GHCR

В образе `ghcr.io/kai-zer-ru/buhgalter` сертификаты уже добавлены. **Дополнительных действий не требуется.**

### Linux

**Debian, Ubuntu:**

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt russian_trusted_sub_ca_2024_pem.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

**Fedora, RHEL, AlmaLinux, Rocky:**

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt russian_trusted_sub_ca_2024_pem.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

**Arch Linux:**

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt /etc/ca-certificates/trust-source/anchors/russian_trusted_root_ca.crt
sudo cp russian_trusted_sub_ca_2024_pem.crt /etc/ca-certificates/trust-source/anchors/russian_trusted_sub_ca_2024.crt
sudo update-ca-trust
```

Проверка (успех — любой HTTP-код **без** `curl: (60) SSL certificate...`):

```bash
curl -v https://platform-api2.max.ru/
```

### Windows

1. Скачайте сертификаты с [gosuslugi.ru/crt](https://www.gosuslugi.ru/crt).
2. Установите Root CA и Sub CA 2024 в **Доверенные корневые центры сертификации** (локальный компьютер).
3. Перезапустите `buhgalter.exe`.

### macOS

Импортируйте оба сертификата в **Системную** связку ключей → **Доверие** → **Всегда доверять**. Перезапустите бинарник.

### Секретный ключ уведомлений

Токены Telegram и MAX шифруются AES-256-GCM. Ключ задаётся в **Настройки → Админка** — ровно **32 символа**.

## Бэкапы

В интерфейсе: **Настройки → Админка → Бэкапы**.

- Ручное создание и скачивание копии базы
- Автобэкап по расписанию
- Восстановление из файла `.db` (подтверждение `RESTORE`)

Файлы лежат в `{каталог данных}/backups/` — при локальном запуске это `data/backups/`, в Docker — `./data/backups/` на хосте.

## Обновление

Сделайте бэкап, замените бинарник или образ, перезапустите. Миграции применятся при старте.

**Docker:**

```bash
cd docker
docker compose pull
docker compose up -d
```

## Отладочные логи

### Сервер (Go)

Каталог логов — `BUHGALTER_LOG_DIR` (по умолчанию `./logs`). Режим — `BUHGALTER_LOG_MODE`:

| Режим | Поведение |
| ----- | --------- |
| `prod` (по умолчанию) | Безопасные request-логи: чувствительные заголовки (`Authorization`, cookie и т.п.) редактируются |
| `dev` | Полные request-логи и расширенная диагностика при ошибках API |

Для отладки доступа с телефона или reverse proxy временно можно поставить `BUHGALTER_LOG_MODE=dev` в `.env` и перезапустить сервер. Audit-лог действий пользователей — отдельно в `logs/audit/`.

### Android-клиент

**Настройки → Сервер → «Включить логирование».** Пока переключатель включён, в `localStorage` накапливается журнал (до 3000 событий):

- HTTP-запросы к API (метод, путь, статус, время; **токены и пароли не пишутся**)
- Синхронизация outbox и прогрев ref-cache
- SWR-кеш (hit / revalidate / offline miss)
- Смена доступности сервера
- Навигация и bootstrap
- Необработанные ошибки и `unhandledrejection`

При **выключении** логирования — диалог: сохранить файл `buhgalter-debug-*.log` в папку **Загрузки** на устройстве или пропустить. В начале файла — снимок окружения (версия приложения, URL сервера, профиль, outbox, reachability).

Для воспроизведения багов: включить логирование → повторить сценарий → выключить и сохранить файл.

Подробнее: [docs/android-client-platform.md](docs/android-client-platform.md#отладочное-логирование).

## Документация

Справочники по установке, данным, UI и API — [docs/README.md](docs/README.md). История изменений — [CHANGELOG.md](CHANGELOG.md).

## API-документация

OpenAPI доступна без авторизации:

- **Демо:** [buhgalter-demo.kai-zer.ru/docs](https://buhgalter-demo.kai-zer.ru/docs)
- **После запуска:** [http://localhost:8765/docs](http://localhost:8765/docs)
- **Исходник:** [docs/api/openapi.yaml](docs/api/openapi.yaml)

В режиме разработки (`make dev-server` + `make dev-web`) — также [http://localhost:5173/docs](http://localhost:5173/docs) (прокси на API).

## Разработка

```bash
make dev-server   # API без встроенного фронта
make dev-web      # фронтенд на http://localhost:5173
```

```bash
make test         # go test + web/android check + android vitest + web e2e
make test-unit    # без e2e
make test-e2e-web
make lint-go
make ci           # lint + тесты + build + docker-build
make clear        # очистка БД, логов, бэкапов и артефактов сборки
```

| Команда                           | Описание                              |
| --------------------------------- | ------------------------------------- |
| `make download-bank-logos`        | Логотипы банков                       |
| `make download-marketplace-logos` | Логотипы маркетплейсов для категорий  |
| `make generate-category-icons`    | SVG иконок категорий                  |
| `make sqlc`                       | Регенерация Go из `server/queries/`   |
| `make build-arm`                  | linux/arm64 бинарник (Raspberry Pi)   |
| `make android-sync`               | Сборка Android UI + `cap sync`        |
| `make android-apk`                | Debug APK                             |
| `make android-apk-release`        | Release APK (universal + per-ABI)     |
| `make android-install`            | Сборка debug и установка через adb    |
| `make android-install-release`    | Сборка release и установка universal APK через adb |

## Лицензия

[MIT License](LICENSE) — Copyright (c) 2026 kai-zer-ru
