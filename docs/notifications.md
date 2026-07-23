# Уведомления

Self-hosted напоминания в Telegram и MAX: настройки пользователя, расписание, шаблоны сообщений и worker на сервере.

API: `GET` / `PUT` `/api/v1/user/notifications`, preview и reset шаблонов — [openapi.yaml](api/openapi.yaml).

---

## Вкладка «Уведомления» (`/settings/notifications`)

Доступна после настройки ключа шифрования уведомлений (админка). Блоки сохраняются **отдельными кнопками «Сохранить»**; после PUT состояние берётся из **ответа API** (`applyNotificationSettings`), не из повторного GET.

| Блок | Содержимое |
|------|------------|
| Telegram / MAX | Включение канала, идентификаторы получателя, тестовая отправка |
| **Настройки** | Переключатели типов событий (таблица на десктопе, карточки на мобильных) |
| **Периоды и расписание** | Числовые параметры напоминаний и время проверки очереди (таблица / карточки) |
| **Шаблоны сообщений** | Текст по каждому `trigger_type`, preview, сброс |

---

## Блок «Настройки»

Таблица **Название / Описание / Состояние** (toggle). На мобильных (`md:hidden`) — карточки с тем же содержимым (как список токенов на вкладке «Токены»).

| Переключатель | API / БД | Шаблоны |
|---------------|----------|---------|
| Долги | `trigger_debt` | `debt_overdue`, `debt_due_soon` |
| Кредиты | `trigger_credit` | `credit_payment` |
| Плановые | `trigger_planned` | `planned_operation` |
| Отрицательный баланс | `trigger_negative_balance` | `balance_shortfall` (+ проверка баланса в worker) |
| Бюджет | `trigger_budget` | `budget_threshold` |
| Автопополнение отключено | `trigger_auto_topup_disabled` | `auto_topup_disabled` |
| Восстановление пароля | `trigger_password_reset` | `password_reset` (только админ) |
| Регистрация пользователя | `trigger_user_registration` | `user_registration` (только админ, если регистрация включена) |

Шаблон `test` не привязан к toggle и всегда доступен.

---

## Блок «Периоды и расписание»

Та же таблица **Название / Описание / Состояние**; на мобильных — карточки. В колонке «Состояние» — целочисленные поля (`IntegerInput`: текст, только цифры, без spinner).

| Строка | API | Связанный toggle |
|--------|-----|------------------|
| Я должен: до срока | `debt_days_before` (0–30) | Долги |
| Я должен: просрочка | `my_debt_overdue_days_limit` (0–365) | Долги |
| Мне должны: задержка | `owed_debt_overdue_start_after_days` (0–365) | Долги |
| Мне должны: лимит | `owed_debt_overdue_days_limit` (0–365) | Долги |
| Кредиты: заранее | `credit_days_before` (0–30) | Кредиты |

Под таблицей — **Расписание**: `notification_time_local` (локальное время `HH:MM`, timezone пользователя). Редактируется, если включён хотя бы один из toggles **Долги**, **Кредиты** или **Плановые**.

---

## Блокировка при выключенном toggle

Один toggle управляет связанными **шаблонами** и **строками периодов** (см. таблицы выше).

### Шаблоны

При **выключенном** переключателе:

- карточка шаблона: textarea и кнопки — `disabled`, `opacity-50`;
- подсказка: «Включите «{название}» в настройках, чтобы редактировать шаблон»;
- backend: `PUT` / preview шаблона → **400** (`TemplateSettingEnabled` в [`types.go`](../server/internal/notify/types.go)).

### Периоды и расписание

При **выключенном** переключателе:

- строки таблицы с соответствующим toggle — `opacity-50`, поле ввода `disabled`, подсказка в колонке «Описание» (тот же текст, что у шаблонов);
- секция «Расписание» — `disabled`, если выключены **Долги**, **Кредиты** и **Плановые**;
- backend: `PUT` с изменением связанного поля → **400** (`PolicySettingEnabled` в [`types.go`](../server/internal/notify/types.go)).

Отдельно для **«Отрицательный баланс»**: выключение отключает проверку баланса в worker — суффикс не дописывается. Подробнее — [balance-shortfall-notifications.md](../roadmap/balance-shortfall-notifications.md).

---

## Недостаток средств на балансе

