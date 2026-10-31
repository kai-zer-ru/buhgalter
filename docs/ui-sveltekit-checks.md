# UI — SvelteKit typecheck и vitest

Типичные поломки `svelte-check` / Android `vitest` в `web/` и `android/ui/`. Цель — не повторять одни и те же ошибки при обёртке навигации в `resolve()` и при правках `$lib/*`.

`make prepare` гоняет проверки кода, включая `svelte-check` в `web` и `android/ui`. `make test` — тесты: Go, vitest web, vitest android, Playwright web. `make test-unit` — то же без e2e.

---

## Typed routes: `resolve()` и динамические пути

SvelteKit типизирует `resolve` / `goto`: аргумент — литерал маршрута (или `Pathname`), **не** произвольный `string`. Обёртка `resolve(helperReturningString())` массово ломает `npm run check`.

### Нельзя

```ts
void goto(resolve(transactionNewPath({ type: 'expense' }))); // helper → string
await goto(resolve(`${base}?${qs}`)); // шаблон из ResolvedPathname → широкий union
```

Cast в полный `Pathname` (union всех путей) тоже **не работает**: generic `resolve` не сужается по union.

### Android (`android/ui`)

| Хелпер | Файл | Назначение |
|--------|------|------------|
| `*Path` / `parseFormReturnPath` | `$lib/android/form-routes.ts` | Строят URL; возвращают `AppHref` (`'/'` как один литерал Pathname) |
| `resolveAppPath(path)` | `$lib/android/form-nav.ts` | Динамический путь → `resolve(path as '/')` |
| `gotoReplace(path)` | `$lib/android/form-nav.ts` | `goto(resolve(…), { replaceState: true })` — **не** оборачивать снова в `resolve`; для мастера кредита |
| `leaveForm(returnTo)` | `$lib/android/form-nav.ts` | После save/cancel обычной формы (открытой push): `history.back()`, иначе fallback `gotoReplace` — **не** `gotoReplace(returnTo)` после push (дубль в history) |

ESLint `svelte/no-navigation-without-resolve` требует, чтобы **`resolve(` был прямо в аргументе `goto`/`href`**. Обёртка вроде `goto(resolveAppPath(x))` правило не видит — либо вызывать `resolve` на месте (`gotoReplace`), либо `eslint-disable-next-line` с пояснением.

Предпочтительные вызовы:

```ts
void goto(resolve(transactionNewPath({ type: 'expense', from: '/' })));
await goto(resolveAppPath(`/transactions?${qs}`));
await leaveForm(returnTo); // обычные формы; мастер кредита — gotoReplace
```

Литералы без динамики — обычный `resolve('/accounts')`.

### Web (`web/`)

Динамические сегменты и `returnTo` — `as Pathname` (или один литерал), как в `BackLink` / `accounts/new`:

```ts
await goto(resolve(returnTo as Pathname));
href={resolve(item.path as Pathname)}
```

---

## Toast

Тип: `'success' | 'error' | 'info' | 'warning'`. Значения вроде `'danger'` **нет** (это variant у `RowActionsMenu`).

Toast (web и Android): в vitest (environment `node`) использовать `globalThis.setTimeout`, не `window.setTimeout`.

Подробнее — [ui-toast.md](ui-toast.md).

---

## Vitest (android/ui)

### Мок `svelte/store`

Если тест мокает `svelte/store` только ради `get` для i18n, нужно сохранять остальной API (`writable` и т.д.): транзитивные импорты (`session-expired` ← `api/client`) иначе падают на «No writable export».

```ts
vi.mock('svelte/store', async (importOriginal) => {
	const actual = await importOriginal<typeof import('svelte/store')>();
	return {
		...actual,
		get: () => (key: string) => translations[key] ?? key
	};
});
```

### `ApiError`

Конструктор: `(code, message, status)`, не наоборот.

### Календарь

`calendarCells` всегда отдаёт **42** ячейки (6 недель). При изменении паддинга — обновить ожидания в `datetime-picker.test.ts` (web и android).

### Новые поля в нормализаторах

Пример: `normalizeProfile` → `trustedOrigins`. Обновлять deep-equal в `*.test.ts` в той же сессии.

---

## Svelte 5: `$derived` / `$state`

Не ссылаться в `$derived` на переменную, объявленную ниже в том же `<script>` — ошибка «used before its declaration» (как `accounts` на `/accounts`).

Порядок: сначала `$state` / `$derived.by`, затем производные `$derived`, которые их читают.

---

## Необязательные параметры в `.svelte`: падает SSR, `check` молчит

Компилятор Svelte вырезает типы TypeScript и превращает необязательный параметр в невалидный JS: `(id?: string) =>` → `(id?) =>`, `{#snippet tile(label: string, color?: string)}` → `function tile($$renderer, label, color?)`. `svelte-check` и линт проходят, а страница в dev и на сервере отдаёт **500** (`Expected , or ) but found ?`).

- В стрелочных функциях и сниппетах внутри `.svelte` писать явный тип: `id: string | null | undefined`, `color: string | undefined` — и передавать аргумент явно (`undefined`).
- Проверка: открыть страницу в dev или отрендерить компонент через `vite.ssrLoadModule` + `render` из `svelte/server`.

## Secure storage / WebCrypto (android)

- В APK: `@aparajita/capacitor-secure-storage`. В browser / e2e: **не** вызывать плагин — memory + `localStorage` (`buhgalter.secure.*`), иначе bootstrap зависает на «Загрузка…».
- `SecureStorage.get` → `DataType`; в native-ветке `secureGet` возвращать строку только при `typeof value === 'string'`.
- `crypto.subtle.deriveBits` / `importKey`: `Uint8Array` с `ArrayBufferLike` часто не подходит к `BufferSource` — передавать срез `ArrayBuffer` (`buffer.slice(byteOffset, …)`).

---

## Чеклист перед сдачей UI-задачи

- [ ] Динамические `href` / `goto` — через `form-routes` / `resolveAppPath` (android) или `as Pathname` (web); без `resolve(plainString)`.
- [ ] Toast — только `success` | `error` | `info` | `warning`.
- [ ] Изменён `$lib/*` → обновлён рядом стоящий `*.test.ts` (ожидания, моки, новые поля).
- [ ] Мок `svelte/store` — через `importOriginal`, если модуль тянет `writable`.
- [ ] `$derived` не ссылается на переменные ниже по файлу.
