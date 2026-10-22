# Android UI (SvelteKit)

Отдельный фронтенд Android-клиента (`android/ui/`), не общий с `web/`. Сборка уходит в Capacitor (`make android-sync` / APK).

## Разработка

```bash
cd android/ui && npm install && npm run dev
```

Из корня: `make prepare-android`, `make test-unit` (включает `npm run check` + vitest). Playwright e2e для Android **не используется** — приёмка на устройстве вручную.

Документация: [docs/android-client.md](../docs/android-client.md), [UI](../docs/android-client-ui.md), [платформа](../docs/android-client-platform.md).
