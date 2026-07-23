# Android-клиент — UI

Интерфейс приложения: отдельный SvelteKit-проект в `android/ui/`, не общий с веб-версией.

## Оболочка

Файлы: `android/ui/src/lib/android/`.

- **AndroidShell.svelte** — фиксированная оболочка на весь экран: шапка, скроллируемый `main`, полоска соединения внизу
- **Боковое меню (drawer)** — выезжает **слева**, ширина ~70% экрана (65–75%)
- **OfflineSyncBanner** — полоска статуса очереди офлайн-операций под шапкой
- **ConnectionStatusBar** — «Нет соединения» закреплена внизу; контент не заходит под неё

Веб-версия (`web/`) эту оболочку не использует.

## Кнопка «Назад» (Android)

`back-handler.ts` + `@capacitor/app`: drawer → модалки / confirm → `history.back()` → на корне (`/`) двойное нажатие с подтверждением → выход.

## Формы операций и переводов

Полноэкранные маршруты вместо `ModalShell`:

| Маршрут | Назначение |
|---------|------------|
| `/transactions/new?type=expense\|income` | Новая операция |
| `/transactions/[id]/edit` | Редактирование |
| `/transfers/new` | Перевод |
| `/transfers/[groupId]/edit` | Редактирование перевода |
| `/debts/new?direction=lent\|borrowed` | Дать в долг / взять в долг |
| `/debts/[id]/settle` | Погашение долга |
| `/accounts/[id]/charge-fee` | Комиссия по кредитной карте |
| `/accounts/[id]/auto-topup` | Автопополнение счёта |
| `/credits/[id]/pay` | Платёж по кредиту |
| `/credits/[id]/complete` | Досрочное закрытие |
| `/credits/[id]/change-account` | Смена счёта списания |
| `/credits/[id]/debit-time` | Время автосписания |
| `/credits/[id]/change-name` | Переименование |
| `/credits/[id]/change-bank` | Смена банка |

Шапка формы — через `shell-header.ts` в `AndroidShell`.

Формы долгов «дать в долг» / «взять в долг» — полноэкранные маршруты (`FormPageShell`), кнопки на списке — в одну строку (`.btn-pair-row`).

### Сохранение и возврат

| Контекст | После «Сохранить» |
|----------|-------------------|
| **Настройки** (`/settings/*`) | Toast, остаётесь на том же разделе |
| **Админка** (`/admin/*`) | Toast, остаётесь на том же разделе |
| **Формы «Добавить» и «Редактировать»** (операция, перевод, долг, счёт; действия по кредиту и т.п.) | `leaveForm(returnTo)` (`form-nav.ts`): **`history.back()`**, чтобы снять слот формы. Не `gotoReplace(returnTo)` после push — иначе в стеке два одинаковых `returnTo` и первое «Назад» выглядит как no-op |
| **Мастер кредита** (`/credits/new/*`) | Шаги и финал — `gotoReplace` (один слот мастера) |
| **Отмена / «Назад» в шапке** (`FormPageShell`) | `onback` или `leaveForm(backHref)` / `history.back()` |

Ссылки на формы передают `from` (напр. «Новый счёт» с `/accounts` → `/accounts/new?from=%2Faccounts`).

Хелперы URL — `$lib/android/form-routes.ts` (`AppHref`); динамический `goto`/`resolve` — `resolveAppPath` / `gotoReplace` / `leaveForm` в `form-nav.ts`. Не оборачивать произвольный `string` в typed `resolve` — см. [ui-sveltekit-checks.md](ui-sveltekit-checks.md).

Исключение: **отключение сервера** (Настройки → Сервер) — переход на `/server-setup`, не обычное сохранение.

### Баланс счёта после мутаций

На карточке счёта (`/accounts/[id]`) после сохранения операции `dataRefreshTick` перезагружает **счёт + баланс + список** (не только операции). Успешный `POST`/`PUT`/`PATCH`/`DELETE` в Android `client.ts` сбрасывает ref-cache (`clearRefCache` + `invalidateApiCache`), как в веб-клиенте — иначе баланс оставался бы от pre-mutation snapshot до remount.

### Селект счетов в формах

В списках счетов форм (`accountSelectOptions`, `transferAccountOptions`) **основной** счёт (`is_primary`) — **первым**; остальной порядок как в API (`ORDER BY name`). Хелпер `sortAccountsForSelect` в `$lib/accounts.ts`.