При включённом **«Отрицательный баланс»** worker дописывает шаблон `balance_shortfall` к исходящим напоминаниям (кредит, плановый расход/перевод, долг «я должен»), если `current_balance` меньше суммы операции.

Placeholder `{amount}` — недостающая сумма. Формула: `max(0, сумма − current_balance)` (копейки).

---

## Бюджет

При включённом **«Бюджет»** worker и hook после расходной операции отправляют `budget_threshold`, когда фактические расходы по лимиту достигают `alert_at_percent` или 100%. Дедупликация — `budget_alert_sent`. Подробнее — [budget.md](budget.md).

Плейсхолдеры: `{name}`, `{spent}`, `{planned}`, `{percent}`, `{budget_url}` → `/budget`.

---

## Ссылки в шаблонах

Базовый URL — `system_settings.external_url` (админка → «Внешний URL»). При пустом значении вместо ссылки подставляется подсказка: «Нет внешней ссылки — настройте внешний URL в админке» (EN: *No external link — configure the external URL in admin settings.*).

| Шаблон | Плейсхолдер | Путь |
|--------|-------------|------|
| `debt_overdue`, `debt_due_soon` | `{debt_url}` | `/debts` |
| `credit_payment` | `{credit_url}` | `/credits/{credit_id}` |
| `planned_operation` | `{transaction_url}` | `/transactions` |
| `test` | `{settings_url}` | `/settings/notifications` |
| `password_reset` | `{reset_url}` | `/admin/users?reset={user_id}` |
| `user_registration` | `{moderation_url}` | `/admin/users?moderate={user_id}` |
| `budget_threshold` | `{budget_url}` | `/budget` |

Шаблон `balance_shortfall` — только `{amount}` (суффикс к основному сообщению, без отдельной ссылки).

### Долги: направление и срок

Шаблон `debt_due_soon` общий для «я должен» и «мне должны»; в текст подставляются:

| Плейсхолдер | Значение |
|-------------|----------|
| `{action}` | по `direction`: `borrowed` → «вернуть долг» / `repay debt to`; `lent` → «получить долг от» / `collect debt from` |
| `{when}` | относительная формулировка срока: «сегодня» / «завтра» / «через N дн.» (EN: today / tomorrow / in N days) |
| `{days}` | число дней до срока (для кастомных шаблонов; в дефолте не используется) |

Дефолт: `Напоминание: {action} {debtor} — {amount}. (Срок: {due_date}, {when})`. Если шаблон ранее сохраняли вручную со старым текстом («вернуть долг», «через {days} дн.») — сбросьте к дефолту или поправьте сами.

Дефолтные шаблоны включают URL на отдельной строке. Плейсхолдеры доступны в UI (кнопки вставки) и в preview API.

Реализация: [`urls.go`](../server/internal/notify/urls.go), подстановка в worker — [`worker.go`](../server/internal/notify/worker.go).

---

## Backend

| Компонент | Файл |
|-----------|------|
| Настройки, PUT/GET | [`service.go`](../server/internal/notify/service.go) |
| Worker, отправка | [`worker.go`](../server/internal/notify/worker.go) |
| Gating шаблонов | `TemplateSettingEnabled` |
| Gating периодов | `PolicySettingEnabled` |
| Суффикс баланса | [`balance.go`](../server/internal/notify/balance.go) |
| Пороги бюджета | [`budgetnotify/check.go`](../server/internal/budgetnotify/check.go) |
| URL в шаблонах | [`urls.go`](../server/internal/notify/urls.go) |

Dedup и `notification_log.trigger_type` — основной триггер (не `balance_shortfall`).

---

## Модель данных

Таблицы `notification_settings`, `notification_templates`, `notification_log` — [data-model.md](data-model.md).

Миграция `032_notification_balance_shortfall.sql` — `trigger_negative_balance`, шаблон `balance_shortfall`.

---

## UI (файлы)

| Файл | Назначение |
|------|------------|
| [`web/src/routes/settings/+page.svelte`](../web/src/routes/settings/+page.svelte) | Вкладка уведомлений, таблицы, gating |
| [`web/src/lib/components/IntegerInput.svelte`](../web/src/lib/components/IntegerInput.svelte) | Целые числа без spinner |
| [`web/src/lib/components/MoneyInput.svelte`](../web/src/lib/components/MoneyInput.svelte) | Деньги (аналогичный текстовый ввод) |
