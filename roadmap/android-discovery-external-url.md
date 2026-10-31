# Android: внешний URL из найденного сервера

Баг **v1.11.0**. В профиль сервера не попадает внешний URL сервера, найденного в локальной сети. Его приходится вписывать вручную. Только клиент Android (`android/ui`), сервер менять не нужно.

---

## Как воспроизвести

1. В веб-админке задан внешний URL (`external_url`, например `https://buhgalter.example.ru`).
2. Android: экран адреса сервера (`/server-setup`) или **Настройки → Сервер** (`/settings/server`).
3. В списке найденных серверов у строки видна подпись «Внешний: buhgalter.example.ru».
4. Нажать на строку, сохранить, войти.
5. **Настройки → Сервер**: поле «Внешний URL (необязательно)» пустое.

**Ожидается:** внешний URL из найденного сервера записан в профиль (`remoteUrl`), и приложение работает вне домашней сети без ручного ввода.

---

## Причина

Сервер отдаёт адрес правильно: `GET /api/v1/health` возвращает `external_url`, а `probeLanOrigin` ([lan-discovery.ts](../android/ui/src/lib/platform/lan-discovery.ts)) кладёт его в `DiscoveredServer.externalUrl`. Оба способа поиска (mDNS и скан подсети) идут через этот probe, поэтому подпись в списке есть.

Дальше адрес теряется в трёх местах:

| Место | Что происходит |
|-------|----------------|
| [ServerDiscoveryPanel.svelte](../android/ui/src/lib/components/ServerDiscoveryPanel.svelte) | Callback `onselect: (origin: string) => void`. По тапу передаётся только `server.origin`, `externalUrl` выбрасывается |
| [server-setup/+page.svelte](../android/ui/src/routes/server-setup/+page.svelte) | `onselect` пишет только `lanUrl`. `currentProfile()` жёстко ставит `remoteUrl: ''`, а `saveProfile()` берёт `remoteUrl` из уже сохранённого профиля. При первом запуске там пусто |
| [settings/server/+page.svelte](../android/ui/src/routes/settings/server/+page.svelte) | `onselect` пишет только `lanUrl`, поле `remoteUrl` не трогает |

В [android-client-platform.md](../docs/android-client-platform.md) про первый запуск сказано: «Внешний URL и SSID — позже в `/settings/server`». Ручного поля внешнего URL на `/server-setup` нет и не нужно. Но если адрес уже известен из discovery, терять его незачем.

---

## Что поменять

1. `ServerDiscoveryPanel`: `onselect` передаёт весь `DiscoveredServer` (или `{ origin, externalUrl? }`), а не только строку origin.
2. `/server-setup`: при выборе сервера запомнить `externalUrl`. В `saveProfile()` взять `remoteUrl` из выбранного сервера, если он есть. Если нет, оставить `existing.remoteUrl`, как сейчас. Если после выбора из списка LAN URL исправили вручную на другой адрес, запомненный `externalUrl` сбросить.
3. `/settings/server`: при выборе сервера подставить `externalUrl` в поле `remoteUrl`, если поле пустое. Непустое поле молча не перезаписывать (см. открытые вопросы).
4. Адрес проходит обычную нормализацию `normalizeProfile()` ([server-profile.ts](../android/ui/src/lib/platform/server-profile.ts)). Непригодный origin отбрасывается, как при ручном вводе. `verifyServerProfile()` уже пингует `remoteUrl`. Если внешний адрес из локальной сети недоступен (нет hairpin NAT), сохранение не должно падать из-за него: LAN URL рабочий, а внешний нужен вне дома. Это поведение проверить и решить отдельно.
5. Обновить [android-client-platform.md](../docs/android-client-platform.md), раздел про профиль сервера и LAN discovery: внешний URL подставляется из найденного сервера.

Логику «какой `remoteUrl` взять» лучше вынести в чистую функцию в `server-profile.ts`, чтобы проверить её unit-тестом без Svelte.

---

## Открытые вопросы

- **Непустой `remoteUrl` в настройках.** Перезаписать, оставить или спросить, если адрес из discovery другой? Предложение: оставить и показать под полем подсказку «Сервер сообщает: …» с кнопкой «Подставить».
- **Ручной ввод LAN URL без discovery.** `verifyServerProfile()` и так ходит на `/health` этого адреса. Можно брать `external_url` из ответа и для ручного ввода. Это шире исходного бага, решить при реализации.
- **Hairpin NAT.** Если внешний адрес из домашней сети недоступен, `verifyServerProfile()` сейчас бросит ошибку на `remoteUrl`. Нужно ли при автоподстановке пропускать проверку внешнего адреса или показывать предупреждение вместо ошибки.

---

## Тесты

- Unit (`android/ui`): функция выбора `remoteUrl`. Случаи: есть `externalUrl` при пустом профиле; `externalUrl` нет; в профиле уже есть другой `remoteUrl`; невалидный `externalUrl`; LAN URL исправлен вручную после выбора.
- Ручная проверка на телефоне: первый запуск → выбрать сервер с внешним URL → **Настройки → Сервер**, поле заполнено; выйти из Wi‑Fi, приложение ходит на внешний адрес.
