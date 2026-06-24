# Nginx reverse proxy

Пример проксирования приложения за HTTPS.

---

## Пример конфигурации

```nginx
server {
    server_name buhgalter.my-site.ru;

    location /api/ {

        proxy_pass http://127.0.0.1:8765;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    listen 443 ssl;
    ssl_certificate /etc/ssl/fullchain.pem;
    ssl_certificate_key /etc/ssl/privkey.pem;
}
```

## external_url в админке

`external_url` задавать только при работе за reverse proxy, например:

`https://buhgalter.example.com`

Без reverse proxy поле оставить пустым.