На главной и `/accounts` блок «Мои средства» — `groupAccountsByType`: сначала `cash`, затем `bank`, внутри типа основной первым. Кредитную карту нельзя сделать основной (`canSetAsPrimary`).

### Создание кредита (пошаговый мастер)

Полноэкранные шаги (`FormPageShell`, draft в `$lib/credits/create-draft.ts`):

| Шаг | Маршрут | Содержание |
|-----|---------|------------|
| Основное | `/credits/new/basics` | тип, имя, сумма / ипотека, дата, срок, ставка, интервал |
| Параметры | `/credits/new/options` | счёт, сумма платежа, первый платёж сегодня, ретроучёт, банк, автосписание / время |
| График | `/credits/new/schedule` | preview / ручные строки, ретро-списания → `POST /credits` |

Вход: `/credits` → «Новый кредит» → `/credits/new?from=…` (редирект на `basics` через `gotoReplace`). Переходы между шагами и отмена — тоже `replaceState`, чтобы в истории оставался **один** слот мастера; «Назад» в шапке формы листает шаги через `prevCreditCreateStep`. После сохранения — карточка кредита с `replaceState` (назад → список, без мастера).

### Только веб-версия

В мобильном клиенте **нет**:

- **Управления пользователями** (`/admin/users`) — создание, модерация, сброс пароля, блокировка; в APK доступны системные настройки, бэкапы и диагностика.

Офлайн outbox / SWR ref-cache — только Android; в веб не переносится.

## Навигация (drawer)

Прямые ссылки в боковом меню (без drill-down подменю); **«Главная»** — первый пункт. **Настройки** и **Админка** ведут на хаб-страницы со списком разделов (`SectionNavHub.svelte`).

| Раздел | Маршруты |
|--------|----------|
| Главная | `/` |
| Счета | `/accounts`, `/accounts/new`, `/accounts/[id]` — редактирование: поле «Начальный баланс» из `initial_balance`, не `balance_display` ([data-model.md](data-model.md)) |
| Операции | `/transactions`, `/transactions/new`, `/transactions/[id]/edit`, `/transfers/new`, `/transfers/[groupId]/edit` |
| Долги | `/debts`, `/debts/[id]/settle`, `/debtors/[id]` |
| Кредиты | `/credits`, `/credits/new/*` (мастер), `/credits/[id]` и подмаршруты действий. Деталь кредита **не** слушает `refCacheTick` (ресурс не в SWR): иначе чтение `credit` в `$effect` после каждого `load()` давало бесконечный «Обновляем данные…». Обновление — при открытии маршрута и явном `load()` после правок на странице |
| Бюджет | `/budget` |
| Статистика | `/stats` |
| Настройки | `/settings` — хаб с переходами в разделы |
| Админка | `/admin` — хаб (только для `is_admin`) |
| Выход | очистка токена → `/login` |

Формы операций, переводов, долгов, счетов, создания кредита и действий по кредиту — полноэкранные маршруты (`FormPageShell`). Нижние кнопки (`footer`) всегда прилипают к низу viewport: скроллится только середина формы (`form-page-scroll`), а `android-shell-main` в режиме формы (`android-shell-main-flush`) не прокручивается.

Остаются попапами: `UpdateAvailableModal`, `ConfirmDialog`, показ созданного API-токена, `CategoryIconPicker`.

Скроллбары в UI скрыты (контент прокручивается, полоса не занимает ширину).

### Настройки (`/settings`)

Хаб-страница (`SectionNavHub`) со списком разделов и chevron. Крошки: на хабе `Главная → Настройки`; на подстранице `Главная → Настройки (ссылка) → Раздел`. Профиль перенесён с `/settings` на `/settings/profile`.

| Раздел | Маршрут |
|--------|---------|
| Профиль | `/settings/profile` — тема: **Как на устройстве** (`system`, по умолчанию), светлая, тёмная; `system` резолвится через `prefers-color-scheme` |
| Пароль | `/settings/password` |
| Безопасность | `/settings/security` |
| Сервер | `/settings/server` |
| API-токены | `/settings/tokens` |
| Уведомления | `/settings/notifications` |
| Категории | `/settings/categories` |
| Импорт / экспорт | `/settings/import` |
| Периодические операции | `/settings/recurring-operations` |

### Админка (`/admin`)

