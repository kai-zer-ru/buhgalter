# Активация future-операций в фоне

**Версия:** v1.2.2  
**Статус:** реализовано — фоновый `FutureRunner` в `scheduler` (раз в минуту + при старте)

## Зачем

Раньше при list/dashboard/stats вызывался `ActivateDueFutureTransactions` — перевод наступивших `future`-операций в `manual` и пересчёт балансов на каждый read-запрос.

## Решение

- `FutureRunner` в `internal/scheduler` — активация раз в минуту (и сразу при старте сервера), по образцу credits/recurring.
- `ActivateAllDueFutureTransactions` — только пользователи с просроченными `future`.
- Убраны вызовы из `List`, `DashboardForUser`, `stats.Summary`.
- Допустимый лаг актуальности данных — до ~1 минуты.

## Связанные места в коде

- `server/internal/scheduler/` — `FutureRunner`, `runFutureActivation`
- `server/internal/transaction/` — `ActivateDueFutureTransactions`, `ActivateAllDueFutureTransactions`
- `server/internal/accountbalance/` — пересчёт балансов после активации
