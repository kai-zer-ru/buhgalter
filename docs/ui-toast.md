# UI — in-app toast

Кратковременная обратная связь в веб-интерфейсе: успех, ошибка, предупреждение, информация. Toast появляется **внизу справа** (на узком экране — по центру снизу).

**Не путать** с [уведомлениями в мессенджере](notifications.md) (Telegram, MAX) — это отдельная подсистема на сервере и вкладка «Уведомления» в настройках.

---

## API

Модуль: `web/src/lib/toast.ts`. Рендер: `ToastContainer` в корневом layout.

| Метод | Тип | Когда использовать |
|-------|-----|-------------------|
| `toast.success(msg)` | success | Сохранено, удалено, действие выполнено |
| `toast.error(msg)` | error | Ошибка API, валидация при submit |
| `toast.warning(msg)` | warning | Предупреждение перед/после действия |
| `toast.info(msg)` | info | Нейтральная информация |
| `toast.fromError(err, fallbackKey?)` | error | `catch` с `ApiError` — через `formatApiError` (для `CONFLICT` и др. общих кодов — текст из `error.message` API) |
| `toast(msg, type?, durationMs?)` | любой | Обратная совместимость |

Длительность по умолчанию: success/info — 3,2 с; error/warning — 4,5 с.

---

## Когда toast, когда нет

| Ситуация | Паттерн |
|----------|---------|
| Результат действия (save, delete, load fail) | **toast** |
| Ошибка/успех в модалке после submit | **toast** |
| Постоянный баннер (сброс пароля админу) | inline / баннер |
| Модалка обновления приложения | модалка |
| Статический текст риска (perpetual token) | текст у поля |
| Подсказка у поля до submit (конфликт направления долга, расчёт графика) | inline у поля |
| Доставка в Telegram/MAX | сервер `internal/notify`, не toast |

---

## Доступность

- Контейнер: `aria-live="polite"`
- error/warning: `role="alert"`
- success/info: `role="status"`

Цвета: `--primary` (success), `--danger` (error), `--warning` (warning), `--text` (info).
