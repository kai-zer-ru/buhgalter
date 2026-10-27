# Уведомления

Self-hosted напоминания в Telegram и MAX: настройки пользователя, расписание, шаблоны сообщений и worker на сервере.

API: `GET` / `PUT` `/api/v1/user/notifications`, preview и reset шаблонов — [openapi.yaml](api/openapi.yaml).

---

## Вкладка «Уведомления» (`/settings/notifications`)

Доступна после настройки ключа шифрования уведомлений (админка). Блоки сохраняются **отдельными кнопками «Сохранить»**; после PUT состояние берётся из **ответа API** (`applyNotificationSettings`), не из повторного GET. Блоки настроек (кроме предупреждения о ключе) свёрнуты в spoiler’ы (`CollapsibleSection`), по умолчанию закрыты.

| Блок | Содержимое |
|------|------------|
| Telegram / MAX | Включение канала, идентификаторы получателя, тестовая отправка (с кнопками, если клавиатура не пуста) |
| **Кнопки по умолчанию** | Global inline-клавиатуры Telegram и MAX (конструктор DnD; пресеты: долги / кредит / операции / бюджет / настройки). Попадают в каждый шаблон. Конструктор только у включённого канала; если включены оба — переключатель вкладками (`.tab`), не кнопками |
| **Настройки** | Переключатели типов событий (таблица на десктопе, карточки на мобильных) |
| **Периоды и расписание** | Числовые параметры напоминаний и время проверки очереди (таблица / карточки) |
| **Шаблоны сообщений** | Текст и per-template клавиатуры по каждому `trigger_type` (вкладки TG/MAX только если оба канала включены; иначе сразу конструктор активного; пресет кнопки соответствует типу шаблона). Глобальные кнопки канала уже в раскладке: их можно перетаскивать, нельзя удалить и нельзя править текст (текст — в «Кнопках по умолчанию»). Свои кнопки добавляются и удаляются. Preview (текст + кнопки), сброс |

На **Android** пункта «Уведомления» в списке настроек нет. Страница `/settings/notifications` по прямому адресу показывает только баннер «Настройте уведомления в веб-версии» (каналы, шаблоны и кнопки — только в web).

---

## Inline-кнопки (Telegram / MAX)

Исходящие уведомления могут содержать **inline-клавиатуру**. Две независимые модели: Telegram (`reply_markup`) и MAX (`attachments.inline_keyboard`; a161 — тот же JSON, другой base URL). Типы кнопок — по официальным докам провайдеров.

### Хранение и API

- Global: `keyboard_telegram` / `keyboard_max` в `notification_settings` (JSON `{"rows":[[{button}]]}`, default пустой).
- Per-template: те же поля у `notification_templates`.
- `GET` / `PUT` `/api/v1/user/notifications`; preview возвращает resolved text **и** resolved keyboards (tg/max). Сброс шаблона сбрасывает и его keyboard; global не трогает.

### Сборка при отправке

У кнопки в хранимом JSON есть стабильный `id`. В запрос Telegram/MAX он не попадает.

Если у шаблона нет `placement`, ряды склеиваются как раньше: **global → кнопки шаблона → balance_shortfall** (если shortfall дописан к сообщению).

Если `placement` есть, это ряды токенов `g:<id>` и `l:<id>`: глобальная кнопка стоит там, куда её перетащили в шаблоне. Удалить её из шаблона нельзя: id, которого нет в `placement`, дописывается в конец этой раскладки, перед `balance_shortfall`. Токен `g:` на уже удалённую глобальную кнопку исчезает. Сброс шаблона убирает свои кнопки и `placement`, global не трогает.

Плейсхолдеры в url/link (и аналогах) — как в тексте шаблонов, база только `system_settings.external_url`.

Если после резолва у кнопки типа url/link/web_app/open_app/login_url URL пуст — **эта кнопка отбрасывается** (T1), остальные остаются. Если клавиатура пуста — markup / attachment **не** передаётся.

Webhook, обработка callback и CAP-6 (`internal_url`) — **вне scope** (см. [telegram-max-buttons.md](../roadmap/telegram-max-buttons.md)).

