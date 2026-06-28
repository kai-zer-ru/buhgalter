# Roadmap

Краткая сводка по версиям. Полный список изменений — [CHANGELOG.md](CHANGELOG.md).

## Содержание

- [v1.0.0](#v100)
- [v1.1.0](#v110)
- [v1.1.1](#v111)
- [v1.2.0](#v120)
- [v1.2.1](#v121)
- [v1.2.2](#v122)
- [v1.3.0](#v130)
- [Общие планы](#общие-планы)

## v1.0.0

Первый стабильный релиз self-hosted приложения для личного учёта финансов.

- [x] Установка: `make build`, Docker (GHCR), бинарники GitHub Releases; `/setup`, миграции SQLite (`001`…`019`)
- [x] Пользователи: регистрация, сессии, API-токены, админка (пользователи, бэкапы, диагностика), audit log
- [x] Счета, категории, операции (доход / расход / перевод, `future`), долги, кредиты с графиком и автосписанием
- [x] Импорт CSV/XLSX (Cubux), экспорт CSV, статистика, контекстные сводки
- [x] Уведомления Telegram и MAX; шаблоны и расписание
- [x] UI: SvelteKit, ru/en, светлая/тёмная тема, PWA; REST API + OpenAPI, Redoc на `/docs`
- [x] Документация: [docs/](docs/) — установка, модель данных, UI-гайды

## v1.1.0

- [x] [Сброс и восстановление пароля](roadmap/password-reset.md) — запрос на странице входа, уведомление админу, сброс в админке
- [x] [Адаптация под мобильные браузеры](roadmap/mobile-web.md) — responsive UI, карточки, touch-targets
- [x] [Списание ретро-платежей](roadmap/credit-retroactive-debit.md) — `retroactive_debit_count` при создании кредита
- [x] Кредиты: редактирование сумм в графике (`PATCH /credits/{id}/schedule`), восстановление укороченных графиков
- [x] Переводы: комиссия, системная категория «Комиссия»; фильтры по дате в TZ пользователя
- [x] UI: `ToggleSwitch`, фильтр категорий по типу, редактирование перевода, сводка счёта с переводами
- [x] [Release notes](docs/release-notes-v1.1.md) · [демо-стенд](https://buhgalter-demo.kai-zer.ru/)

## v1.1.1

- [x] Поддержка `.env`; `BUHGALTER_ALLOWED_HOSTS` для доступа без reverse proxy
- [x] OpenAPI (`/docs`, `/docs/openapi.yaml`) без авторизации
- [x] «Счета» в верхнем меню; страница `/accounts` — список счетов
- [x] Мелкие правки UI: курсор на кнопках, подсветка активного раздела меню

## v1.2.0

- [x] [Ипотека (MVP)](roadmap/mortgage.md) — тип `mortgage`, взнос, расчёт суммы кредита
- [x] Автосписание кредитов по `debit_time_local`; банк кредита в API/UI
- [x] Периодические операции — страница `/recurring-operations`
- [x] Восстановление БД из бэкапа на `/setup`; «сделать основным» в списке счетов
- [x] UI: хлебные крошки, прогноз баланса на главной, админка в «Настройки → Админка»
- [x] Триггер уведомлений `password_reset` для администраторов

## v1.2.1

- [x] Docker: bind mounts `./data`, `./backups`, `./logs`; entrypoint uid 1000; [миграция](docs/install/docker.md#docker-bind-mount-migration)
- [x] `BUHGALTER_LOG_MODE`; поле `error.field` в JSON-ошибках API
- [x] Ипотека: ежедневные проценты, `POST /credits/schedule/preview`, ручной платёж из договора
- [x] UI: `RowActionsMenu`, редактирование операций, `FilterPanel` на мобильных, кеш справочников
- [x] [Release notes](docs/release-notes-v1.2.1.md) · [ui-row-actions](docs/ui-row-actions.md) · [ui-stats](docs/ui-stats.md) · [transactions-display](docs/transactions-display.md)

## v1.2.2

- [x] Проверка версии раз в сутки — попап для админов о бэкапе и release notes ([release notes](docs/release-notes-v1.2.2.md))
- [x] `/stats` → «По категориям»: отдельные секции доходов и расходов ([ui-stats](docs/ui-stats.md))
- [x] Кнопки-иконки Доход / Расход / Перевод вместо общего «Операция» ([transactions-display](docs/transactions-display.md), [ui-row-actions](docs/ui-row-actions.md))
- [x] Разобраться с долгой загрузкой страниц — денормализация баланса, batch SQL, индексы, кеш middleware, `GET /api/v1/ui/meta` ([release notes](docs/release-notes-v1.2.2.md))
- [x] полностью покрыть код тестами e2e
- [ ] [Активация future-операций в фоне](roadmap/activate-future-transactions.md) — убрать `ActivateDueFutureTransactions` из hot-path каждого запроса
- [x] отключить постоянные запросы api/v1/admin/password-reset-requests
- [x] Убрать поле "Разница", дублирует баланс
- [x] Страница счёта, список операций (например яндекс). При клике в операции перевода на счёт, с которого был перевод в данный счёт - не переходит на страницу другого счёта. Но при клике на счёт, на который был перевод с текущего счёта - переходит на другой счёт верно

## v1.2.4


## v1.3.0
- [ ] Добавляем статус пользователя. Админ (первый пользователь) - естественно активен. При создании пользователей из админки - польззователь активен. При самостоятельной регистрации - статус "на проверке". В админке можно менять статусы (через меню ...). А именно: 
    - Активного пользователя можно забанить
    - Забаненного пользователя можно разблокировать
    - Пользователя "На модерации" можно активировать и заблокировать

## Общие планы

- [ ] [Командная работа](roadmap/team-collaboration.md) — несколько пользователей на одной базе
- [ ] [PostgreSQL](roadmap/postgresql.md) — опциональная БД вместо SQLite
- [ ] [Webhook](roadmap/webhooks.md) — исходящие события для внешних интеграций
- [ ] [Синхронизация с банками](roadmap/bank-sync.md) — автоматический импорт операций
- [ ] [Сканер чеков](roadmap/receipt-scanner.md) — расход из фото или QR чека
- [ ] [Home Assistant](roadmap/home-assistant.md) — интеграция для умного дома
- [ ] [Кредитная карта](roadmap/credit-card.md) — тип счёта (лимит, выписка, погашение)
- [ ] [Кнопки в Telegram / MAX](roadmap/telegram-max-buttons.md) — inline-кнопки в исходящих уведомлениях
- [ ] [Эволюция уведомлений](roadmap/notifications-evolution.md) — развитие политик/шаблонов/частоты
- [ ] [Android-клиент](roadmap/android-client.md) — нативное приложение
