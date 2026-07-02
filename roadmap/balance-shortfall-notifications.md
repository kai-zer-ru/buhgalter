# Уведомления: недостаток средств на балансе

Общая документация по уведомлениям (настройки, периоды, блокировка UI/API) — [docs/notifications.md](../docs/notifications.md).

Задача v1.3.0 — при напоминаниях об исходящих операциях проверять `accounts.current_balance` и при нехватке дописывать настраиваемый суффикс.

## Зачем

Пользователь получает напоминание о платеже по кредиту, плановом расходе или долге заранее — и сразу видит, что на счёте не хватает денег.

Пример:

> Платёж по кредиту «Ипотека»: 12 000 ₽. Дата: завтра. На балансе не хватает 1 200 ₽!

## Настройка в UI

Блок **«Настройки»** (бывший «Триггеры») на вкладке уведомлений — таблица **Название / Описание / Состояние** (toggle):

| Переключатель | API / БД | Шаблоны |
|---------------|----------|---------|
| Долги | `trigger_debt` | `debt_overdue`, `debt_due_soon` |
| Кредиты | `trigger_credit` | `credit_payment` |
| Плановые | `trigger_planned` | `planned_operation` |
| Отрицательный баланс | `trigger_negative_balance` | `balance_shortfall` (+ проверка баланса в worker) |
| Бюджет | `trigger_budget` | `budget_threshold` |
| Автопополнение отключено | `trigger_auto_topup_disabled` | `auto_topup_disabled` |
| Восстановление пароля | `trigger_password_reset` | `password_reset` (только админ) |
| Регистрация пользователя | `trigger_user_registration` | `user_registration` (только админ) |

Сохранение toggles — отдельная кнопка «Сохранить» в карточке; состояние после PUT берётся из **ответа API** (`applyNotificationSettings`), не из повторного GET.

### Блокировка шаблонов и периодов

**Каждый** переключатель в блоке «Настройки» управляет редактированием связанных шаблонов (см. таблицу выше) и **строк таблицы «Периоды и расписание»** (долги → четыре строки, кредиты → одна). Секция «Расписание» (`notification_time_local`) доступна, если включён хотя бы один из toggles Долги / Кредиты / Плановые. Подробнее — [docs/notifications.md](../docs/notifications.md).

Шаблон `test` не привязан к toggle и всегда доступен.

При **выключенном** переключателе:

- карточка шаблона: textarea и кнопки (placeholder, preview, reset, save) — `disabled`, визуально `opacity-50`;
- подсказка: «Включите «{название настройки}» в настройках, чтобы редактировать шаблон» (i18n: `settings.notifications.templates.disabled_setting`);
- backend: `PUT` / preview шаблона → **400** (`TemplateSettingEnabled` в [`types.go`](../server/internal/notify/types.go)).

При **включённом** — обычное редактирование.

Отдельно для **«Отрицательный баланс»**: выключение также **отключает проверку баланса** в worker — суффикс не дописывается, даже если денег не хватает.

Шаблон **«Отрицательный баланс»** (`balance_shortfall`):

- placeholder `{amount}` — сумма **недостающих** средств;
- дефолт RU: `На балансе не хватает {amount}!`

## Когда проверяется баланс

| Основной триггер | Условие | Счёт | Сумма |
|------------------|---------|------|-------|
| `credit_payment` | `trigger_negative_balance=1` | `credits.debit_account_id` | сумма платежа |
| `planned_operation` | `trigger_negative_balance=1`, `expense` / `transfer`, `affects_balance=1` | `transactions.account_id` | сумма операции |
| `debt_due_soon`, `debt_overdue` | `trigger_negative_balance=1`, `direction=borrowed`, есть счёт, `affects_balance=1` | счёт открывающей операции | остаток долга |

Не проверяется: `trigger_negative_balance=0`, доходы, долги «мне должны», операции без счёта, `affects_balance=0`.

**Формула:** `shortfall = max(0, сумма − current_balance)` (копейки). Прогноз (`forecast_balance`) не используется.

## Backend

| Компонент | Файл |
|-----------|------|
| Worker, отправка | [`worker.go`](../server/internal/notify/worker.go) |
| Склейка суффикса | [`balance.go`](../server/internal/notify/balance.go) — `formatWithBalanceShortfall` |
| Gating шаблонов | [`types.go`](../server/internal/notify/types.go) — `TemplateSettingEnabled` |
| PUT/preview reject | [`service.go`](../server/internal/notify/service.go) — `rejectTemplateWhenSettingDisabled` |

Dedup и `notification_log.trigger_type` — основной триггер (не `balance_shortfall`).

## Миграция

`032_notification_balance_shortfall.sql` — колонка `trigger_negative_balance`, расширение CHECK для `balance_shortfall`.

## Тест-план

- [x] credit: balance < payment, toggle on → суффикс в тексте
- [x] credit: toggle off → без суффикса
- [x] `TemplateSettingEnabled` для всех пар toggle ↔ шаблон
- [x] PUT `balance_shortfall` / `debt_overdue` при выключенной настройке → 400
- [x] PUT `trigger_negative_balance: false` → roundtrip в ответе PUT и GET
- [x] UI: таблица «Настройки»; шаблон заблокирован при выключенном toggle (любой)
- [x] UI: состояние toggle после сохранения из ответа PUT (не сбрасывается в `true`)
- [x] UI/API: строки «Периоды и расписание» и расписание заблокированы при выключенном toggle — [docs/notifications.md](../docs/notifications.md)
