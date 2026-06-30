# Фронтенд (SvelteKit)

Веб-интерфейс Бухгалтера. Сборка встраивается в Go-бинарник (`make build`).

## Разработка

Из корня репозитория:

```bash
make dev-server   # API на :8765 без встроенного фронта
make dev-web      # Vite на http://localhost:5173 (прокси API)
```

Проверка типов: `cd web && npm run check`. E2E: `make test-e2e` (из корня).

Общие команды, структура проекта и документация UI — в [README.md](../README.md) и [docs/README.md](../docs/README.md).
