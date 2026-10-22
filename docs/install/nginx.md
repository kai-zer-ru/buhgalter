# Nginx reverse proxy

Пример проксирования приложения за HTTPS. Бухгалтер отдаёт и API, и веб-интерфейс с одного порта — отдельный прокси на Vite **не нужен**.

Готовый файл в репозитории: [docker/nginx.conf.example](../../docker/nginx.conf.example).

---

## Пример конфигурации

```nginx
server {
    listen 443 ssl;
    server_name buhgalter.my-site.ru;

    ssl_certificate     /etc/ssl/fullchain.pem;
    ssl_certificate_key /etc/ssl/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8765;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

`Host $host` обязателен: ExternalAccess всегда берёт hostname из `Host` (и для `external_url`, и для `BUHGALTER_ALLOWED_HOSTS`). `X-Forwarded-Host` не учитывается — раньше при заданном `external_url` он доверялся. Без `Host` nginx подставит upstream (`127.0.0.1`); с прокси на loopback запрос может пройти как localhost. Иначе **403** `ERR_EXTERNAL_ACCESS_DENIED`.

Если Бухгалтер в Docker с пробросом `8765:8765`, `proxy_pass` остаётся на `http://127.0.0.1:8765` (nginx на том же хосте).

---

## WebSocket (`/api/v1/realtime`)

Веб держит постоянное соединение для инвалидации кеша. Без `Upgrade` клиент тихо откатывается на SWR с cooldown.

```nginx
location /api/v1/realtime {
    proxy_pass http://127.0.0.1:8765;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;
}
```

Остальной трафик — как в примере выше (`location /`).

---

## external_url в админке

После настройки HTTPS укажите в **Настройки → Админка** поле **внешний URL**, например:

`https://buhgalter.example.com`

Оно используется для ссылок в уведомлениях (в том числе прямой ссылки на сброс пароля в Telegram/MAX) и разрешения доступа через reverse proxy. Hostname в `Host` запроса должен совпадать с hostname этого URL. Без reverse proxy поле оставьте пустым — в уведомлении о сбросе пароля админ увидит подсказку настроить внешний URL.
