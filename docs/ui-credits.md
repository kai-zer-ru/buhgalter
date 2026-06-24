# UI — кредиты

Единые правила badge-меток и спойлеров графика платежей на экранах `/credits` и `/credits/[id]`.

Связанные документы: [data-model.md](data-model.md) (сущности кредитов), [ui-dialogs.md](ui-dialogs.md) (модалки), [ui-stable-layout.md](ui-stable-layout.md) (вкладки списка), [ui-navigation.md](ui-navigation.md) (кликабельные ссылки).

---

## Badge-метки

### Стили

Классы в `web/src/routes/layout.css` (`@layer components`):

| Класс | Назначение |
|-------|------------|
| `.badge` | Нейтральная метка: фон от `--border`, текст `--text-muted`, рамка, `text-xs`, `rounded-lg` |
| `.badge-success` | Акцентная метка (завершённый кредит): оттенок `--primary` |

Метки — `inline-flex` с `gap` между соседними в flex-контейнере; **не** сливать текст подряд без обёртки.

### Где показываются

| Метка | i18n | Условие | Страница |
|-------|------|---------|----------|
| Рассрочка | `credits.badge.installment` | `interest_rate === 0` | список, детальная |
| Добавлен в учёт после выдачи | `credits.badge.retroactive` | `added_retroactively` | список, детальная |
| Завершённый | `credits.badge.closed` | `status === 'closed'` | только детальная (`.badge-success`) |

В форме создания (`CreditForm.svelte`) в колонке «Статус» графика — отдельная метка **«Учтён при добавлении»** (`credits.payment.status.retroactive`), не путать с badge на карточке кредита.

### Заголовок детальной страницы

Под `h1` — **отдельный flex-ряд** badge-меток (`flex flex-wrap items-center gap-2`).

Если `added_retroactively`, ниже — **две строки** метаданных (не одна строка через «·»):

| Поле | i18n |
|------|------|
| Дата выдачи | `credits.field.issueDate` |
| Добавлен в учёт | `credits.field.recordedAt` |

---

## Список `/credits`

### Вкладки

| Вкладка | i18n | Фильтр API |
|---------|------|------------|
| Активные | `credits.tab.active` | `status=active` |
| Завершённые | `credits.tab.closed` | `status=closed` |

Не использовать «Архив» — закрытый кредит **завершён**, не архивирован.

Кнопка **«+ Новый кредит»** — в шапке на обеих вкладках ([ui-stable-layout.md](ui-stable-layout.md)).

Под названием кредита в таблице — те же badge, что на детальной (без «Завершённый»).

---

## График платежей — спойлеры

**Файл:** `web/src/routes/credits/[id]/+page.svelte`

График не одной таблицей, а **тремя спойлерами** (`<details>`) внутри карточки с заголовком «График платежей» (`credits.schedule.title`).

### Группы

| Спойлер | i18n | Критерий (backend) | По умолчанию |
|---------|------|-------------------|--------------|
| Неоплаченные | `credits.schedule.group.pending` | `!is_applied`, не `retroactive` | **развёрнут** (`open`) |
| Оплаченные | `credits.schedule.group.applied` | `is_applied`, не `retroactive` | свёрнут |
| Учтённые при добавлении | `credits.schedule.group.retroactive` | `kind === 'retroactive'` | свёрнут |

- Пустые группы **не рендерятся**
- В `<summary>` — название группы и счётчик `(N)` (`tabular-nums`, `--text-muted`)
- Внутри каждого спойлера — та же таблица: дата, сумма, статус, «Удалить платёж» (если активный кредит)

### Статусы в колонке «Статус»

| Состояние | i18n |
|-----------|------|
| Ожидает | `credits.payment.status.pending` |
| Будущий (операция `kind=future`) | `credits.payment.status.future` |
| Списан | `credits.payment.status.applied` |
| Учтён при добавлении | `credits.payment.status.retroactive` |

### Разметка спойлера

```svelte
<details open class="border-b" style:border-color="var(--border)">
  <summary
    class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
  >
    {$_('credits.schedule.group.pending')}
    <span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">({count})</span>
  </summary>
  <!-- таблица платежей группы -->
</details>
```

---

## Чеклист для новых экранов кредитов

1. Badge — только классы `.badge` / `.badge-success`, не голый текст подряд.
2. Новый тип метки — добавить i18n `credits.badge.*` и стиль (или переиспользовать `.badge`).
3. Длинный график — только через спойлеры по статусу оплаты, неоплаченные развёрнуты.
4. Destructive-действия — [ui-dialogs.md](ui-dialogs.md) (`$lib/confirm`).