---

## Блок «Настройки»

Таблица **Название / Описание / Состояние** (toggle). На мобильных (`md:hidden`) — карточки с тем же содержимым (как список токенов на вкладке «Токены»).

| Переключатель | API / БД | Шаблоны |
|---------------|----------|---------|
| Долги | `trigger_debt` | `debt_overdue`, `debt_due_soon` |
| Кредиты | `trigger_credit` | `credit_payment` |
| Подписки | `trigger_subscription` | `subscription_charge` |
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
| Подписки: заранее | `subscription_days_before` (0–30) | Подписки |

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

Плейсхолдеры: `{amount}` — недостающая сумма (`max(0, сумма − current_balance)`, копейки) **в валюте счёта**; `{account_url}` → `/accounts/{id}` счёта с недостачей. Событие одного счёта (баланс, автопополнение, кредит, подписка, плановая) форматирует сумму валютой этого счёта; бюджет — `budgets.currency`. Шаблоны не меняются, меняются значения плейсхолдеров.

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
| `balance_shortfall` | `{account_url}` | `/accounts/{id}` |
| `auto_topup_disabled` | `{account_url}` | `/accounts/{id}` |
| `subscription_charge` | `{subscription_url}` | `/subscriptions/{id}` |

Шаблон `balance_shortfall` — суффикс к основному сообщению; ссылка на счёт — `{account_url}`.

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
| Клавиатуры (модель, merge, T1) | [`keyboard.go`](../server/internal/notify/keyboard.go), [`keyboard_merge.go`](../server/internal/notify/keyboard_merge.go), [`keyboard_validate.go`](../server/internal/notify/keyboard_validate.go) |
| Gating шаблонов | `TemplateSettingEnabled` |
| Gating периодов | `PolicySettingEnabled` |
| Суффикс баланса | [`balance.go`](../server/internal/notify/balance.go) |
| Пороги бюджета | [`budgetnotify/check.go`](../server/internal/budgetnotify/check.go) |
| URL в шаблонах | [`urls.go`](../server/internal/notify/urls.go) |

Dedup: `notification_log` с тем же user/trigger/channel/entity/`dedup_date` и **`status = sent`**. Строка `error` повтор не блокирует (worker после рестарта, `Deliver`, сброс пароля, регистрация). Слот воркера на календарный день в памяти процесса — отдельно: до рестарта очередь в `notification_time_local` не крутится повторно. `trigger_type` — основной триггер (не `balance_shortfall`).

Worker `processPlanned` при включённом `trigger_planned` по-прежнему переводит due `kind=future` → `manual` в момент рассылки. Если toggle выключен — worker future не трогает; фоновый `FutureRunner` активирует всех (~1 мин, в т.ч. при старте).

---

## Модель данных

Таблицы `notification_settings`, `notification_templates`, `notification_log` — [data-model.md](data-model.md).

Миграция `032_notification_balance_shortfall.sql` — `trigger_negative_balance`, шаблон `balance_shortfall`.

Миграция `055_notification_keyboards.sql` — JSON `keyboard_telegram` / `keyboard_max` в settings и templates.

---

## UI (файлы)

| Файл | Назначение |
|------|------------|
| [`web/src/lib/settings/NotificationsTab.svelte`](../web/src/lib/settings/NotificationsTab.svelte) | Вкладка уведомлений (web), таблицы, gating, кнопки |
| [`web/src/lib/settings/KeyboardBuilder.svelte`](../web/src/lib/settings/KeyboardBuilder.svelte) | Конструктор inline-клавиатур (DnD) |
| [`android/ui/src/routes/settings/notifications/+page.svelte`](../android/ui/src/routes/settings/notifications/+page.svelte) | Android: баннер «настройте в веб» |
| [`web/src/lib/components/IntegerInput.svelte`](../web/src/lib/components/IntegerInput.svelte) | Целые числа без spinner |
| [`web/src/lib/components/MoneyInput.svelte`](../web/src/lib/components/MoneyInput.svelte) | Деньги (аналогичный текстовый ввод) |
