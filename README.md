

# Бухгалтер

Self-hosted приложение для личного учёта финансов: один бинарник с веб-интерфейсом, API и SQLite. Данные хранятся у вас — на своём сервере или домашней машине.

**Стек:** Go (API), SQLite, SvelteKit (UI).

[![GitHub](https://img.shields.io/badge/GitHub-kai--zer--ru%2Fbuhgalter-blue?logo=github)](https://github.com/kai-zer-ru/buhgalter)
[![Поддержать](https://img.shields.io/badge/донат-Tinkoff-FFDD2D.svg)](https://www.tbank.ru/rm/r_wKLcbFgjYa.ncgWMwrHSA/vyQvd5941/)


---

## Сообщество и поддержка

Новости, обновления и помощь по теме:

- **Telegram** — [@kai_zer_ru_ha](https://t.me/kai_zer_ru_ha)
- **Max** — [kai_zer_ru_ha](https://max.ru/id251603503331_biz)
- **Дзен** — [kai_zer_ru_ha](https://dzen.ru/kai_zer_ru_ha)
- **VK** — [kai_zer_ru_ha](https://vk.com/kai_zer_ru_ha)
- **Обсуждение** - [Чат в Max](https://max.ru/join/KoCsTSA3VGOCiIFdSAW0myVJEwXZi-rt9fTfGxdgk6A)
- **Поддержка автора** - [Т-Банк](https://www.tbank.ru/rm/r_wKLcbFgjYa.ncgWMwrHSA/vyQvd5941/)

---

## Содержание

- [О проекте](#о-проекте)
- [Демо](#демо)
- [Сборка из исходников](#сборка-из-исходников)
- [Docker](#docker)
  - [docker run](#docker-run)
  - [docker compose](#docker-compose)
- [Nginx (reverse proxy + HTTPS)](#nginx-reverse-proxy--https)
- [Бинарник с GitHub Releases](#бинарник-с-github-releases)
- [Первый запуск (/setup)](#первый-запуск-setup)
- [Переменные окружения](#переменные-окружения)
- [Уведомления MAX — сертификаты Минцифры](#уведомления-max--сертификаты-минцифры-обязательно)
  - [Образ GHCR](#образ-ghcr)
  - [Linux](#linux)
  - [Windows](#windows)
  - [macOS](#macos)
  - [Секретный ключ уведомлений](#секретный-ключ-уведомлений)
- [Бэкапы](#бэкапы)
- [Обновление](#обновление)
- [Документация](#документация)
- [API-документация](#api-документация)
- [Разработка](#разработка)
- [Лицензия](#лицензия)

## О проекте

Бухгалтер помогает вести счета, операции, долги и кредиты, смотреть статистику и получать уведомления в Telegram или MAX. Подходит для одного пользователя или небольшой семьи на собственном хостинге.

**Основные возможности:**

- Счета (наличные и банковские), категории с иконками, операции и переводы **с комиссией**
- Двойной баланс по счетам: текущий и прогноз на текущий месяц (с учётом операций `future`)
- Долги и кредиты с графиками платежей (редактирование сумм будущих платежей, учёт задним числом, списание ретро-платежей со счёта, выбор банка кредита)
- Поддержка ипотеки (MVP): `property_price`, `down_payment`, автоматический расчёт суммы кредита (`price - down payment`), опция «не списывать первоначальный взнос с баланса», отдельный счёт списания взноса
- Автоматическая оплата кредита по локальному времени (`debit_time_local`): списание выполняется по дате/времени через планировщик, без массового предсоздания будущих транзакций; включается переключателем (по умолчанию `00:00`), отключается установкой `debit_time_local = null`
- Для ипотек в интерфейсе используется только ежемесячная периодичность платежа (`month`)
- Денежные суммы в кредитах отображаются в едином формате: разделители тысяч + знак валюты
- Для системных категорий (кредиты/долги/служебные) запрещены плановые операции `future` — действует backend-валидация
- Периодические операции: неделя/2 недели/месяц/год, отдельная страница управления, создание на основе существующей операции, inline-редактирование в списке и форма добавления по кнопке-спойлеру
- Длинные графики платежей отображаются компактно: сначала первые 10 строк с кнопкой раскрытия полного списка
- Списки операций и кредитов отображаются в порядке от новых к старым
- Импорт и экспорт (Cubux CSV/XLSX)
- Статистика и поиск по операциям
- Уведомления (Telegram, MAX) с настраиваемыми шаблонами — для MAX official обязательны [сертификаты Минцифры](#уведомления-max--сертификаты-минцифры-обязательно)
- Раздел администрирования в **Настройки → Админка**: пользователи, **сброс пароля**, уведомления админам о запросах сброса, бэкапы, диагностика, секретный ключ шифрования токенов уведомлений
- Адаптивный интерфейс для мобильных браузеров
- REST API и интерактивная документация OpenAPI

## Демо

Попробовать без установки: **[buhgalter-demo.kai-zer.ru](https://buhgalter-demo.kai-zer.ru/)**

| Логин | Пароль        |
| ----- | ------------- |
| demo  | demo_1_demo   |

## Сборка из исходников

**Требования:** Go 1.26+, Node.js 22+ (только для сборки фронтенда), `make`.

```bash
git clone https://github.com/kai-zer-ru/buhgalter.git
cd buhgalter
make build
./buhgalter
```

Бинарник также попадает в `bin/buhgalter`. Откройте [http://localhost:8765](http://localhost:8765) и пройдите [первый запуск](#первый-запуск-setup).

Каталоги по умолчанию: `data/` (база и маркер установки), `data/backups/` (бэкапы), `logs/`.

Для настройки (порт, allowed hosts и т.д.) создайте `.env` в каталоге запуска — см. [Переменные окружения](#переменные-окружения).

## Docker

Образ публикуется в GHCR после релиза: `ghcr.io/kai-zer-ru/buhgalter` (теги `latest`, `vX.Y.Z`). Подробнее — [docs/install/docker.md](docs/install/docker.md).

Данные: том или volume на `/app/data` (БД, `.configured`), бэкапы — `/app/backups`. Порт приложения: **8765**.

Пока **внешний URL** в админке не задан, доступны **localhost** (всегда) и Host из `BUHGALTER_ALLOWED_HOSTS` в `.env` (см. [Переменные окружения](#переменные-окружения)).

### docker run

Один контейнер без compose — удобно для быстрой проверки или своих скриптов:

```bash
docker pull ghcr.io/kai-zer-ru/buhgalter:latest

docker run -d --name buhgalter \
  -p 8765:8765 \
  -v buhgalter-data:/app/data \
  -v buhgalter-backups:/app/backups \
  --env-file .env \
  --restart unless-stopped \
  ghcr.io/kai-zer-ru/buhgalter:latest
```

`-p 8765:8765` — доступ с любого интерфейса хоста (в т.ч. `http://192.168.x.x:8765` в LAN). Только с этой машины: `-p 127.0.0.1:8765:8765`.

Локальная сборка образа: `make docker-build` (тег `buhgalter:local`), затем подставьте имя образа в `docker run`.

Откройте [http://localhost:8765/setup](http://localhost:8765/setup) и пройдите [первый запуск](#первый-запуск-setup).

### docker compose

Рекомендуемый способ для постоянной установки — `[docker/docker-compose.yml](docker/docker-compose.yml)`:

```bash
docker pull ghcr.io/kai-zer-ru/buhgalter:latest
docker compose -f docker/docker-compose.yml up -d
```

Тома по умолчанию: `buhgalter-data`, `buhgalter-backups`. Порт в compose: `127.0.0.1:8765:8765` (только localhost на хосте). Для доступа с других устройств в LAN замените в compose на `"8765:8765"`.

**Локальная сборка** — в compose закомментируйте `image:` и раскомментируйте `build:`:

```yaml
build:
  context: ..
  dockerfile: docker/Dockerfile
```

```bash
docker compose -f docker/docker-compose.yml up --build -d
```

**Обновление:**

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d
```

## Nginx (reverse proxy + HTTPS)

Приложение слушает HTTP на `:8765` без встроенного TLS. Для доступа по HTTPS поставьте nginx (или другой reverse proxy) перед бинарником или контейнером.

Готовый пример: `[docker/nginx.conf.example](docker/nginx.conf.example)`.

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

Если Бухгалтер в Docker с пробросом `8765:8765`, `proxy_pass` остаётся на `http://127.0.0.1:8765` (nginx на том же хосте).

В разделе **Настройки → Админка** укажите **внешний URL** — например `https://buhgalter.my-site.ru` — для ссылок в уведомлениях и **доступа из интернета** через reverse proxy.

## Бинарник с GitHub Releases

На странице [Releases](https://github.com/kai-zer-ru/buhgalter/releases) скачайте архив под свою ОС (Linux, Windows, macOS), распакуйте и запустите:

```bash
tar -xzf buhgalter_*_linux_amd64.tar.gz
./buhgalter
```

Проверьте контрольную сумму из `checksums.txt` в том же релизе.

## Первый запуск (/setup)

При первом запуске откроется `/setup`:

1. Имя и логин администратора
2. Пароль (дважды): минимум 8 символов, хотя бы одна буква и одна цифра, не совпадает с логином
3. При необходимости — восстановление БД из `.db` бэкапа прямо на `/setup` (до завершения первичной настройки)

Факт завершения установки хранится в `data/.configured` (вне SQLite). Восстановление бэкапа **не** сбрасывает этот маркер — повторный setup не откроется.

При сбое во время setup (например, обрыв после записи в БД) маркер синхронизируется с состоянием БД при следующем запросе; повторная отправка формы не приводит к внутренней ошибке — при уже выполненной настройке API вернёт «Настройка уже выполнена» (409).

Если приложение за reverse proxy — см. [Nginx](#nginx-reverse-proxy--https) и поле **внешний URL** в админке.

## Переменные окружения

Файл `.env` читается при старте (путь — `BUHGALTER_ENV_FILE`, по умолчанию `.env` в **текущем рабочем каталоге**). Приложение **не изменяет** `.env` — только пользователь. Уже заданные в shell переменные не перезаписываются.

**Где разместить `.env`:**

- **Бинарник** — положите `.env` в каталог, из которого запускаете `./buhgalter` (обычно рядом с бинарником). Если запускаете из другого места, укажите путь через `BUHGALTER_ENV_FILE` или переменную окружения shell
- **Docker Compose** — `cp build/release/.env.example .env` в каталоге с `docker-compose.yml`, отредактируйте

**Зачем нужен `.env`:**

- Задать `BUHGALTER_ALLOWED_HOSTS` для доступа с других устройств в LAN или с удалённого сервера (без reverse proxy)
- Переопределить порт и пути к данным/логам

**Пример `.env` для запуска на удалённом сервере (без reverse proxy):**

```env
# IP адреса/домены, с которых разрешён прямой доступ (localhost/127.0.0.1 доступны всегда)
BUHGALTER_ALLOWED_HOSTS=["192.168.1.100","example.com"]
```

| Переменная               | По умолчанию                   | Описание                                                                                                                                                            |
| ------------------------ | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `BUHGALTER_ADDR`         | `:8765`                        | Адрес и порт HTTP-сервера                                                                                                                                           |
| `BUHGALTER_DB_PATH`      | `./data/buhgalter.db`          | Путь к файлу SQLite                                                                                                                                                 |
| `BUHGALTER_DATA_DIR`     | `./data`                       | Каталог данных (БД, `.configured`, бэкапы)                                                                                                                          |
| `BUHGALTER_LOG_DIR`      | `./logs`                       | Каталог логов                                                                                                                                                       |
| `BUHGALTER_CORS_ORIGINS` | `*`                            | CORS: `*` — отражает Origin запроса (нужно для cookie-сессий; буквальный `*` с credentials запрещён браузером). Список через запятую — только перечисленные origins |
| `BUHGALTER_ENV_FILE`     | `.env`                         | Путь к файлу конфигурации `.env` (рядом с `docker-compose.yml` или в каталоге запуска бинарника)                                                                    |
| `BUHGALTER_ALLOWED_HOSTS` | `["127.0.0.1","localhost","::1"]` | Host для прямого доступа без reverse proxy (JSON-массив). **localhost** / `127.0.0.1` / `::1` разрешены всегда, в `.env` можно не указывать. Если запускаете на удалённом сервере и заходите по IP/домену — укажите их здесь |
| `BUHGALTER_STATIC_EMBED` | `true`                         | Встроенный фронтенд в бинарнике (`false` — отдельный Vite dev)                                                                                                      |

Миграции БД применяются автоматически при старте (goose).

## Уведомления MAX — сертификаты Минцифры (обязательно)

Официальный API MAX (`platform-api2.max.ru`) использует TLS-сертификаты **НУЦ Минцифры** (Russian Trusted CA). Без доверия к ним отправка уведомлений через провайдер **official** в MAX завершится ошибкой TLS (`x509: certificate signed by unknown authority`, `unable to get local issuer certificate (20)`).

**Важно:** одиночные файлы `russian_trusted_sub_ca.cer` с [gosuslugi.ru/crt](https://www.gosuslugi.ru/crt) — это **старый** выпускающий центр (2022). Сертификат `*.max.ru` подписан **новым** Sub CA (2024). Нужен файл `**russian_trusted_sub_ca_2024_pem.crt`** из архива Госуслуг.

Скачайте архивы (Linux, без авторизации):

```bash
mkdir -p ~/certs && cd ~/certs
curl -fsSLO https://gu-st.ru/content/lending/linux_russian_trusted_root_ca_pem.zip
curl -fsSLO https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.zip
unzip -o linux_russian_trusted_root_ca_pem.zip
unzip -o russian_trusted_sub_ca_pem.zip
# нужны: russian_trusted_root_ca_pem.crt и russian_trusted_sub_ca_2024_pem.crt
```

Портал со всеми вариантами для Windows/macOS/Android: **[gosuslugi.ru/crt](https://www.gosuslugi.ru/crt)**.

После установки перезапустите Бухгалтер и проверьте тестовую отправку в **Настройки → Уведомления → MAX**.

### Образ GHCR

В официальном образе `ghcr.io/kai-zer-ru/buhgalter` сертификаты уже добавлены в хранилище доверенных CA. **Дополнительных действий не требуется.**

### Linux

Используйте файлы из архивов выше: `russian_trusted_root_ca_pem.crt` и `**russian_trusted_sub_ca_2024_pem.crt`**.

**Debian, Ubuntu и производные:**

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt russian_trusted_sub_ca_2024_pem.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

**Fedora, RHEL, AlmaLinux, Rocky Linux:**

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt russian_trusted_sub_ca_2024_pem.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

**Arch Linux:**

На Arch **не** используйте путь `/usr/share/pki/ca-trust-source/anchors` — это Fedora/RHEL. Готового пакета в официальных репозиториях Arch нет.

```bash
cd ~/certs
sudo cp russian_trusted_root_ca_pem.crt /etc/ca-certificates/trust-source/anchors/russian_trusted_root_ca.crt
sudo cp russian_trusted_sub_ca_2024_pem.crt /etc/ca-certificates/trust-source/anchors/russian_trusted_sub_ca_2024.crt
sudo update-ca-trust
```

Если ранее ставили старый `russian_trusted_sub_ca.cer` — удалите его из `anchors/` (он не подходит для MAX) и оставьте **2024**-версию.

Проверьте наличие обоих файлов:

```bash
ls /etc/ca-certificates/trust-source/anchors/
```

**Проверка** (успех — любой HTTP-код **без** `curl: (60) SSL certificate...`; 404/401 от API — нормально):

```bash
curl -v https://platform-api2.max.ru/
```

### Windows

1. Скачайте архивы с [gosuslugi.ru/crt](https://www.gosuslugi.ru/crt) или распакуйте `russian_trusted_sub_ca_pem.zip` на Linux и перенесите `russian_trusted_sub_ca_2024_pem.crt` на Windows.
2. Установите **Russian Trusted Root CA** и **Russian Trusted Sub CA (2024)** — двойной щелчок по `.cer`/`.crt` → **Установить сертификат** → **Локальный компьютер** → **Поместить все сертификаты в следующее хранилище** → **Доверенные корневые центры сертификации**.
3. Перезапустите `buhgalter.exe`.

Через PowerShell **от имени администратора** (пути подставьте свои):

```powershell
certutil -addstore -f "ROOT" C:\certs\russian_trusted_root_ca_pem.crt
certutil -addstore -f "ROOT" C:\certs\russian_trusted_sub_ca_2024_pem.crt
```

### macOS

1. Скачайте сертификаты с [gosuslugi.ru/crt](https://www.gosuslugi.ru/crt) — нужны корневой и **Sub CA 2024** (из `russian_trusted_sub_ca_pem.zip`).
2. **Связка ключей** → **Системная** связка → импортируйте оба файла → для каждого: **Доверие** → **Всегда доверять**.
3. Перезапустите бинарник Бухгалтера.

Через терминал (пути подставьте свои):

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/certs/russian_trusted_root_ca_pem.crt
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/certs/russian_trusted_sub_ca_2024_pem.crt
```

**Проверка:**

```bash
curl -v https://platform-api2.max.ru/
```

### Секретный ключ уведомлений

Токены Telegram и MAX шифруются AES-256-GCM. Ключ задаётся в **Настройки → Админка** — сохраняется в `system_settings.notification_secret_key`, текущее значение не отображается. **Ровно 32 символа.**

## Бэкапы

В интерфейсе: **Настройки → Админка → Бэкапы**.

- Ручное создание и скачивание копии базы
- Автобэкап по расписанию (время и хранение настраиваются)
- Восстановление из файла `.db` (нужно подтверждение `RESTORE`)

Файлы бэкапов лежат в `data/backups/` (или в Docker-томе `buhgalter-backups`). Перед обновлением или рискованными операциями рекомендуется сделать бэкап.

## Обновление

Сделайте бэкап, замените бинарник/образ, перезапустите. Миграции применятся при старте.

**Бинарник / ручная установка:** остановите процесс, замените `buhgalter`, запустите снова. Сделайте бэкап до обновления.

**Docker (compose):**

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d
```

**Docker (`docker run`):** остановите контейнер, `docker pull ghcr.io/kai-zer-ru/buhgalter:latest`, запустите снова с теми же `-v` и `-p`.

## Документация

Справочники по установке, модели данных, UI-соглашениям и API — [docs/README.md](docs/README.md).

## API-документация

Документация OpenAPI доступна без авторизации — можно читать не запуская проект:

- **Демо-стенд:** [buhgalter-demo.kai-zer.ru/docs](https://buhgalter-demo.kai-zer.ru/docs)
- **OpenAPI YAML (демо):** [buhgalter-demo.kai-zer.ru/docs/openapi.yaml](https://buhgalter-demo.kai-zer.ru/docs/openapi.yaml)

После запуска своего сервера (порт **8765**):

- **Redoc:** [http://localhost:8765/docs](http://localhost:8765/docs)
- **OpenAPI YAML:** [http://localhost:8765/docs/openapi.yaml](http://localhost:8765/docs/openapi.yaml)

Исходник спецификации в репозитории: `[docs/api/openapi.yaml](docs/api/openapi.yaml)`.

В режиме разработки (`make dev-server` + `make dev-web`) те же URL доступны и через Vite: [http://localhost:5173/docs](http://localhost:5173/docs) (прокси на API).

## Разработка

```bash
# Терминал 1 — API без встроенного фронта
make dev-server

# Терминал 2 — фронтенд (http://localhost:5173)
make dev-web
```

```bash
make test         # go test + svelte-check + e2e (playwright)
make test-unit    # только unit/integration + svelte-check (без e2e)
make lint-go      # golangci-lint
make ci           # lint + все тесты + build + docker-build
make docker-build # сборка образа (тег buhgalter:local)
make clear     # очистка БД, логов, бэкапов и артефактов сборки
```

Если `make build` падает с `EACCES` на `web/build` или `server/internal/static/dist` — после Docker/act файлы могли остаться от root. Запустите `make fix-build-perms` или повторите `make build` (права чинятся автоматически).


| Команда                           | Описание                                             |
| --------------------------------- | ---------------------------------------------------- |
| `make download-bank-logos`        | Логотипы банков → `data/banks/`, `web/static/banks/` |
| `make download-marketplace-logos` | Логотипы маркетплейсов → `data/category_icons/`      |
| `make generate-category-icons`    | SVG иконок категорий                                 |
| `make sqlc`                       | Регенерация Go из `server/queries/`                  |


## Лицензия

[MIT License](LICENSE) — Copyright (c) 2026 kai-zer-ru