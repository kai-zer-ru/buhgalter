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

Если Бухгалтер в Docker с пробросом `8765:8765`, `proxy_pass` остаётся на `http://127.0.0.1:8765` (nginx на том же хосте).

---

## external_url в админке

После настройки HTTPS укажите в **Настройки → Админка** поле **внешний URL**, например:

`https://buhgalter.example.com`

Оно используется для ссылок в уведомлениях и разрешения доступа через reverse proxy. Без reverse proxy поле оставьте пустым.
