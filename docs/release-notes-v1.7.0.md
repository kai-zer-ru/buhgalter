# Release notes — v1.7.0

Краткая сводка для пользователей и администраторов. Полный список изменений — [CHANGELOG.md](../CHANGELOG.md).

---

> **ОБЯЗАТЕЛЬНО СДЕЛАЙТЕ БЕКАП!** Перед обновлением сервера сохраните копию `data/buhgalter.db` и каталога `backups/`. Новых миграций БД в этом релизе нет.

## Живые обновления на вебе

Открытая вкладка больше не ждёт ручного обновления и не долбит сервер лишними GET:

- постоянный канал **WebSocket** (`GET /api/v1/realtime`);
- после операции в другой вкладке, с телефона или автосписания на сервере — открытый экран обновляется сам;
- при живом сокете фоновый SWR-revalidate выключен; если прокси режет Upgrade — тихий откат на SWR с cooldown 60 с;
- прогресс импорта приходит по сокету (poll раз в 1,2 с остаётся запасным вариантом);
- после обрыва и reconnect — догон изменений и мягкая перечитка экрана.

**Android** в v1.7.0 сокет не использует: по-прежнему SWR cooldown ~1 мин и лента синхронизации. REST API для CRUD не менялся.

Подробнее — [realtime-updates.md](../roadmap/realtime-updates.md), [ui-api-cache.md](ui-api-cache.md).

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

**Сервер / Docker:** бэкап → замена бинарника или `docker compose pull && up -d`. Новых миграций БД нет. Живой канал — сервер **v1.7.0**.

**Android:** новый APK не обязателен для этой фичи (сокет только на вебе). Версия приложения в репозитории поднята до **1.7.0** вместе с сервером.

## Для разработчиков

- OpenAPI `1.7.0`: `GET /api/v1/realtime` (WebSocket handshake).
- Пакет `server/internal/realtime` (Hub, Publish после мутаций / import / scheduler).
- Клиент: `web/src/lib/realtime.ts`; E2E — две вкладки без F5.
- Документация: [realtime-updates.md](../roadmap/realtime-updates.md), [ui-api-cache.md](ui-api-cache.md), [install/nginx.md](install/nginx.md), обновлённый [README.md](../README.md).
