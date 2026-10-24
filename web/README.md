# Фронтенд (SvelteKit)

Веб-интерфейс Бухгалтера. Сборка встраивается в Go-бинарник (`make build`).

## Разработка

Из корня репозитория:

```bash
make dev-server   # API на :8765 без встроенного фронта
make dev-web      # Vite на http://localhost:5173 (прокси API)
```

Проверки кода (fmt, lint, типы): `make prepare` (или точечно `cd web && npm run check`). Тесты: `make test` (vitest и Playwright) или `make test-unit` без e2e.

Общие команды, структура проекта и документация UI — в [README.md](../README.md) и [docs/README.md](../docs/README.md).