Хаб-страница (только для `is_admin`; иначе редирект на `/`). Системные настройки перенесены с `/admin` на `/admin/system`. Крошки — по той же схеме, что у настроек.

| Раздел | Маршрут |
|--------|---------|
| Система | `/admin/system` |
| Бэкапы | `/admin/backups` |
| Диагностика | `/admin/diagnostics` |

Управление пользователями (`/admin/users`) — только в веб-версии.

### Footer drawer

- **Синхронизация** — статус очереди и кнопка «Синхронизировать» (активна и офлайн: принудительный reconnect); drawer **остаётся открытым** во время sync
- **Выйти** слева, **версия APK** справа (`v1.4.0` из `package.json` при сборке)
- Клик по версии **закрывает drawer** и открывает попап «Версии»
- Попап: версия сервера, версия приложения, ссылка «Скачать APK» (GitHub release серверной версии), предупреждение при рассинхроне версий
- Значок **!** у версии в drawer — когда версия приложения **старше** `current_version` с `GET /api/v1/version/check` (`app < server`)
- При офлайне в попапе версия сервера «Недоступно», ссылка на APK скрыта
- При `app < server` клиент подтягивает UI-строки с `GET /api/v1/ui/i18n/{lang}` (см. [android-client-platform.md](android-client-platform.md#локализация-android))
- **«Поделиться»** из другого приложения (текст / изображение) открывает `/transactions/new?type=expense` с префиллом описания

## Блокировка приложения

- Экран PIN (`AppLockScreen`) — после входа, до основного UI
- Настройка: `/settings/security` — включение, смена PIN, биометрия, **таймаут** в фоне (30 с…1 час)
- По умолчанию: повторная блокировка через 1 мин в фоне
- **Выход** сбрасывает PIN и биометрию — новый вход требует настройки блокировки заново (см. [android-client-platform.md](android-client-platform.md))

## Публичные экраны (без drawer)

| Экран | Маршрут | Назначение |
|-------|---------|------------|
| Адрес сервера | `/server-setup` | Первый запуск: LAN discovery + ручной **LAN URL** |
| Выбор входа | `/login` | Кнопки «Логин / пароль» и «API-токен»; сменить сервер |
| Логин / пароль | `/login/password` | `POST /auth/login` → session Bearer; назад → `/login` |
| API-токен | `/login/token` | Вставка Bearer API-токена; назад → `/login` |
| Сервер (расширенно) | `/settings/server` | Внешний URL, домашние Wi‑Fi, LAN fallback, HTTPS TOFU, отладочное логирование, экспорт outbox |

Пока URL сервера не задан, приложение удерживает пользователя на `/server-setup`.

## Поиск сервера в LAN

Компонент `ServerDiscoveryPanel.svelte` на `/server-setup` и `/settings/server`.

1. Требуется **Wi‑Fi** (не мобильная сеть).
2. Параллельно: **mDNS** (`_buhgalter._tcp`, ~4 с, plugin `LanDiscovery`) и **скан /24** на порту API (8765).
3. Каждый кандидат проверяется через `GET /api/v1/health`.
4. В списке: LAN-адрес (`192.168.x.x:8765`), версия, статус БД; если сервер вернул `external_url` (внешний URL из админки) — строка «Внешний: domain» (reverse DNS по IP **не** используется).
5. Тап по строке подставляет **LAN URL** в форму (не HTTPS-домен).

Код: `lan-discovery.ts`, `mdns-discovery.ts`, `LanDiscoveryPlugin.java`.

## Главная

- Сводка по счетам (включая кредитные карты), карточка долгов, виджет бюджета месяца — как в веб-дашборде
- Последние операции (ручные и плановые)
- Создание дохода / расхода / перевода; из меню операции — повтор и «сделать периодической» (`/settings/recurring-operations?from_tx=`)

## Ошибка загрузки страницы

Если **первая загрузка данных** экрана не удалась, пользователь видит карточку с текстом ошибки и кнопкой **«Повторить»** — не пустой экран и не toast вместо контента.

### Компоненты и хелперы

| Файл | Назначение |
|------|------------|
| `PageLoadGate.svelte` | Обёртка: загрузка → ошибка (с retry) → слот с контентом |
| `page-load.ts` | `capturePageLoadError()`, `reportPageLoadFailure()` |
| `EmptyStateCard.svelte` | Карточка для loading/error/пустых списков (используется внутри gate) |

Ключ i18n: `common.loadFailed` — запасной текст, если нет сообщения от API.

### Правила

1. **Только начальная загрузка страницы** — `PageLoadGate` / `loadError` для данных при открытии экрана или смене id. Состояния кнопок («Сохранить…», `saving`) и отправка форм — по-прежнему через toast.
2. **Пустой список ≠ ошибка** — `EmptyStateCard` с `*.empty` показывается только после успешной загрузки, когда записей нет. При сбое загрузки — `PageLoadGate` с `loadError`, а не текст «список пуст».
3. **Фоновое обновление (SWR)** — если данные уже на экране (кеш ref-cache), сбой фонового запроса даёт **toast**, экран не заменяется ошибкой (`reportPageLoadFailure({ background: true, hasData: true })`).
4. **Первая загрузка без кеша** — inline-ошибка в `PageLoadGate`, toast **не** дублируется.
5. **Шапка, крошки, фильтры** — остаются видимыми; gate оборачивает только тело списка / формы (как пустые состояния в [ui-empty-states.md](ui-empty-states.md)).
6. **Повторить** — `onretry` вызывает ту же `load()` / `loadAll()`, что и при mount.

### Пример

```svelte
<script lang="ts">
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let items = $state<Item[]>([]);

	async function load(opts: { background?: boolean } = {}) {
		if (!opts.background) loading = true;
		try {
			items = await fetchItems();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: opts.background,
				hasData: items.length > 0
			});
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if items.length === 0}
		<EmptyStateCard message={$_('items.empty')} />
	{:else}
		<!-- список -->
	{/if}
</PageLoadGate>
```

Проп `inline` — для loading текстом `<p>` (как на списках); без него — карточка «Загрузка…».

### Где применено

Все маршруты и вкладки с загрузкой данных при открытии: главная, счета, операции, кредиты, долги, бюджет, статистика, детали сущностей, полноэкранные формы (`FormPageShell`), настройки (профиль, API-токены, счета, категории, уведомления, периодические операции), админка (диагностика, бэкапы, система), `TransactionContextStats`, создание счёта.

Исключения (свои экраны ошибок, не `PageLoadGate`):

- **`+layout.svelte`** — bootstrap: нет URL → `/server-setup`; есть токен → PIN сразу (профиль из `last_user` / ref-cache / stub; probe в фоне); без токена и сервер недоступен → «Сервер недоступен» с «Повторить»
- **`/server-setup`** — inline-строка под полем (без toast-каскада)
- **Ошибки действий** (сохранить, удалить, sync outbox) — toast ([ui-toast.md](ui-toast.md))

## Select / Combobox / DateTimePicker

Те же компоненты, что в веб-UI: без `usePortal` список якорится через `top/bottom: 100%` на обёртку **только поля** (label и hint — снаружи `.relative`). Иначе на узком экране панель открывается вверх и «отрывается» от контрола (раньше — профиль: тема, часовой пояс). Общие правила — [ui-dialogs.md](ui-dialogs.md).

## Стили и тема оформления

Общая тема и компоненты (`layout.css`, кнопки, карточки) скопированы из веб-UI на этапе выделения проекта; дальше живут только в `android/ui/`.

Стили drawer: `android/ui/src/lib/android/android-shell.css`.

**Тема UI** (`users.theme` / `localStorage`): `light` | `dark` | `system`. Значение `system` («Как на устройстве») — default для новых пользователей и до логина; на DOM применяется resolved light/dark через `prefers-color-scheme` (слушатель `change`). То же на web. Лаунчер/splash night — отдельно, см. [android-client-platform.md](android-client-platform.md) (не зависит от настройки SPA).

## Иконка лаунчера

Генерация: цветная `android/ui/static/icon-512.png` + mono `icon-monochrome-512.png` → `make android-icons`.  
Adaptive: фон + foreground + **`<monochrome>`** (`drawable/ic_launcher_monochrome`) для Material You / тематических иконок ОС.  
Тёмный фон night (`values-night/`) и splash — системный night mode; на HyperOS/Xiaomi смена night bg home icon не гарантирована (см. [android-client-platform.md](android-client-platform.md)). SPA-тема лаунчер не меняет.

## См. также

- [android-client-platform.md](android-client-platform.md) — авторизация, офлайн, сборка
- [android/README.md](../android/README.md) — команды `make android-apk`, `make android-install`
