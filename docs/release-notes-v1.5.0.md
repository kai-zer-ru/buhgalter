# Release notes — v1.5.0 (фрагмент: модули)

Краткая сводка. Полный список изменений релиза — по мере закрытия остальных пунктов v1.5.0 в [ROADMAP](../ROADMAP.md) и [CHANGELOG.md](../CHANGELOG.md).

---

> **ОБЯЗАТЕЛЬНО СДЕЛАЙТЕ БЕКАП!** Перед обновлением сервера сохраните копию `data/buhgalter.db` и каталога `backups/`.

## Модули в админке (feature toggles)

Администратор может включать и выключать опциональные разделы (долги, кредиты, бюджет, подписки, уведомления и др.) — **только в web-админке**, страница **Админка → Модули** (`/admin/features`). Выключенный модуль скрывается в web и Android; API отвечает `404` с кодом `FEATURE_DISABLED`; фоновые задачи модуля не выполняются. Данные в БД сохраняются.

Подробнее — [feature-toggles.md](feature-toggles.md).

### Breaking: регистрация

Поле `registration_enabled` **убрано** из `GET/PUT /api/v1/admin/settings`.

- Управление — флаг `registration` в `GET/PUT /api/v1/admin/features`.
- Публичное значение для страницы входа по-прежнему в `GET /setup/status` как `registration_enabled`.
- При миграции текущее значение из `system_settings` переносится в `feature_flags`.
