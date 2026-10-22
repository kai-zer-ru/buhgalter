# Release notes — v1.7.0

Краткая сводка для пользователей и администраторов. Полный список изменений — [CHANGELOG.md](../CHANGELOG.md).

---

> **ОБЯЗАТЕЛЬНО СДЕЛАЙТЕ БЕКАП!** Перед обновлением сервера сохраните копию `data/buhgalter.db` и каталога `backups/`. Новых миграций БД в этом релизе нет.

## Живые обновления на вебе

Открытая вкладка больше не ждёт ручного обновления и не долбит сервер лишними GET:

- постоянный канал **WebSocket** (`GET /api/v1/realtime`);
- после операции в другой вкладке, с телефона или автосписания на сервере — открытый экран обновляется сам;
- событие `invalidate` несёт **`hint_paths`** — перечитываются только затронутые блоки (не весь дашборд «на всякий случай»);
- при живом сокете фоновый SWR-revalidate выключен; если прокси режет Upgrade — тихий откат на SWR с cooldown 60 с (скрытая вкладка не revalidate);
- прогресс импорта приходит по сокету (poll раз в 1,2 с остаётся запасным вариантом);
- после обрыва и reconnect — догон изменений операций и мягкая перечитка экрана.

`import.done` применяется синхронно из WebSocket (до следующего кадра `invalidate`), чтобы экран сразу переходил к «Импорт завершён».

После создания или правки операции список счетов снова показывает актуальные балансы (раньше при живом сокете мог «залипнуть» на `0.00` из кеша справочника).

**Android** в v1.7.0 сокет не использует: по-прежнему SWR cooldown ~1 мин и лента синхронизации. REST API для CRUD не менялся.

Подробнее — [realtime-updates.md](../roadmap/realtime-updates.md), [ui-api-cache.md](ui-api-cache.md).

## Главная (веб)

Если у общего баланса нет ближайшего прогноза (нет предстоящих подписок и периодических в текущем месяце) — под суммой muted-текст «Нет ближайших подписок и периодических».

## Подписки

- Уже привязанная к подписке операция в меню строки больше не предлагает «Сделать подпиской» / «Прикрепить к подписке» (веб и Android).
- Привязка расхода к подписке закрывает ближайший слот очереди (`upcoming_run_ats`), если дата операции попадает в его окно: раннее списание банка больше не оставляет «ещё одно списание завтра». История вне окна только линкуется к подписке.

## Android

- Попап «Версии» в боковом меню больше не залипает на старой версии сервера после апгрейда: проверка версии всегда идёт в сеть.
- SMS банка в шторке «Сообщения» (заголовок вроде «Ваш Т-Банк») снова попадают в историю и черновики: разбор MessagingStyle, нормализация Unicode-дефисов в имени отправителя, расширенный список SMS-приложений.

Подробнее — [android-client-platform.md](android-client-platform.md), [notification-intercept.md](../roadmap/notification-intercept.md).

## Reverse proxy (nginx)

Если Бухгалтер за nginx (или другим прокси), для живого канала нужны заголовки Upgrade:

```nginx
location /api/v1/realtime {
    proxy_pass http://127.0.0.1:8765;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 3600s;
}
```

Без этого веб просто останется на SWR — учёт не сломается. Пример в репозитории: [docker/nginx.conf.example](../docker/nginx.conf.example), подробнее — [install/nginx.md](install/nginx.md).

## Обновление

**Сервер / Docker:** бэкап → замена бинарника или `docker compose pull && up -d`. Новых миграций БД нет. Живой канал и правка очереди подписок — сервер **v1.7.0**.

**Android:** установите новый APK поверх старого или из попапа «Версии» в боковом меню (фикс SMS и попапа версий — в APK). Сокет realtime только на вебе; CRUD по-прежнему через REST.

## Для разработчиков

- OpenAPI `1.7.0`: `GET /api/v1/realtime` (WebSocket handshake).
- Пакет `server/internal/realtime` (Hub, Publish после мутаций / import / scheduler).
- Клиент: `web/src/lib/realtime.ts`, полный `clearRefCache` на мутациях веба; счета из `ui/meta` не пишутся в SWR-ключи `/accounts*` ([ui-api-cache.md](ui-api-cache.md)).
- E2E: две вкладки без F5; сценарий create account → expense → balance.
- Документация: [realtime-updates.md](../roadmap/realtime-updates.md), [ui-api-cache.md](ui-api-cache.md), [install/nginx.md](install/nginx.md), [notification-intercept.md](../roadmap/notification-intercept.md), обновлённый [README.md](../README.md).
